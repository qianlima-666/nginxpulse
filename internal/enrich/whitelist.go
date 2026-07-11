package enrich

import (
	"bytes"
	"net"
	"strings"

	"github.com/qianlima-666/nginxpulse/internal/config"
)

type WhitelistMatch struct {
	RuleType  string
	RuleValue string
	Domestic  string
	Global    string
}

type WhitelistMatcher struct {
	enabled       bool
	ipMap         map[string]string
	cidrs         []whitelistCIDR
	ranges        []whitelistRange
	cities        []whitelistCity
	nonMainland   bool
	locationRules bool
}

type whitelistCIDR struct {
	net   *net.IPNet
	label string
}

type whitelistRange struct {
	start net.IP
	end   net.IP
	label string
}

type whitelistCity struct {
	value      string
	normalized string
}

func NewWhitelistMatcher(cfg *config.WhitelistConfig) *WhitelistMatcher {
	if cfg == nil {
		return nil
	}
	matcher := &WhitelistMatcher{
		enabled:     cfg.Enabled,
		ipMap:       make(map[string]string),
		nonMainland: cfg.NonMainland,
	}
	for _, raw := range cfg.IPs {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "/") {
			if _, cidr, err := net.ParseCIDR(trimmed); err == nil && cidr != nil {
				matcher.cidrs = append(matcher.cidrs, whitelistCIDR{net: cidr, label: trimmed})
			}
			continue
		}
		if strings.Contains(trimmed, "-") {
			if start, end, ok := parseIPRange(trimmed); ok {
				matcher.ranges = append(matcher.ranges, whitelistRange{start: start, end: end, label: trimmed})
			}
			continue
		}
		if parsed := net.ParseIP(trimmed); parsed != nil {
			normalized := parsed.String()
			matcher.ipMap[normalized] = normalized
		}
	}
	for _, raw := range cfg.Cities {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		normalized := normalizeCityRule(value)
		if normalized == "" {
			continue
		}
		matcher.cities = append(matcher.cities, whitelistCity{
			value:      value,
			normalized: normalized,
		})
	}
	matcher.locationRules = len(matcher.cities) > 0 || matcher.nonMainland
	return matcher
}

func (m *WhitelistMatcher) Enabled() bool {
	if m == nil {
		return false
	}
	return m.enabled
}

func (m *WhitelistMatcher) Match(ip string) (WhitelistMatch, bool) {
	if m == nil || !m.enabled {
		return WhitelistMatch{}, false
	}
	normalized := normalizeIPForWhitelist(ip)
	if normalized == "" {
		return WhitelistMatch{}, false
	}
	if label, ok := m.ipMap[normalized]; ok {
		return WhitelistMatch{RuleType: "ip", RuleValue: label}, true
	}
	parsed := net.ParseIP(normalized)
	if parsed != nil {
		for _, rule := range m.cidrs {
			if rule.net.Contains(parsed) {
				return WhitelistMatch{RuleType: "cidr", RuleValue: rule.label}, true
			}
		}
		for _, rule := range m.ranges {
			if ipInRange(parsed, rule.start, rule.end) {
				return WhitelistMatch{RuleType: "range", RuleValue: rule.label}, true
			}
		}
	}

	if !m.locationRules {
		return WhitelistMatch{}, false
	}
	domestic, global, ok := getIPLocationLocalOnly(normalized)
	if !ok {
		return WhitelistMatch{}, false
	}
	if len(m.cities) > 0 {
		normalizedDomestic := normalizeLocationMatch(domestic)
		for _, city := range m.cities {
			if city.normalized != "" && strings.Contains(normalizedDomestic, city.normalized) {
				return WhitelistMatch{
					RuleType:  "city",
					RuleValue: city.value,
					Domestic:  domestic,
					Global:    global,
				}, true
			}
		}
	}
	if m.nonMainland && isNonMainland(domestic, global) {
		return WhitelistMatch{
			RuleType:  "non_mainland",
			RuleValue: "非大陆",
			Domestic:  domestic,
			Global:    global,
		}, true
	}
	return WhitelistMatch{}, false
}

