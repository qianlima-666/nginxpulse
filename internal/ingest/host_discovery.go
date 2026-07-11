package ingest

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/likaia/nginxpulse/internal/config"
	"github.com/likaia/nginxpulse/internal/ingest/source"
	"github.com/sirupsen/logrus"
)

const maxHostDiscoveryLinesPerFile = 2000
const maxHostDiscoveryTailBytes int64 = 512 * 1024
const maxHostDiscoveryScannerBuffer = 1024 * 1024

var domainLikePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*$`)

var newHostDiscoverySource = source.NewFromConfig

func (p *LogParser) discoverAutoHostWebsites() {
	if p.repo == nil {
		return
	}

	templates := config.GetAutoDiscoverWebsiteTemplates()
	if len(templates) == 0 {
		return
	}

	for _, template := range templates {
		hosts := p.discoverHostsForTemplate(template)
		for _, host := range hosts {
			site := template
			site.Name = host
			site.Domains = []string{host}
			site.AutoDiscoverHosts = false

			websiteID, registered := config.RegisterRuntimeWebsite(site)
			if websiteID == "" {
				continue
			}
			if registered {
				if err := p.repo.EnsureWebsiteSchema(websiteID); err != nil {
					config.UnregisterRuntimeWebsite(websiteID)
					logrus.WithError(err).Warnf("自动发现站点 %s 建表失败", host)
					continue
				}
				logrus.Infof("自动发现站点: %s (%s)", host, websiteID)
			}
		}
	}
}

func (p *LogParser) discoverHostsForTemplate(template config.WebsiteConfig) []string {
	hosts := make(map[string]struct{})
	addHosts := func(parser *logLineParser, paths []string) {
		for _, path := range paths {
			for _, host := range p.discoverHostsInPath(path, parser) {
				hosts[host] = struct{}{}
			}
		}
	}

	if len(template.Sources) > 0 {
		for i := range template.Sources {
			sourceCfg := template.Sources[i]
			mode := strings.ToLower(strings.TrimSpace(sourceCfg.Mode))
			if mode == "stream" {
				continue
			}
			parser, err := newLogLineParser(template, &sourceCfg)
			if err != nil {
				logrus.WithError(err).Warnf("自动发现模板 %s 解析配置无效", template.Name)
				continue
			}
			if strings.ToLower(strings.TrimSpace(sourceCfg.Type)) == "local" {
				addHosts(parser, localSourceDiscoveryPaths(sourceCfg))
				continue
			}
			for _, host := range p.discoverHostsInSource(template, sourceCfg, parser) {
				hosts[host] = struct{}{}
			}
		}
	} else {
		parser, err := newLogLineParser(template, nil)
		if err != nil {
			logrus.WithError(err).Warnf("自动发现模板 %s 解析配置无效", template.Name)
			return nil
		}
		addHosts(parser, []string{template.LogPath})
	}

	result := make([]string, 0, len(hosts))
	for host := range hosts {
		result = append(result, host)
	}
	return result
}

func (p *LogParser) discoverHostsInSource(template config.WebsiteConfig, sourceCfg config.SourceConfig, parser *logLineParser) []string {
	src, err := newHostDiscoverySource(template.Name, sourceCfg)
	if err != nil {
		logrus.WithError(err).Warnf("自动发现模板 %s 初始化 source %s 失败", template.Name, sourceCfg.ID)
		return nil
	}

	targets, err := src.ListTargets(context.Background())
	if err != nil {
		logrus.WithError(err).Warnf("自动发现模板 %s 枚举 source %s 目标失败", template.Name, sourceCfg.ID)
		return nil
	}

	hosts := make(map[string]struct{})
	for _, target := range targets {
		for _, host := range p.discoverHostsInTarget(context.Background(), src, target, parser) {
			hosts[host] = struct{}{}
		}
	}

	result := make([]string, 0, len(hosts))
	for host := range hosts {
		result = append(result, host)
	}
	return result
}

func localSourceDiscoveryPaths(sourceCfg config.SourceConfig) []string {
	paths := make([]string, 0, 2)
	if strings.TrimSpace(sourceCfg.Path) != "" {
		paths = append(paths, sourceCfg.Path)
	}
	if strings.TrimSpace(sourceCfg.Pattern) != "" {
		paths = append(paths, sourceCfg.Pattern)
	}
	return paths
}

func (p *LogParser) discoverHostsInPath(path string, parser *logLineParser) []string {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}

	paths := []string{path}
	if strings.Contains(path, "*") {
		matches, err := filepath.Glob(path)
		if err != nil {
			logrus.WithError(err).Warnf("自动发现解析日志路径模式失败: %s", path)
			return nil
		}
		paths = matches
	}

	hosts := make(map[string]struct{})
	for _, filePath := range paths {
		fileHosts := p.discoverHostsInFile(filePath, parser)
		for _, host := range fileHosts {
			hosts[host] = struct{}{}
		}
	}

	result := make([]string, 0, len(hosts))
	for host := range hosts {
		result = append(result, host)
	}
	return result
}

func (p *LogParser) discoverHostsInFile(filePath string, parser *logLineParser) []string {
	file, err := os.Open(filePath)
	if err != nil {
		logrus.WithError(err).Warnf("自动发现无法打开日志文件: %s", filePath)
		return nil
	}
	defer file.Close()

	var reader io.Reader = file
	var closer io.Closer
	if isGzipFile(filePath) {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			logrus.WithError(err).Warnf("自动发现无法解析 gzip 日志文件: %s", filePath)
			return nil
		}
		reader = gzReader
		closer = gzReader
	} else {
		hosts, err := p.discoverHostsInRecentPlainFile(file, parser)
		if err != nil {
			logrus.WithError(err).Warnf("自动发现读取日志尾部失败: %s", filePath)
		} else if len(hosts) > 0 {
			return hosts
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			logrus.WithError(err).Warnf("自动发现无法重置日志文件位置: %s", filePath)
			return nil
		}
	}
	if closer != nil {
		defer closer.Close()
	}

	return discoverHostsFromReader(reader, parser, maxHostDiscoveryLinesPerFile, filePath)
}

func (p *LogParser) discoverHostsInRecentPlainFile(file *os.File, parser *logLineParser) ([]string, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	if size <= 0 {
		return nil, nil
	}

	offset := size - maxHostDiscoveryTailBytes
	if offset < 0 {
		offset = 0
	}

	buf := make([]byte, int(size-offset))
	if _, err := file.ReadAt(buf, offset); err != nil && err != io.EOF {
		return nil, err
	}

	if offset > 0 {
		newline := bytes.IndexByte(buf, '\n')
		if newline < 0 {
			return nil, nil
		}
		buf = buf[newline+1:]
	}
	if len(buf) == 0 {
		return nil, nil
	}

	return discoverHostsFromReader(bytes.NewReader(buf), parser, 0, file.Name()), nil
}

func (p *LogParser) discoverHostsInTarget(ctx context.Context, src source.LogSource, target source.TargetRef, parser *logLineParser) []string {
	meta := target.Meta
	if meta.Size == 0 && meta.ModTime.IsZero() && meta.ETag == "" {
		updated, err := src.Stat(ctx, target)
		if err != nil {
			logrus.WithError(err).Warnf("自动发现无法获取远端目标信息: %s", target.Key)
			return nil
		}
		meta = updated
		target.Meta = updated
	}

	if meta.Compressed {
		reader, err := openHostDiscoveryRange(ctx, src, target, 0, -1)
		if err != nil {
			logrus.WithError(err).Warnf("自动发现无法读取压缩远端日志: %s", target.Key)
			return nil
		}
		if reader == nil {
			return nil
		}
		defer reader.Close()

		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			logrus.WithError(err).Warnf("自动发现无法解析远端 gzip 日志文件: %s", target.Key)
			return nil
		}
		defer gzReader.Close()

		return discoverHostsFromReader(gzReader, parser, maxHostDiscoveryLinesPerFile, target.Key)
	}

	hosts, err := p.discoverHostsInRecentTarget(ctx, src, target, parser)
	if err != nil {
		logrus.WithError(err).Warnf("自动发现读取远端日志尾部失败: %s", target.Key)
	} else if len(hosts) > 0 {
		return hosts
	}

	reader, err := openHostDiscoveryRange(ctx, src, target, 0, maxHostDiscoveryTailBytes)
	if err != nil {
		logrus.WithError(err).Warnf("自动发现无法读取远端日志文件: %s", target.Key)
		return nil
	}
	if reader == nil {
		return nil
	}
	defer reader.Close()

	return discoverHostsFromReader(reader, parser, maxHostDiscoveryLinesPerFile, target.Key)
}

func (p *LogParser) discoverHostsInRecentTarget(ctx context.Context, src source.LogSource, target source.TargetRef, parser *logLineParser) ([]string, error) {
	size := target.Meta.Size
	if size <= 0 {
		return nil, nil
	}

	offset := size - maxHostDiscoveryTailBytes
	if offset < 0 {
		offset = 0
	}

	reader, err := openHostDiscoveryRange(ctx, src, target, offset, -1)
	if err != nil {
		return nil, err
	}
	if reader == nil {
		return nil, nil
	}
	defer reader.Close()

	buf, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	if offset > 0 {
		newline := bytes.IndexByte(buf, '\n')
		if newline < 0 {
			return nil, nil
		}
		buf = buf[newline+1:]
	}
	if len(buf) == 0 {
		return nil, nil
	}

	return discoverHostsFromReader(bytes.NewReader(buf), parser, 0, target.Key), nil
}

func openHostDiscoveryRange(ctx context.Context, src source.LogSource, target source.TargetRef, start, end int64) (io.ReadCloser, error) {
	reader, err := src.OpenRange(ctx, target, start, end)
	if err != nil {
		if errors.Is(err, source.ErrRangeNotSupported) && start > 0 {
			reader, err = src.OpenRange(ctx, target, 0, -1)
			if err != nil {
				return nil, err
			}
			if skipErr := skipReaderBytes(reader, start); skipErr != nil {
				reader.Close()
				return nil, skipErr
			}
			return reader, nil
		}
		return nil, err
	}
	return reader, nil
}

func discoverHostsFromReader(reader io.Reader, parser *logLineParser, maxLines int, filePath string) []string {
	hosts := make(map[string]struct{})
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), maxHostDiscoveryScannerBuffer)
	lines := 0
	for scanner.Scan() {
		lines++
		if host := extractHostForDiscovery(parser, scanner.Text()); host != "" {
			hosts[host] = struct{}{}
		}
		if maxLines > 0 && lines >= maxLines {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		logrus.WithError(err).Warnf("自动发现扫描日志文件失败: %s", filePath)
	}

	result := make([]string, 0, len(hosts))
	for host := range hosts {
		result = append(result, host)
	}
	return result
}

func extractHostForDiscovery(parser *logLineParser, line string) string {
	switch parser.parseType {
	case parseTypeCaddyJSON:
		decoder := json.NewDecoder(strings.NewReader(line))
		decoder.UseNumber()
		var payload map[string]interface{}
		if err := decoder.Decode(&payload); err != nil {
			return ""
		}
		request := getMap(payload, "request")
		return normalizeDiscoveredHost(getString(request, "host"))
	default:
		matches := parser.regex.FindStringSubmatch(line)
		if len(matches) == 0 {
			return ""
		}
		return normalizeDiscoveredHost(extractField(matches, parser.indexMap, hostAliases))
	}
}

func normalizeDiscoveredHost(raw string) string {
	value := normalizeOptionalField(raw)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "://") {
		if parsed, err := url.Parse(value); err == nil {
			value = parsed.Host
		}
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	value = strings.Trim(value, "[]")
	value = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
	if value == "" || len(value) > 255 {
		return ""
	}
	if net.ParseIP(value) != nil {
		return value
	}
	if !domainLikePattern.MatchString(value) {
		return ""
	}
	return value
}
