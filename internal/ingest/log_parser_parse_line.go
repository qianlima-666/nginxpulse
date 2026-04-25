package ingest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/likaia/nginxpulse/internal/enrich"
	"github.com/likaia/nginxpulse/internal/store"
)

func hasAnyField(indexMap map[string]int, aliases []string) bool {
	for _, name := range aliases {
		if _, ok := indexMap[name]; ok {
			return true
		}
	}
	return false
}

// parseLogLine 解析单行日志
func (p *LogParser) parseLogLine(websiteID, sourceID string, line string) (*store.NginxLogRecord, error) {
	parser, err := p.getLineParserForSource(websiteID, sourceID)
	if err != nil {
		return nil, err
	}

	switch parser.parseType {
	case parseTypeCaddyJSON:
		return p.parseCaddyJSONLine(line, parser)
	default:
		return p.parseRegexLogLine(parser, line)
	}
}

func (p *LogParser) parseLogTimestamp(parser *logLineParser, line string) (time.Time, error) {
	switch parser.parseType {
	case parseTypeCaddyJSON:
		decoder := json.NewDecoder(strings.NewReader(line))
		decoder.UseNumber()
		var payload map[string]interface{}
		if err := decoder.Decode(&payload); err != nil {
			return time.Time{}, err
		}
		return parseCaddyTime(payload, parser.timeLayout)
	default:
		return p.parseRegexLogTimestamp(parser, line)
	}
}

func (p *LogParser) parseRegexLogTimestamp(parser *logLineParser, line string) (time.Time, error) {
	matches := parser.regex.FindStringSubmatch(line)
	if len(matches) == 0 {
		return time.Time{}, errors.New("日志格式不匹配")
	}
	rawTime := extractField(matches, parser.indexMap, timeAliases)
	if rawTime == "" {
		return time.Time{}, errors.New("日志缺少时间字段")
	}
	return parseLogTime(rawTime, parser.timeLayout)
}

func (p *LogParser) parseRegexLogLine(parser *logLineParser, line string) (*store.NginxLogRecord, error) {
	matches := parser.regex.FindStringSubmatch(line)
	if len(matches) == 0 {
		return nil, errors.New("日志格式不匹配")
	}

	ip := extractField(matches, parser.indexMap, ipAliases)
	rawTime := extractField(matches, parser.indexMap, timeAliases)
	statusStr := extractField(matches, parser.indexMap, statusAliases)
	urlValue := extractField(matches, parser.indexMap, urlAliases)
	method := extractField(matches, parser.indexMap, methodAliases)
	requestLine := extractField(matches, parser.indexMap, requestAliases)

	if method == "" || urlValue == "" {
		if requestLine != "" {
			parsedMethod, parsedURL, err := parseRequestLine(requestLine)
			if err != nil {
				return nil, err
			}
			if method == "" {
				method = parsedMethod
			}
			if urlValue == "" {
				urlValue = parsedURL
			}
		}
	}
	queryValue := extractField(matches, parser.indexMap, queryAliases)
	if urlValue != "" && queryValue != "" && queryValue != "-" && !strings.Contains(urlValue, "?") {
		if strings.HasPrefix(queryValue, "?") {
			urlValue += queryValue
		} else {
			urlValue += "?" + queryValue
		}
	}

	if ip == "" || rawTime == "" || statusStr == "" || urlValue == "" {
		return nil, errors.New("日志缺少必要字段")
	}

	timestamp, err := parseLogTime(rawTime, parser.timeLayout)
	if err != nil {
		return nil, err
	}

	statusCode, err := strconv.Atoi(statusStr)
	if err != nil {
		return nil, err
	}

	bytesSent := 0
	bytesStr := extractField(matches, parser.indexMap, bytesAliases)
	if bytesStr != "" && bytesStr != "-" {
		if parsed, err := strconv.Atoi(bytesStr); err == nil {
			bytesSent = parsed
		}
	}

	requestLength, requestTimeMs := extractRequestMetrics(matches, parser.indexMap)
	upstreamTimeMs := parseMillisecondsField(extractField(matches, parser.indexMap, upstreamTimeAliases), false)
	upstreamAddr := normalizeOptionalField(extractField(matches, parser.indexMap, upstreamAddrAliases))
	host := normalizeOptionalField(extractField(matches, parser.indexMap, hostAliases))
	requestID := normalizeOptionalField(extractField(matches, parser.indexMap, requestIDAliases))
	referPath := extractField(matches, parser.indexMap, refererAliases)
	userAgent := extractField(matches, parser.indexMap, userAgentAliases)
	return p.buildLogRecord(logRecordParams{
		ip:             ip,
		method:         method,
		urlValue:       urlValue,
		referer:        referPath,
		userAgent:      userAgent,
		statusCode:     statusCode,
		bytesSent:      bytesSent,
		requestLength:  requestLength,
		requestTimeMs:  requestTimeMs,
		upstreamTimeMs: upstreamTimeMs,
		upstreamAddr:   upstreamAddr,
		host:           host,
		requestID:      requestID,
		timestamp:      timestamp,
	})
}

