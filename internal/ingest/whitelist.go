package ingest

import (
	"fmt"
	"strings"
	"time"

	"github.com/qianlima-666/nginxpulse/internal/config"
	"github.com/qianlima-666/nginxpulse/internal/enrich"
	"github.com/qianlima-666/nginxpulse/internal/store"
	"github.com/sirupsen/logrus"
)

type whitelistHit struct {
	count       int
	websiteID   string
	match       enrich.WhitelistMatch
	log         store.NginxLogRecord
	fingerprint string
}

func (p *LogParser) recordWhitelistHit(
	websiteID string,
	log store.NginxLogRecord,
	match enrich.WhitelistMatch,
	hits map[string]*whitelistHit,
) map[string]*whitelistHit {
	fingerprint := buildWhitelistFingerprint(websiteID, match.RuleType, match.RuleValue, log.IP)
	if fingerprint == "" {
		return hits
	}
	if hits == nil {
		hits = make(map[string]*whitelistHit)
	}
	if existing, ok := hits[fingerprint]; ok {
		existing.count++
		return hits
	}
	hits[fingerprint] = &whitelistHit{
		count:       1,
		websiteID:   websiteID,
		match:       match,
		log:         log,
		fingerprint: fingerprint,
	}
	return hits
}

func mergeWhitelistHits(dst, src map[string]*whitelistHit) map[string]*whitelistHit {
	if len(src) == 0 {
		return dst
	}
	if dst == nil {
		dst = make(map[string]*whitelistHit, len(src))
	}
	for fingerprint, hit := range src {
		if existing, ok := dst[fingerprint]; ok {
			existing.count += hit.count
			continue
		}
		dst[fingerprint] = hit
	}
	return dst
}

func (p *LogParser) flushWhitelistHits(hits map[string]*whitelistHit) {
	if p == nil || p.repo == nil || len(hits) == 0 {
		return
	}
	for _, hit := range hits {
		siteName := ""
		if site, ok := config.GetWebsiteByID(hit.websiteID); ok {
			siteName = site.Name
		}
		metadata := map[string]interface{}{
			"website_id": hit.websiteID,
			"ip":         hit.log.IP,
			"url":        hit.log.Url,
			"method":     hit.log.Method,
			"status":     hit.log.Status,
			"timestamp":  hit.log.Timestamp.Unix(),
			"time":       hit.log.Timestamp.Format(time.RFC3339),
			"rule_type":  hit.match.RuleType,
			"rule_value": hit.match.RuleValue,
		}
		if siteName != "" {
			metadata["website_name"] = siteName
		}
		if hit.match.Domestic != "" {
			metadata["domestic_location"] = hit.match.Domestic
		}
		if hit.match.Global != "" {
			metadata["global_location"] = hit.match.Global
		}
		entry := store.SystemNotification{
			Level:       "info",
			Category:    "whitelist",
			Title:       "白名单命中",
			Message:     buildWhitelistMessage(siteName, hit.log, hit.match),
			Fingerprint: hit.fingerprint,
			Metadata:    metadata,
		}
		if _, err := p.repo.CreateSystemNotificationWithCount(entry, hit.count); err != nil {
			logrus.WithError(err).Warn("写入白名单命中通知失败")
		}
	}
}

func buildWhitelistFingerprint(websiteID, ruleType, ruleValue, ip string) string {
	normalizedID := strings.TrimSpace(websiteID)
	normalizedIP := strings.TrimSpace(ip)
	if normalizedID == "" || normalizedIP == "" || ruleType == "" {
		return ""
	}
	ruleValue = strings.TrimSpace(ruleValue)
	if ruleValue == "" {
		ruleValue = "-"
	}
	return fmt.Sprintf("whitelist:%s:%s:%s:%s", normalizedID, ruleType, ruleValue, normalizedIP)
}

func buildWhitelistMessage(siteName string, log store.NginxLogRecord, match enrich.WhitelistMatch) string {
	label := whitelistRuleLabel(match.RuleType)
	message := ""
	if match.RuleType == "non_mainland" {
		message = fmt.Sprintf("IP %s 命中%s", log.IP, label)
	} else if match.RuleValue != "" {
		message = fmt.Sprintf("IP %s 命中%s: %s", log.IP, label, match.RuleValue)
	} else {
		message = fmt.Sprintf("IP %s 命中%s", log.IP, label)
	}
	if siteName != "" {
		message = fmt.Sprintf("站点 %s · %s", siteName, message)
	}
	if location := formatWhitelistLocation(match.Domestic, match.Global); location != "" {
		message = fmt.Sprintf("%s · 位置 %s", message, location)
	}
	return message
}

func whitelistRuleLabel(ruleType string) string {
	switch ruleType {
	case "ip":
		return "IP"
	case "cidr":
		return "IP 段"
	case "range":
		return "IP 段"
	case "city":
		return "城市"
	case "non_mainland":
		return "非大陆访问"
	default:
		return "规则"
	}
}

func formatWhitelistLocation(domestic, global string) string {
	d := strings.TrimSpace(domestic)
	g := strings.TrimSpace(global)
	if d == "" && g == "" {
		return ""
	}
	if d == "" {
		return g
	}
	if g == "" {
		return d
	}
	return d + " / " + g
}
