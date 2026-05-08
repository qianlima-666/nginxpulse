package ingest

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/likaia/nginxpulse/internal/config"
	"github.com/likaia/nginxpulse/internal/enrich"
	"github.com/likaia/nginxpulse/internal/store"
	"github.com/sirupsen/logrus"
)

const (
	pendingLocationLabel     = "待解析"
	defaultIPGeoResolveBatch = 1000
	ipGeoFailureCooldown     = 12 * time.Hour
	ipGeoFailureFingerprint  = "ip_geo_api_failure"
)

func PendingLocationLabel() string {
	return pendingLocationLabel
}

var (
	ipGeoMu      sync.RWMutex
	ipGeoRunning bool
)

func startIPGeoParsing() bool {
	ipGeoMu.Lock()
	defer ipGeoMu.Unlock()
	if ipGeoRunning {
		return false
	}
	ipGeoRunning = true
	return true
}

func finishIPGeoParsing() {
	ipGeoMu.Lock()
	ipGeoRunning = false
	ipGeoMu.Unlock()
}

func IsIPGeoParsing() bool {
	ipGeoMu.RLock()
	defer ipGeoMu.RUnlock()
	return ipGeoRunning
}

// HasPendingIPGeo reports whether pending IP geo entries exist.
func (p *LogParser) HasPendingIPGeo() bool {
	if p == nil || p.repo == nil {
		return false
	}
	pending, err := p.repo.HasIPGeoPending()
	if err != nil {
		logrus.WithError(err).Warn("检测 IP 归属地待解析队列失败")
		return false
	}
	return pending
}

// GetIPGeoPendingCount returns the number of pending IP geo entries.
func (p *LogParser) GetIPGeoPendingCount() int64 {
	if p == nil || p.repo == nil {
		return 0
	}
	total, err := p.repo.CountIPGeoPending()
	if err != nil {
		logrus.WithError(err).Warn("读取 IP 归属地待解析数量失败")
		return 0
	}
	return total
}

// ProcessPendingIPGeo resolves pending IP geo entries and backfills locations.
func (p *LogParser) ProcessPendingIPGeo(limit int) int {
	if p == nil || p.repo == nil || p.demoMode {
		return 0
	}
	if IsIPParsing() {
		return 0
	}
	if !startIPGeoParsing() {
		return 0
	}
	defer finishIPGeoParsing()

	pendingTotal, err := p.repo.CountIPGeoPending()
	if err != nil {
		logrus.WithError(err).Warn("读取 IP 归属地待解析数量失败")
		return 0
	}
	if pendingTotal <= 0 {
		resetIPGeoProgress()
		return p.recoverPendingIPGeoFromLogs(limit)
	}
	reportIPGeoPendingCount(pendingTotal)
	touchIPGeoProgressStart()

	if limit <= 0 {
		limit = defaultIPGeoResolveBatch
	}

	cutoff := time.Now().Add(-ipGeoFailureCooldown)
	pending, err := p.repo.FetchIPGeoPendingWithCooldown(limit, cutoff)
	if err != nil {
		logrus.WithError(err).Warn("读取 IP 归属地待解析队列失败")
		return 0
	}
	if len(pending) == 0 {
		return 0
	}

	results := make(map[string]store.IPGeoCacheEntry, len(pending))
	cached, err := p.repo.GetIPGeoCache(pending)
	if err != nil {
		logrus.WithError(err).Warn("读取 IP 归属地缓存失败")
	}

	missing := make([]string, 0, len(pending))
	unknownCached := make([]string, 0)
	for _, ip := range pending {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		if entry, ok := cached[ip]; ok {
			if entry.Domestic == "未知" && entry.Global == "未知" {
				unknownCached = append(unknownCached, ip)
			} else {
				results[ip] = entry
				continue
			}
		}
		missing = append(missing, ip)
	}
	if len(unknownCached) > 0 {
		if err := p.repo.DeleteIPGeoCache(unknownCached); err != nil {
			logrus.WithError(err).Warn("清理未知 IP 归属地缓存失败")
		}
		enrich.DeleteIPGeoCacheEntries(unknownCached)
	}

	if len(missing) > 0 {
		fetched, failed, fetchErr := enrich.GetIPLocationBatch(missing)
		if fetchErr != nil {
			logrus.WithError(fetchErr).Warn("IP 归属地远端查询失败，将保留待解析 IP")
		}
		for ip, loc := range fetched {
			results[ip] = store.IPGeoCacheEntry{
				Domestic: loc.Domestic,
				Global:   loc.Global,
				Source:   loc.Source,
			}
		}
		if p.repo != nil && len(fetched) > 0 {
			entries := make(map[string]store.IPGeoCacheEntry, len(fetched))
			for ip, loc := range fetched {
				entries[ip] = store.IPGeoCacheEntry{
					Domestic: loc.Domestic,
					Global:   loc.Global,
					Source:   loc.Source,
				}
			}
			if err := p.repo.UpsertIPGeoCache(entries); err != nil {
				logrus.WithError(err).Warn("写入 IP 归属地缓存失败")
			}
			if p.ipGeoCacheLimit > 0 {
				if err := p.repo.TrimIPGeoCache(p.ipGeoCacheLimit); err != nil {
					logrus.WithError(err).Warn("清理 IP 归属地缓存失败")
				}
			}
		}
		for _, ip := range pending {
			ip = strings.TrimSpace(ip)
			if ip == "" {
				continue
			}
			if _, ok := results[ip]; ok {
				continue
			}
			if fetchErr != nil {
				continue
			}
			if _, isFailed := failed[ip]; isFailed {
				continue
			}
			results[ip] = store.IPGeoCacheEntry{
				Domestic: "未知",
				Global:   "未知",
				Source:   "unknown",
			}
		}

		if p.repo != nil && (fetchErr != nil || len(failed) > 0) {
			failureRecords := make(map[string]string, len(failed))
			for ip, reason := range failed {
				failureRecords[ip] = reason
			}
			if fetchErr != nil {
				for _, ip := range pending {
					ip = strings.TrimSpace(ip)
					if ip == "" {
						continue
					}
					if _, ok := results[ip]; ok {
						continue
					}
					if _, ok := failureRecords[ip]; ok {
						continue
					}
					failureRecords[ip] = "request_error"
				}
			}
			detail := ""
			if fetchErr != nil {
				detail = fetchErr.Error()
			}
			if len(failureRecords) > 0 {
				if err := p.repo.InsertIPGeoAPIFailures(failureRecords, "ip-api", detail, 0); err != nil {
					logrus.WithError(err).Warn("记录 IP 归属地远端失败失败")
				}
				samples := make([]string, 0, 3)
				for ip := range failureRecords {
					samples = append(samples, ip)
					if len(samples) >= 3 {
						break
					}
				}
				p.notifySystem(
					"warning",
					"ip_geo",
					"IP 归属地查询失败",
					fmt.Sprintf("远端 IP 归属地查询失败，已记录 %d 个 IP。", len(failureRecords)),
					"ip_geo_api_failure",
					map[string]interface{}{
						"count":   len(failureRecords),
						"samples": samples,
						"error":   detail,
					},
				)
			}
		}
	} else {
		for _, ip := range pending {
			ip = strings.TrimSpace(ip)
			if ip == "" {
				continue
			}
			if _, ok := results[ip]; ok {
				continue
			}
			results[ip] = store.IPGeoCacheEntry{
				Domestic: "未知",
				Global:   "未知",
				Source:   "unknown",
			}
		}
	}

	if err := p.repo.UpdateIPGeoLocations(results, pendingLocationLabel); err != nil {
		logrus.WithError(err).Warn("回填 IP 归属地失败")
		return 0
	}

	resolved := make([]string, 0, len(results))
	for ip := range results {
		resolved = append(resolved, ip)
	}
	if len(resolved) > 0 {
		if err := p.repo.DeleteIPGeoPending(resolved); err != nil {
			logrus.WithError(err).Warn("清理 IP 归属地待解析队列失败")
			return 0
		}
		p.cleanupResolvedIPGeoFailures(resolved)
	}

	addIPGeoProcessed(int64(len(resolved)))
	if pendingTotal <= int64(len(resolved)) {
		finalizeIPGeoProgress()
	}

	return len(resolved)
}