func (p *LogParser) parseCaddyJSONLine(line string, parser *logLineParser) (*store.NginxLogRecord, error) {
	decoder := json.NewDecoder(strings.NewReader(line))
	decoder.UseNumber()

	var payload map[string]interface{}
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}

	request := getMap(payload, "request")
	headers := getMap(request, "headers")

	ip := getString(request, "remote_ip")
	if ip == "" {
		ip = getString(request, "client_ip")
	}
	if ip == "" {
		ip = getString(payload, "remote_ip")
	}

	method := getString(request, "method")
	urlValue := getString(request, "uri")

	statusCode, ok := getInt(payload, "status")
	if !ok {
		return nil, errors.New("日志缺少状态码")
	}

	bytesSent, _ := getInt(payload, "size")
	requestLength, _ := getInt(payload, "request_length")
	requestTimeMs := parseJSONDurationMs(payload)
	upstreamTimeMs := parseMillisecondsField(getString(payload, "upstream_duration"), false)
	upstreamAddr := normalizeOptionalField(getString(payload, "upstream"))
	host := normalizeOptionalField(getString(request, "host"))
	requestID := normalizeOptionalField(getString(payload, "request_id"))
	referPath := getHeader(headers, "Referer")
	userAgent := getHeader(headers, "User-Agent")

	timestamp, err := parseCaddyTime(payload, parser.timeLayout)
	if err != nil {
		return nil, err
	}

	return p.buildLogRecord(logRecordParams{
		ip:             ip,
		method:         method,
		urlValue:       urlValue,
		referer:        referPath,
		userAgent:      userAgent,
		statusCode:     statusCode,
		bytesSent:      bytesSent,
		requestLength:  requestLength,
		requestTimeMs:  requestTimeMs,
		upstreamTimeMs: upstreamTimeMs,
		upstreamAddr:   upstreamAddr,
		host:           host,
		requestID:      requestID,
		timestamp:      timestamp,
	})
}

type logRecordParams struct {
	ip             string
	method         string
	urlValue       string
	referer        string
	userAgent      string
	statusCode     int
	bytesSent      int
	requestLength  int
	requestTimeMs  int64
	upstreamTimeMs int64
	upstreamAddr   string
	host           string
	requestID      string
	timestamp      time.Time
}

