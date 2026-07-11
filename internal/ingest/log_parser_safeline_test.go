package ingest

import (
	"fmt"
	"testing"
	"time"

	"github.com/qianlima-666/nginxpulse/internal/config"
)

func TestSafeLineDefaultParserParsesRayWAFLine(t *testing.T) {
	parser, err := newLogLineParser(config.WebsiteConfig{LogType: "safeline"}, nil)
	if err != nil {
		t.Fatalf("newLogLineParser(safeline) error: %v", err)
	}

	now := time.Now().In(time.FixedZone("CST", 8*3600)).Truncate(time.Second)
	line := fmt.Sprintf(
		`192.168.1.242 - - [%s] "1.111.com" "GET /csgx/api/webservice/rules?=1770383547502 HTTP/2.0" 200 36 "-" "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36 Edg/144.0.0.0" "-"`,
		now.Format(defaultNginxTimeLayout),
	)

	p := &LogParser{retentionDays: 30}
	record, err := p.parseRegexLogLine(parser, line)
	if err != nil {
		t.Fatalf("parseRegexLogLine error: %v", err)
	}

	if record.IP != "192.168.1.242" {
		t.Fatalf("unexpected ip: %q", record.IP)
	}
	if record.Method != "GET" {
		t.Fatalf("unexpected method: %q", record.Method)
	}
	if record.Url != "/csgx/api/webservice/rules?=1770383547502" {
		t.Fatalf("unexpected url: %q", record.Url)
	}
	if record.Status != 200 {
		t.Fatalf("unexpected status: %d", record.Status)
	}
	if record.BytesSent != 36 {
		t.Fatalf("unexpected bytes: %d", record.BytesSent)
	}
}

func TestSafeLineAliasRayWAF(t *testing.T) {
	parser, err := newLogLineParser(config.WebsiteConfig{LogType: "raywaf"}, nil)
	if err != nil {
		t.Fatalf("newLogLineParser(raywaf) error: %v", err)
	}
	if parser.source != "safeline-waf" {
		t.Fatalf("unexpected parser source: %q", parser.source)
	}
}

func TestSafeLinePipeSeparatedLineWithoutXFF(t *testing.T) {
	parser, err := newLogLineParser(config.WebsiteConfig{LogType: "safeline"}, nil)
	if err != nil {
		t.Fatalf("newLogLineParser(safeline) error: %v", err)
	}

	line := `222.176.201.91 | - | 13/Mar/2026:19:45:13 +0800 | "154.219.111.204" | "POST /api/v1/tunnel/tunnel HTTP/1.1" | 468 | 14862 | "-" | "Mozilla/5.0 (Linux; Android 10; Q) AppleWebKit/605.1.15 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/605.1.15 Edg/83.0.478.45 APNetwork/1.2.6"`

	p := &LogParser{retentionDays: 365}
	record, err := p.parseRegexLogLine(parser, line)
	if err != nil {
		t.Fatalf("parseRegexLogLine error: %v", err)
	}

	if record.IP != "222.176.201.91" {
		t.Fatalf("unexpected ip: %q", record.IP)
	}
	if record.Method != "POST" {
		t.Fatalf("unexpected method: %q", record.Method)
	}
	if record.Url != "/api/v1/tunnel/tunnel" {
		t.Fatalf("unexpected url: %q", record.Url)
	}
	if record.Status != 468 {
		t.Fatalf("unexpected status: %d", record.Status)
	}
	if record.BytesSent != 14862 {
		t.Fatalf("unexpected bytes: %d", record.BytesSent)
	}
}

func TestSafeLinePipeSeparatedLineWithXFF(t *testing.T) {
	parser, err := newLogLineParser(config.WebsiteConfig{LogType: "leichi"}, nil)
	if err != nil {
		t.Fatalf("newLogLineParser(leichi) error: %v", err)
	}

	line := `120.26.17.71 | - | 13/Mar/2026:20:24:43 +0800 | "audioshare.cn" | "GET /.safeline/static/favicon.png HTTP/1.1" | 200 | 5877 | "https://audioshare.cn/archives/category/%e6%95%99%e7%a8%8b" | "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36" "125.107.193.16"`

	p := &LogParser{retentionDays: 365}
	record, err := p.parseRegexLogLine(parser, line)
	if err != nil {
		t.Fatalf("parseRegexLogLine error: %v", err)
	}

	if record.IP != "120.26.17.71" {
		t.Fatalf("unexpected ip: %q", record.IP)
	}
	if record.Method != "GET" {
		t.Fatalf("unexpected method: %q", record.Method)
	}
	if record.Url != "/.safeline/static/favicon.png" {
		t.Fatalf("unexpected url: %q", record.Url)
	}
	if record.Host != "audioshare.cn" {
		t.Fatalf("unexpected host: %q", record.Host)
	}
}
