package ingest

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/likaia/nginxpulse/internal/config"
	"github.com/likaia/nginxpulse/internal/ingest/source"
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

type fakeHostDiscoverySource struct {
	targets []source.TargetRef
	content map[string]string
}

func (f *fakeHostDiscoverySource) ID() string {
	return "fake"
}

func (f *fakeHostDiscoverySource) Type() source.SourceType {
	return source.SourceSFTP
}

func (f *fakeHostDiscoverySource) ListTargets(ctx context.Context) ([]source.TargetRef, error) {
	_ = ctx
	return f.targets, nil
}

func (f *fakeHostDiscoverySource) OpenRange(ctx context.Context, target source.TargetRef, start, end int64) (io.ReadCloser, error) {
	_ = ctx
	content := f.content[target.Key]
	if start > int64(len(content)) {
		start = int64(len(content))
	}
	if end > int64(len(content)) {
		end = int64(len(content))
	}
	if end > start && end > 0 {
		content = content[start:end]
	} else {
		content = content[start:]
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

func (f *fakeHostDiscoverySource) OpenStream(ctx context.Context, target source.TargetRef) (io.ReadCloser, error) {
	_ = ctx
	_ = target
	return nil, source.ErrStreamNotSupported
}

func (f *fakeHostDiscoverySource) Stat(ctx context.Context, target source.TargetRef) (source.TargetMeta, error) {
	_ = ctx
	return source.TargetMeta{Size: int64(len(f.content[target.Key]))}, nil
}

func TestDiscoverHostsForTemplateSupportsSFTPSource(t *testing.T) {
	originalFactory := newHostDiscoverySource
	t.Cleanup(func() {
		newHostDiscoverySource = originalFactory
	})

	logContent := strings.Join([]string{
		`203.0.113.10 [24/Apr/2026:10:05:35 +0800] "GET /ignored HTTP/1.1" 200 512 "-" "curl/8.0" -`,
		`203.0.113.11 [24/Apr/2026:10:05:36 +0800] "GET / HTTP/1.1" 200 512 "-" "curl/8.0" SFTP.Example.COM`,
	}, "\n") + "\n"

	newHostDiscoverySource = func(websiteID string, cfg config.SourceConfig) (source.LogSource, error) {
		_ = websiteID
		_ = cfg
		return &fakeHostDiscoverySource{
			targets: []source.TargetRef{{
				SourceID: cfg.ID,
				Key:      "/remote/access.log",
				Meta: source.TargetMeta{
					Size: int64(len(logContent)),
				},
			}},
			content: map[string]string{
				"/remote/access.log": logContent,
			},
		}, nil
	}

	parser := &LogParser{}
	hosts := parser.discoverHostsForTemplate(config.WebsiteConfig{
		Name:      "SFTP Auto Discover",
		LogFormat: `$remote_addr [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $host`,
		Sources: []config.SourceConfig{{
			ID:   "sftp-main",
			Type: "sftp",
			Host: "10.0.0.10",
			Path: "/remote/access.log",
			User: "nginxpulse",
		}},
	})

	if len(hosts) != 1 || hosts[0] != "sftp.example.com" {
		t.Fatalf("unexpected hosts: %#v", hosts)
	}
}
