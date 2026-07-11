package worker

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/qianlima-666/nginxpulse/internal/config"
	"github.com/qianlima-666/nginxpulse/internal/enrich"
	"github.com/qianlima-666/nginxpulse/internal/store"
	"github.com/sirupsen/logrus"
)

var (
	demoMethods = []string{"GET", "POST", "PUT", "DELETE"}
	demoPaths   = []string{
		"/", "/index.html", "/login", "/logout", "/dashboard",
		"/api/stats/overall", "/api/stats/logs", "/api/stats/location",
		"/api/user/profile", "/api/user/login", "/api/user/logout",
		"/api/content/posts", "/api/realtime",
		"/assets/app.js", "/assets/app.css", "/images/logo.png",
		"/health", "/rss.xml", "/sitemap.xml", "/favicon.ico",
		"/docs/getting-started", "/search?q=nginx", "/admin", "/admin/settings",
	}
	demoReferers = []string{
		"-",
		"http://192.168.6.131:9200/",
		"http://192.168.6.131:9200/daily",
		"https://intranet.local/dashboard",
		"https://app.internal.local/login",
	}
	demoUserAgents = []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Mobile Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 SafeLine-CE/v9-2-8",
	}
	demoStatusCodes = []int{200, 200, 200, 204, 301, 302, 400, 401, 403, 404, 500}
)

// RunDemoGenerator inserts random log records into the database on a fixed interval.
func RunDemoGenerator(ctx context.Context, repo *store.Repository, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		select {
		case <-ticker.C:
			generateDemoLogs(repo, rng)
		case <-ctx.Done():
			return
		}
	}
}

func generateDemoLogs(repo *store.Repository, rng *rand.Rand) {
	websiteIDs := config.GetAllWebsiteIDs()
	if len(websiteIDs) == 0 {
		return
	}

	externalIPs := loadExternalIPs()

	for _, id := range websiteIDs {
		count := rng.Intn(18) + 8 // 8-25 per minute per site
		batch := make([]store.NginxLogRecord, 0, count)
		now := time.Now()

		for i := 0; i < count; i++ {
			ip := randomDemoIP(rng, externalIPs)
			method := demoMethods[rng.Intn(len(demoMethods))]
			path := demoPaths[rng.Intn(len(demoPaths))]
			status := demoStatusCodes[rng.Intn(len(demoStatusCodes))]
			bytesSent := rng.Intn(48000-120) + 120
			referer := demoReferers[rng.Intn(len(demoReferers))]
			ua := demoUserAgents[rng.Intn(len(demoUserAgents))]
			browser, osName, device := enrich.ParseUserAgent(ua)

			timestamp := now.Add(-time.Duration(rng.Intn(60)) * time.Second)
			pageviewFlag := enrich.ShouldCountAsPageView(status, path, ip)

			batch = append(batch, store.NginxLogRecord{
				IP:               ip,
				PageviewFlag:     pageviewFlag,
				Timestamp:        timestamp,
				Method:           method,
				Url:              path,
				Status:           status,
				BytesSent:        bytesSent,
				Referer:          referer,
				UserBrowser:      browser,
				UserOs:           osName,
				UserDevice:       device,
				DomesticLocation: "",
				GlobalLocation:   "",
			})
		}

		fillDemoLocations(batch)

		if err := repo.BatchInsertLogsForWebsite(id, batch); err != nil {
			logrus.WithError(err).Warnf("demo data insert failed: %s", id)
		}
	}
}

func fillDemoLocations(batch []store.NginxLogRecord) {
	ips := make([]string, 0, len(batch))
	for _, entry := range batch {
		ips = append(ips, entry.IP)
	}

	locations, _, _ := enrich.GetIPLocationBatch(ips)
	for i := range batch {
		if location, ok := locations[batch[i].IP]; ok {
			batch[i].DomesticLocation = location.Domestic
			batch[i].GlobalLocation = location.Global
		} else {
			batch[i].DomesticLocation = "未知"
			batch[i].GlobalLocation = "未知"
		}
	}
}

func randomDemoIP(rng *rand.Rand, externalIPs []string) string {
	if len(externalIPs) > 0 {
		return externalIPs[rng.Intn(len(externalIPs))]
	}
	return randomPrivateIP(rng)
}

func randomPrivateIP(rng *rand.Rand) string {
	switch rng.Intn(4) {
	case 0:
		return "10." + randOctet(rng) + "." + randOctet(rng) + "." + randOctet(rng)
	case 1:
		return "172." + intRange(rng, 16, 31) + "." + randOctet(rng) + "." + randOctet(rng)
	case 2:
		return "192.168." + randOctet(rng) + "." + randOctet(rng)
	default:
		return "127.0.0.1"
	}
}

func randOctet(rng *rand.Rand) string {
	return intRange(rng, 1, 254)
}

func intRange(rng *rand.Rand, min, max int) string {
	return strconv.Itoa(rng.Intn(max-min+1) + min)
}

func loadExternalIPs() []string {
	paths := []string{
		filepath.Join(config.DataDir, "external_ips.txt"),
		filepath.Join("/app", "assets", "external_ips.txt"),
		filepath.Join("docs", "external_ips.txt"),
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		ips := make([]string, 0, len(lines))
		for _, line := range lines {
			ip := strings.TrimSpace(line)
			if ip == "" {
				continue
			}
			ips = append(ips, ip)
		}
		if len(ips) > 0 {
			return ips
		}
	}

	return nil
}