func (p *LogParser) buildLogRecord(params logRecordParams) (*store.NginxLogRecord, error) {
	ip := params.ip
	method := params.method
	urlValue := params.urlValue
	referer := params.referer
	userAgent := params.userAgent
	statusCode := params.statusCode
	bytesSent := params.bytesSent
	timestamp := params.timestamp

	ip = normalizeIP(ip)
	if ip == "" || method == "" || urlValue == "" {
		return nil, errors.New("日志缺少必要字段")
	}
	if statusCode <= 0 {
		return nil, errors.New("日志缺少状态码")
	}

	cutoffTime := time.Now().AddDate(0, 0, -p.retentionDays)
	if timestamp.Before(cutoffTime) {
		return nil, errors.New("日志超过保留天数")
	}

	decodedPath, err := url.QueryUnescape(urlValue)
	if err != nil {
		decodedPath = urlValue
	}

	referPath := referer
	if referPath != "" {
		if decodedRefer, err := url.QueryUnescape(referPath); err == nil {
			referPath = decodedRefer
		}
	}

	if userAgent == "" {
		userAgent = "-"
	}

	pageviewFlag := enrich.ShouldCountAsPageView(statusCode, decodedPath, ip)
	browser, os, device := enrich.ParseUserAgent(userAgent)

	return &store.NginxLogRecord{
		ID:               0,
		IP:               ip,
		PageviewFlag:     pageviewFlag,
		Timestamp:        timestamp,
		Method:           method,
		Url:              decodedPath,
		Status:           statusCode,
		BytesSent:        bytesSent,
		RequestLength:    params.requestLength,
		RequestTimeMs:    params.requestTimeMs,
		UpstreamTimeMs:   params.upstreamTimeMs,
		UpstreamAddr:     params.upstreamAddr,
		Host:             params.host,
		RequestID:        params.requestID,
		Referer:          referPath,
		UserBrowser:      browser,
		UserOs:           os,
		UserDevice:       device,
		DomesticLocation: "",
		GlobalLocation:   "",
	}, nil
}

func normalizeIP(raw string) string {
	ip := strings.TrimSpace(raw)
	if ip == "" {
		return ip
	}
	if strings.Contains(ip, ",") {
		parts := strings.Split(ip, ",")
		if len(parts) > 0 {
			ip = strings.TrimSpace(parts[0])
		}
	}
	if strings.HasPrefix(ip, "[") {
		if end := strings.Index(ip, "]"); end > 0 {
			ip = ip[1:end]
		}
		return ip
	}
	if strings.Count(ip, ":") == 1 && strings.Contains(ip, ".") {
		if host, _, err := net.SplitHostPort(ip); err == nil {
			return host
		}
	}
	return ip
}

func normalizeLogPath(path string) string {
	cleaned := strings.TrimSpace(path)
	if cleaned == "" {
		return cleaned
	}
	cleaned = filepath.Clean(cleaned)
	if abs, err := filepath.Abs(cleaned); err == nil {
		return abs
	}
	return cleaned
}

func getMap(source map[string]interface{}, key string) map[string]interface{} {
	if source == nil {
		return nil
	}
	value, ok := source[key]
	if !ok {
		return nil
	}
	if mapped, ok := value.(map[string]interface{}); ok {
		return mapped
	}
	return nil
}

