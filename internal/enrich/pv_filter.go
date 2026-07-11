package enrich

import (
	"net"
	"regexp"
	"strings"

	"github.com/qianlima-666/nginxpulse/internal/config"
)

var (
	// 全局过滤规则编译后的正则表达式
	excludePatterns []*regexp.Regexp
	excludeIPs      map[string]bool
	statusCodes     map[int]bool
	excludePrivate  bool
)

// InitPVFilters 初始化PV过滤规则
func InitPVFilters() {
	cfg := config.ReadConfig()

	// 初始化状态码过滤
	statusCodes = make(map[int]bool)
	for _, code := range cfg.PVFilter.StatusCodeInclude {
		statusCodes[code] = true
	}

	// 初始化正则表达式过滤
	excludePatterns = make([]*regexp.Regexp, len(cfg.PVFilter.ExcludePatterns))
	for i, pattern := range cfg.PVFilter.ExcludePatterns {
		excludePatterns[i] = regexp.MustCompile(pattern)
	}

	// 初始化IP过滤
	excludeIPs = make(map[string]bool)
	for _, ip := range cfg.PVFilter.ExcludeIPs {
		normalized := normalizeIP(ip)
		if normalized == "" {
			continue
		}
		excludeIPs[normalized] = true
	}

	excludePrivate = true
	if cfg.PVFilter.ExcludeIPs != nil && len(cfg.PVFilter.ExcludeIPs) == 0 {
		excludePrivate = false
	}
}

// normalizeIP extracts a usable IP string from log tokens
// (handles X-Forwarded-For lists and host:port forms).
func normalizeIP(raw string) string {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return ""
	}

	if strings.Contains(candidate, ",") {
		parts := strings.Split(candidate, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			candidate = part
			break
		}
	}

	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return ""
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

	return candidate
}

// ShouldCountAsPageView 判断是否符合 PV 过滤条件
func ShouldCountAsPageView(statusCode int, path string, ip string) int {
	// 检查状态码
	if !statusCodes[statusCode] {
		return 0
	}

	normalizedIP := normalizeIP(ip)

	// 过滤内网/保留地址
	if excludePrivate && isPrivateIP(net.ParseIP(normalizedIP)) {
		return 0
	}

	// 检查排除 IP 列表
	if normalizedIP != "" && excludeIPs[normalizedIP] {
		return 0
	}

	// 检查是否匹配全局排除模式
	for _, pattern := range excludePatterns {
		if pattern.MatchString(path) {
			return 0
		}
	}

	return 1
}