func (p *LogParser) cleanupResolvedIPGeoFailures(resolved []string) {
	if p == nil || p.repo == nil || len(resolved) == 0 {
		return
	}
	deleted, err := p.repo.DeleteIPGeoAPIFailuresByIPs(resolved)
	if err != nil {
		logrus.WithError(err).Warn("清理已恢复的 IP 归属地失败记录失败")
		return
	}
	if deleted <= 0 {
		return
	}
	remaining, err := p.repo.CountIPGeoAPIFailures()
	if err != nil {
		logrus.WithError(err).Warn("统计剩余 IP 归属地失败记录失败")
		return
	}
	if remaining > 0 {
		return
	}
	if _, err := p.repo.DeleteSystemNotificationByFingerprint(ipGeoFailureFingerprint); err != nil {
		logrus.WithError(err).Warn("清理 IP 归属地失败系统通知失败")
	}
}

func (p *LogParser) recoverPendingIPGeoFromLogs(limit int) int {
	if p == nil || p.repo == nil || p.demoMode {
		return 0
	}
	if limit <= 0 {
		limit = defaultIPGeoResolveBatch
	}

	pending := make([]string, 0, limit)
	seen := make(map[string]struct{}, limit)
	for _, websiteID := range config.GetAllWebsiteIDs() {
		if len(pending) >= limit {
			break
		}
		ips, err := p.repo.FetchPendingIPGeoFromLogs(websiteID, pendingLocationLabel, limit-len(pending))
		if err != nil {
			logrus.WithError(err).Warn("从日志中提取待解析 IP 失败")
			continue
		}
		for _, ip := range ips {
			if ip == "" {
				continue
			}
			if _, ok := seen[ip]; ok {
				continue
			}
			seen[ip] = struct{}{}
			pending = append(pending, ip)
			if len(pending) >= limit {
				break
			}
		}
	}
	if len(pending) == 0 {
		return 0
	}

	cached := make(map[string]store.IPGeoCacheEntry)
	if entries, err := p.repo.GetIPGeoCache(pending); err != nil {
		logrus.WithError(err).Warn("读取 IP 归属地缓存失败")
	} else if len(entries) > 0 {
		for ip, entry := range entries {
			if entry.Domestic == "未知" && entry.Global == "未知" {
				continue
			}
			cached[ip] = entry
		}
		if err := p.repo.UpdateIPGeoLocations(cached, pendingLocationLabel); err != nil {
			logrus.WithError(err).Warn("回填缓存中的 IP 归属地失败")
		}
	}

	missing := make([]string, 0, len(pending))
	for _, ip := range pending {
		if _, ok := cached[ip]; ok {
			continue
		}
		missing = append(missing, ip)
	}
	if len(missing) > 0 {
		if err := p.repo.UpsertIPGeoPending(missing); err != nil {
			logrus.WithError(err).Warn("补充 IP 归属地待解析队列失败")
		}
	}

	return len(cached)
}