func normalizeIPForWhitelist(raw string) string {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return ""
	}
	if strings.Contains(candidate, ",") {
		parts := strings.Split(candidate, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				candidate = part
				break
			}
		}
	}
	if strings.HasPrefix(candidate, "[") {
		if idx := strings.Index(candidate, "]"); idx != -1 {
			host := candidate[1:idx]
			if host != "" {
				candidate = host
			}
		}
	}
	if host, _, err := net.SplitHostPort(candidate); err == nil {
		candidate = host
	} else if strings.Count(candidate, ":") == 1 && strings.Contains(candidate, ".") {
		host := strings.SplitN(candidate, ":", 2)[0]
		if host != "" {
			candidate = host
		}
	}
	if parsed := net.ParseIP(candidate); parsed != nil {
		return parsed.String()
	}
	return strings.TrimSpace(candidate)
}

func parseIPRange(value string) (net.IP, net.IP, bool) {
	parts := strings.SplitN(value, "-", 2)
	if len(parts) != 2 {
		return nil, nil, false
	}
	start := net.ParseIP(strings.TrimSpace(parts[0]))
	end := net.ParseIP(strings.TrimSpace(parts[1]))
	if start == nil || end == nil {
		return nil, nil, false
	}
	if (start.To4() == nil) != (end.To4() == nil) {
		return nil, nil, false
	}
	if compareIP(start, end) > 0 {
		return nil, nil, false
	}
	return start, end, true
}

func ipInRange(ip, start, end net.IP) bool {
	if ip == nil || start == nil || end == nil {
		return false
	}
	if (ip.To4() == nil) != (start.To4() == nil) {
		return false
	}
	return compareIP(ip, start) >= 0 && compareIP(ip, end) <= 0
}

func compareIP(a, b net.IP) int {
	aa := a.To16()
	bb := b.To16()
	if aa == nil || bb == nil {
		return 0
	}
	return bytes.Compare(aa, bb)
}

func normalizeCityRule(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return normalizeLocationMatch(trimmed)
}

func normalizeLocationMatch(value string) string {
	clean := strings.TrimSpace(value)
	if clean == "" || clean == "未知" {
		return ""
	}
	clean = strings.ReplaceAll(clean, "·", "")
	clean = strings.ReplaceAll(clean, " ", "")
	clean = strings.ReplaceAll(clean, "-", "")
	clean = strings.ToLower(clean)
	clean = trimLocationSuffix(clean)
	return clean
}

func trimLocationSuffix(value string) string {
	suffixes := []string{
		"特别行政区",
		"自治区",
		"维吾尔自治区",
		"回族自治区",
		"壮族自治区",
		"省",
		"市",
		"地区",
		"盟",
		"州",
		"县",
		"区",
	}
	for _, suffix := range suffixes {
		if strings.HasSuffix(value, strings.ToLower(suffix)) {
			return strings.TrimSuffix(value, strings.ToLower(suffix))
		}
	}
	return value
}

func isNonMainland(domestic, global string) bool {
	global = strings.TrimSpace(global)
	if global == "" || global == "未知" {
		return false
	}
	if !isChinaGlobal(global) {
		return true
	}
	domestic = strings.TrimSpace(domestic)
	if domestic == "" || domestic == "未知" {
		return false
	}
	keywords := []string{
		"香港",
		"澳门",
		"台湾",
		"台北",
		"台中",
		"台南",
		"高雄",
		"新北",
		"桃园",
		"基隆",
		"嘉义",
		"新竹",
		"屏东",
	}
	for _, key := range keywords {
		if strings.Contains(domestic, key) {
			return true
		}
	}
	return false
}

func isChinaGlobal(value string) bool {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return false
	}
	if normalized == "中国" {
		return true
	}
	normalized = strings.ToLower(normalized)
	return normalized == "china" || normalized == "cn"
}
