package ingest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/likaia/nginxpulse/internal/config"
)

func TestExtractHostForDiscoveryFromNPMLine(t *testing.T) {
	parser, err := newLogLineParser(config.WebsiteConfig{LogType: "nginx-proxy-manager"}, nil)
	if err != nil {
		t.Fatalf("newLogLineParser error: %v", err)
	}

	line := `[24/Apr/2026:10:05:35 +0800] - 200 200 - GET https app.example.com "/dashboard" [Client 203.0.113.10] [Length 512] [Gzip -] [Sent-to 10.0.0.2] "Mozilla/5.0" "https://example.com/"`
	host := extractHostForDiscovery(parser, line)
	if host != "app.example.com" {
		t.Fatalf("unexpected host: %q", host)
	}
}

func TestExtractHostForDiscoveryFromLogFormat(t *testing.T) {
	parser, err := newLogLineParser(config.WebsiteConfig{
		LogFormat: `$remote_addr [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $host`,
	}, nil)
	if err != nil {
		t.Fatalf("newLogLineParser error: %v", err)
	}

	line := `203.0.113.10 [24/Apr/2026:10:05:35 +0800] "GET / HTTP/1.1" 200 512 "-" "curl/8.0" WWW.Example.COM.`
	host := extractHostForDiscovery(parser, line)
	if host != "www.example.com" {
		t.Fatalf("unexpected host: %q", host)
	}
}

func TestNormalizeDiscoveredHostRejectsInvalidHost(t *testing.T) {
	if host := normalizeDiscoveredHost("bad host/value"); host != "" {
		t.Fatalf("expected invalid host to be rejected, got %q", host)
	}
}

func TestDiscoverHostsInFilePrefersRecentLines(t *testing.T) {
	parser, err := newLogLineParser(config.WebsiteConfig{
		LogFormat: `$remote_addr [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $host`,
	}, nil)
	if err != nil {
		t.Fatalf("newLogLineParser error: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")

	lines := make([]string, 0, maxHostDiscoveryLinesPerFile+1)
	for i := 0; i < maxHostDiscoveryLinesPerFile; i++ {
		lines = append(lines, fmt.Sprintf(
			`203.0.113.%d [24/Apr/2026:10:05:35 +0800] "GET /old HTTP/1.1" 200 512 "-" "curl/8.0" -`,
			(i%200)+1,
		))
	}
	lines = append(lines, `203.0.113.10 [24/Apr/2026:10:05:35 +0800] "GET /latest HTTP/1.1" 200 512 "-" "curl/8.0" recent.example.com`)

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	parserInstance := &LogParser{}
	hosts := parserInstance.discoverHostsInFile(path, parser)
	if len(hosts) != 1 || hosts[0] != "recent.example.com" {
		t.Fatalf("unexpected hosts: %#v", hosts)
	}
}