func getString(source map[string]interface{}, key string) string {
	if source == nil {
		return ""
	}
	value, ok := source[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

func getInt(source map[string]interface{}, key string) (int, bool) {
	if source == nil {
		return 0, false
	}
	value, ok := source[key]
	if !ok || value == nil {
		return 0, false
	}
	switch typed := value.(type) {
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return int(parsed), true
		}
		if parsed, err := typed.Float64(); err == nil {
			return int(parsed), true
		}
	case float64:
		return int(typed), true
	case float32:
		return int(typed), true
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case string:
		if parsed, err := strconv.Atoi(typed); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

var durationUnitPattern = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)(ns|us|µs|μs|ms|s|m|h)`)

func parseJSONDurationMs(payload map[string]interface{}) int64 {
	if payload == nil {
		return 0
	}
	if raw, ok := payload["duration_ms"]; ok {
		if value, ok := parseDurationValueMs(raw, true); ok {
			return value
		}
	}
	if raw, ok := payload["duration"]; ok {
		if value, ok := parseDurationValueMs(raw, false); ok {
			return value
		}
	}
	if raw, ok := payload["duration_ns"]; ok {
		if value, ok := parseDurationValueMs(raw, false); ok {
			return value
		}
	}
	return 0
}

func extractRequestTimeMs(matches []string, indexMap map[string]int) int64 {
	if value := extractField(matches, indexMap, []string{"request_time_ms", "request_time_msec", "duration_ms"}); value != "" {
		return parseMillisecondsField(value, true)
	}
	return parseMillisecondsField(extractField(matches, indexMap, []string{"request_time", "duration"}), false)
}

func extractRequestMetrics(matches []string, indexMap map[string]int) (int, int64) {
	requestLength := parseIntField(extractField(matches, indexMap, requestLengthAliases))
	requestTimeMs := extractRequestTimeMs(matches, indexMap)
	if requestLength > 0 || requestTimeMs > 0 {
		return requestLength, requestTimeMs
	}

	first := extractField(matches, indexMap, []string{"trace_first"})
	second := extractField(matches, indexMap, []string{"trace_second"})
	if first == "" || second == "" {
		return 0, 0
	}

	if looksLikeDurationToken(first) {
		if length, ok := parseIntToken(second); ok {
			return length, parseMillisecondsField(first, false)
		}
	}
	if looksLikeDurationToken(second) {
		if length, ok := parseIntToken(first); ok {
			return length, parseMillisecondsField(second, false)
		}
	}
	return 0, 0
}

func parseIntField(raw string) int {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "-" {
		return 0
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func parseIntToken(raw string) (int, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "-" {
		return 0, false
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed < 0 {
		return 0, false
	}
	return parsed, true
}

func looksLikeDurationToken(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "-" {
		return false
	}
	if strings.Contains(trimmed, ".") {
		return true
	}
	return durationUnitPattern.MatchString(trimmed)
}

func parseMillisecondsField(raw string, assumeMilliseconds bool) int64 {
	value, ok := parseDurationValueMs(raw, assumeMilliseconds)
	if !ok || value < 0 {
		return 0
	}
	return value
}

func parseDurationValueMs(raw interface{}, assumeMilliseconds bool) (int64, bool) {
	switch typed := raw.(type) {
	case json.Number:
		if parsed, err := typed.Float64(); err == nil {
			return scaleFloatDuration(parsed, assumeMilliseconds), true
		}
	case float64:
		return scaleFloatDuration(typed, assumeMilliseconds), true
	case float32:
		return scaleFloatDuration(float64(typed), assumeMilliseconds), true
	case int:
		return scaleFloatDuration(float64(typed), assumeMilliseconds), true
	case int64:
		return scaleFloatDuration(float64(typed), assumeMilliseconds), true
	case string:
		return parseDurationStringMs(typed, assumeMilliseconds)
	}
	return 0, false
}

func scaleFloatDuration(value float64, assumeMilliseconds bool) int64 {
	if assumeMilliseconds {
		if value < 0 {
			return 0
		}
		return int64(value)
	}
	if value > 1e9 {
		return int64(value / 1e6)
	}
	if value < 0 {
		return 0
	}
	return int64(value * 1000)
}

func parseDurationStringMs(raw string, assumeMilliseconds bool) (int64, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "-" {
		return 0, false
	}
	if strings.Contains(trimmed, ",") {
		parts := strings.Split(trimmed, ",")
		var maxValue int64
		found := false
		for _, part := range parts {
			if value, ok := parseDurationStringMs(part, assumeMilliseconds); ok {
				if !found || value > maxValue {
					maxValue = value
				}
				found = true
			}
		}
		return maxValue, found
	}
	if len(durationUnitPattern.FindAllString(trimmed, -1)) > 0 {
		var total time.Duration
		matches := durationUnitPattern.FindAllStringSubmatch(trimmed, -1)
		if strings.Join(durationUnitPattern.FindAllString(trimmed, -1), "") != trimmed {
			return 0, false
		}
		for _, match := range matches {
			value, err := strconv.ParseFloat(match[1], 64)
			if err != nil {
				return 0, false
			}
			unit := strings.ToLower(match[2])
			switch unit {
			case "ns":
				total += time.Duration(value)
			case "us", "µs", "μs":
				total += time.Duration(value * float64(time.Microsecond))
			case "ms":
				total += time.Duration(value * float64(time.Millisecond))
			case "s":
				total += time.Duration(value * float64(time.Second))
			case "m":
				total += time.Duration(value * float64(time.Minute))
			case "h":
				total += time.Duration(value * float64(time.Hour))
			default:
				return 0, false
			}
		}
		return total.Milliseconds(), true
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, false
	}
	return scaleFloatDuration(value, assumeMilliseconds), true
}

func normalizeOptionalField(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "-" {
		return ""
	}
	return trimmed
}

func getHeader(headers map[string]interface{}, name string) string {
	if headers == nil {
		return ""
	}
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			switch typed := value.(type) {
			case []interface{}:
				if len(typed) > 0 {
					return fmt.Sprint(typed[0])
				}
			case []string:
				if len(typed) > 0 {
					return typed[0]
				}
			case string:
				return typed
			default:
				return fmt.Sprint(typed)
			}
		}
	}
	return ""
}

func parseCaddyTime(payload map[string]interface{}, layout string) (time.Time, error) {
	if payload == nil {
		return time.Time{}, errors.New("日志缺少时间字段")
	}
	if value, ok := payload["ts"]; ok {
		if ts, err := parseAnyTime(value, layout); err == nil {
			return ts, nil
		}
	}
	if value, ok := payload["time"]; ok {
		if ts, err := parseAnyTime(value, layout); err == nil {
			return ts, nil
		}
	}
	if value, ok := payload["timestamp"]; ok {
		if ts, err := parseAnyTime(value, layout); err == nil {
			return ts, nil
		}
	}
	return time.Time{}, errors.New("日志缺少时间字段")
}

func parseAnyTime(value interface{}, layout string) (time.Time, error) {
	switch typed := value.(type) {
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return time.Unix(parsed, 0), nil
		}
		if parsed, err := typed.Float64(); err == nil {
			return parseFloatEpoch(parsed), nil
		}
	case float64:
		return parseFloatEpoch(typed), nil
	case float32:
		return parseFloatEpoch(float64(typed)), nil
	case int:
		return time.Unix(int64(typed), 0), nil
	case int64:
		return time.Unix(typed, 0), nil
	case string:
		return parseLogTime(typed, layout)
	}
	return time.Time{}, errors.New("时间格式不支持")
}

func parseFloatEpoch(value float64) time.Time {
	if value > 1e12 {
		value = value / 1000
	}
	sec := int64(value)
	nsec := int64((value - float64(sec)) * float64(time.Second))
	return time.Unix(sec, nsec)
}

func extractField(matches []string, indexMap map[string]int, aliases []string) string {
	for _, name := range aliases {
		if idx, ok := indexMap[name]; ok {
			if idx > 0 && idx < len(matches) {
				return matches[idx]
			}
		}
	}
	return ""
}

func parseRequestLine(line string) (string, string, error) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return "", "", errors.New("无效的 request 格式")
	}
	return parts[0], parts[1], nil
}

func parseLogTime(raw, layout string) (time.Time, error) {
	if ts, ok := parseEpochTime(raw); ok {
		return ts, nil
	}

	layouts := make([]string, 0, 3)
	if layout != "" {
		layouts = append(layouts, layout)
		if layout == defaultZoraxyTimeLayout {
			layouts = append(layouts,
				"2006-01-02 15:04:05",
				"2006-01-02 15:04:05.000",
				"2006-01-02 15:04:05.000000000",
			)
		}
	}
	layouts = append(layouts, defaultNginxTimeLayout, time.RFC3339, time.RFC3339Nano)

	var lastErr error
	for _, l := range layouts {
		parsed, err := time.Parse(l, raw)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("时间解析失败")
	}
	return time.Time{}, lastErr
}

func parseEpochTime(raw string) (time.Time, bool) {
	if raw == "" {
		return time.Time{}, false
	}

	for _, r := range raw {
		if (r < '0' || r > '9') && r != '.' {
			return time.Time{}, false
		}
	}

	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return time.Time{}, false
	}

	if value > 1e12 {
		value = value / 1000
	}

	sec := int64(value)
	nsec := int64((value - float64(sec)) * float64(time.Second))
	return time.Unix(sec, nsec), true
}

// EmptyParserResult 生成空结果
func EmptyParserResult(name, id string) ParserResult {
	return ParserResult{
		WebName:      name,
		WebID:        id,
		TotalEntries: 0,
		Duration:     0,
		Success:      true,
		Error:        nil,
	}
}
