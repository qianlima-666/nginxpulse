package ingest

import (
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
