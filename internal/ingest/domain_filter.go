package ingest

import (
	"net"
	"net/url"
	"strings"

	"github.com/qianlima-666/nginxpulse/internal/config"
)

type domainMatcher struct {
	domains []string
}

func newWebsiteDomainMatcher(websiteID string) domainMatcher {
	website, ok := config.GetWebsiteByID(websiteID)
	if !ok {
		return domainMatcher{}
	}
	return newDomainMatcher(website.Domains)
}

func newDomainMatcher(domains []string) domainMatcher {
	matcher := domainMatcher{domains: make([]string, 0, len(domains))}
	for _, raw := range domains {
		domain := normalizeDomainForLogMatch(raw)
		if domain == "" {
			continue
		}
		matcher.domains = append(matcher.domains, domain)
	}
	return matcher
}

func (m domainMatcher) includesHost(host string) bool {
	if len(m.domains) == 0 {
		return true
	}
	normalizedHost := normalizeDomainForLogMatch(host)
	if normalizedHost == "" {
		return true
	}
	for _, domain := range m.domains {
		if normalizedHost == domain || strings.HasSuffix(normalizedHost, "."+domain) {
			return true
		}
	}
	return false
}

func normalizeDomainForLogMatch(raw string) string {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" || value == "-" {
		return ""
	}
	value = strings.TrimPrefix(value, "*.")
	if strings.Contains(value, "://") {
		if parsed, err := url.Parse(value); err == nil && parsed.Host != "" {
			value = parsed.Host
		}
	}
	if strings.Contains(value, "@") {
		if parsed, err := url.Parse("//" + value); err == nil && parsed.Host != "" {
			value = parsed.Host
		}
	}
	if strings.HasPrefix(value, "[") {
		if host, _, err := net.SplitHostPort(value); err == nil {
			value = host
		} else {
			value = strings.Trim(value, "[]")
		}
	} else if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	} else if idx := strings.IndexAny(value, "/?#"); idx >= 0 {
		value = value[:idx]
	}
	return strings.Trim(strings.TrimSuffix(value, "."), "[]")
}
