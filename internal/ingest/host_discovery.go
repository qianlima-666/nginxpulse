package ingest

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/likaia/nginxpulse/internal/config"
	"github.com/sirupsen/logrus"
)

const maxHostDiscoveryLinesPerFile = 2000

var domainLikePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*$`)

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
			if strings.ToLower(strings.TrimSpace(sourceCfg.Type)) != "local" {
				continue
			}
			parser, err := newLogLineParser(template, &sourceCfg)
			if err != nil {
				logrus.WithError(err).Warnf("自动发现模板 %s 解析配置无效", template.Name)
				continue
			}
			addHosts(parser, localSourceDiscoveryPaths(sourceCfg))
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
	}
	if closer != nil {
		defer closer.Close()
	}

	hosts := make(map[string]struct{})
	scanner := bufio.NewScanner(reader)
	lines := 0
	for scanner.Scan() {
		lines++
		if host := extractHostForDiscovery(parser, scanner.Text()); host != "" {
			hosts[host] = struct{}{}
		}
		if lines >= maxHostDiscoveryLinesPerFile {
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
