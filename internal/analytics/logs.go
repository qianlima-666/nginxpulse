package analytics

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/likaia/nginxpulse/internal/ingest"
	"github.com/likaia/nginxpulse/internal/sqlutil"
	"github.com/likaia/nginxpulse/internal/store"
	"github.com/likaia/nginxpulse/internal/timeutil"
	"github.com/sirupsen/logrus"
)

// LogEntry 表示单条日志信息
type LogEntry struct {
	ID                     int    `json:"id"`
	IP                     string `json:"ip"`
	Timestamp              int64  `json:"timestamp"`
	Time                   string `json:"time"` // 格式化后的时间字符串
	Method                 string `json:"method"`
	URL                    string `json:"url"`
	StatusCode             int    `json:"status_code"`
	BytesSent              int    `json:"bytes_sent"`
	RequestLength          int    `json:"request_length"`
	RequestTimeMs          int64  `json:"request_time_ms"`
	UpstreamResponseTimeMs int64  `json:"upstream_response_time_ms"`
	UpstreamAddr           string `json:"upstream_addr"`
	Host                   string `json:"host"`
	RequestID              string `json:"request_id"`
	Referer                string `json:"referer"`
	UserBrowser            string `json:"user_browser"`
	UserOS                 string `json:"user_os"`
	UserDevice             string `json:"user_device"`
	DomesticLocation       string `json:"domestic_location"`
	GlobalLocation         string `json:"global_location"`
	PageviewFlag           bool   `json:"pageview_flag"`
	IsNewVisitor           bool   `json:"is_new_visitor"`
}

// LogsStats 日志查询结果
type LogsStats struct {
	Logs                               []LogEntry `json:"logs"`
	IPParsing                          bool       `json:"ip_parsing"`
	IPParsingProgress                  int        `json:"ip_parsing_progress"`
	IPParsingEstimatedTotalSeconds     int64      `json:"ip_parsing_estimated_total_seconds,omitempty"`
	IPParsingEstimatedRemainingSeconds int64      `json:"ip_parsing_estimated_remaining_seconds,omitempty"`
	IPGeoParsing                       bool       `json:"ip_geo_parsing"`
	IPGeoPending                       bool       `json:"ip_geo_pending"`
	IPGeoProgress                      int        `json:"ip_geo_progress,omitempty"`
	IPGeoEstimatedRemainingSeconds     int64      `json:"ip_geo_estimated_remaining_seconds,omitempty"`
	ParsingPending                     bool       `json:"parsing_pending"`
	ParsingPendingRange                *TimeRange `json:"parsing_pending_range,omitempty"`
	ParsingPendingProgress             int        `json:"parsing_pending_progress,omitempty"`
	Pagination                         struct {
		Total    int  `json:"total"`
		Page     int  `json:"page"`
		PageSize int  `json:"pageSize"`
		Pages    int  `json:"pages"`
		HasMore  bool `json:"hasMore"`
		Exact    bool `json:"exact"`
	} `json:"pagination"`
}

type TimeRange struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// GetType 实现 StatsResult 接口
func (s LogsStats) GetType() string {
	return "logs"
}

// LogsStatsManager 实现日志查询功能
type LogsStatsManager struct {
	repo *store.Repository
}

type logsQueryOptions struct {
	filter            string
	timeRange         string
	timeStart         int64
	timeEnd           int64
	statusCode        int
	statusClass       string
	excludeInternal   bool
	excludeSpider     bool
	excludeForeign    bool
	ipFilter          string
	locationFilter    string
	urlFilter         string
	pageviewOnly      bool
	includeNewVisitor bool
	newVisitorFilter  string
	newRangeStart     int64
	newRangeEnd       int64
}

// NewLogsStatsManager 创建日志查询管理器
func NewLogsStatsManager(userRepoPtr *store.Repository) *LogsStatsManager {
	return &LogsStatsManager{
		repo: userRepoPtr,
	}
}

// Query 实现 StatsManager 接口
func (m *LogsStatsManager) Query(query StatsQuery) (StatsResult, error) {
	result := LogsStats{}
	queryStartedAt := time.Now()
	result.IPParsing = ingest.IsIPParsing()
	result.IPParsingProgress = ingest.GetIPParsingProgress()
	result.IPParsingEstimatedTotalSeconds = ingest.GetIPParsingEstimatedTotalSeconds()
	result.IPParsingEstimatedRemainingSeconds = ingest.GetIPParsingEstimatedRemainingSeconds()
	result.IPGeoParsing = ingest.IsIPGeoParsing()
	if m.repo != nil {
		if pendingCount, err := m.repo.CountIPGeoPending(); err == nil {
			result.IPGeoPending = pendingCount > 0
			if pendingCount > 0 {
				result.IPGeoProgress = ingest.GetIPGeoParsingProgress(pendingCount)
				result.IPGeoEstimatedRemainingSeconds = ingest.GetIPGeoEstimatedRemainingSeconds(pendingCount)
			}
		}
	}

	// 从查询参数中获取分页和排序信息
	page := 1
	pageSize := 100
	sortField := "timestamp"
	sortOrder := "desc"
	var filter string
	var timeRange string
	var timeStart int64
	var timeEnd int64
	var statusCode int
	var statusClass string
	var excludeInternal bool
	var excludeSpider bool
	var excludeForeign bool
	var ipFilter string
	var locationFilter string
	var urlFilter string
	var pageviewOnly bool
	var newVisitorFilter string
	var includeNewVisitor bool
	var newRangeStart int64
	var newRangeEnd int64
	var distinctIP bool

	if pageVal, ok := query.ExtraParam["page"].(int); ok && pageVal > 0 {
		page = pageVal
	}

	if pageSizeVal, ok := query.ExtraParam["pageSize"].(int); ok && pageSizeVal > 0 {
		pageSize = pageSizeVal
		if pageSize > 1000 {
			pageSize = 1000 // 设置上限以防过大查询
		}
	}

	if field, ok := query.ExtraParam["sortField"].(string); ok && field != "" {
		// 验证字段名有效性，防止SQL注入
		validFields := map[string]bool{
			"timestamp": true, "ip": true, "url": true,
			"status_code": true, "bytes_sent": true,
			"request_length": true, "request_time_ms": true, "upstream_response_time_ms": true,
		}
		if validFields[field] {
			sortField = field
		}
	}

	if order, ok := query.ExtraParam["sortOrder"].(string); ok {
		if order == "asc" || order == "desc" {
			sortOrder = order
		}
	}

	if filterVal, ok := query.ExtraParam["filter"].(string); ok {
		filter = filterVal
	}
	if timeRangeVal, ok := query.ExtraParam["timeRange"].(string); ok {
		timeRange = timeRangeVal
	}
	if timeStartVal, ok := query.ExtraParam["timeStart"].(string); ok {
		parsed, err := parseTimeFilter(timeStartVal)
		if err != nil {
			return result, fmt.Errorf("解析开始时间失败: %v", err)
		}
		timeStart = parsed
	}
	if timeEndVal, ok := query.ExtraParam["timeEnd"].(string); ok {
		parsed, err := parseTimeFilter(timeEndVal)
		if err != nil {
			return result, fmt.Errorf("解析结束时间失败: %v", err)
		}
		timeEnd = parsed
	}
	if statusCodeVal, ok := query.ExtraParam["statusCode"].(int); ok && statusCodeVal > 0 {
		statusCode = statusCodeVal
	}
	if statusClassVal, ok := query.ExtraParam["statusClass"].(string); ok {
		statusClass = statusClassVal
	}
	if excludeInternalVal, ok := query.ExtraParam["excludeInternal"].(bool); ok {
		excludeInternal = excludeInternalVal
	}
	if excludeSpiderVal, ok := query.ExtraParam["excludeSpider"].(bool); ok {
		excludeSpider = excludeSpiderVal
	}
	if excludeForeignVal, ok := query.ExtraParam["excludeForeign"].(bool); ok {
		excludeForeign = excludeForeignVal
	}
	if ipFilterVal, ok := query.ExtraParam["ipFilter"].(string); ok {
		ipFilter = strings.TrimSpace(ipFilterVal)
	}
	if locationFilterVal, ok := query.ExtraParam["locationFilter"].(string); ok {
		locationFilter = strings.TrimSpace(locationFilterVal)
	}
	if urlFilterVal, ok := query.ExtraParam["urlFilter"].(string); ok {
		urlFilter = strings.TrimSpace(urlFilterVal)
	}
	if pageviewOnlyVal, ok := query.ExtraParam["pageviewOnly"].(bool); ok {
		pageviewOnly = pageviewOnlyVal
	}
	if newVisitorVal, ok := query.ExtraParam["newVisitor"].(string); ok && newVisitorVal != "" {
		newVisitorFilter = newVisitorVal
		includeNewVisitor = true
	}
	if distinctVal, ok := query.ExtraParam["distinctIp"].(bool); ok {
		distinctIP = distinctVal
	}
	if includeNewVisitor {
		var err error
		newRangeStart, newRangeEnd, err = resolveNewVisitorRange(timeRange, timeStart, timeEnd)
		if err != nil {
			return result, err
		}
	}

	rangeStart, rangeEnd, err := resolveQueryRange(timeRange, timeStart, timeEnd)
	if err != nil {
		return result, err
	}
	if status, ok := ingest.GetWebsiteParseStatus(query.WebsiteID); ok {
		pending, pendingRange := computeParsingPending(status, rangeStart, rangeEnd)
		result.ParsingPending = pending
		result.ParsingPendingRange = pendingRange
		if pending {
			result.ParsingPendingProgress = computePendingProgress(status, rangeStart, rangeEnd)
		}
	}

	// 计算分页
	offset := (page - 1) * pageSize
	tableName := fmt.Sprintf("%s_nginx_logs", query.WebsiteID)
	logAlias := "l"
	firstSeenJoin := fmt.Sprintf(`LEFT JOIN "%s_first_seen" fs ON fs.ip_id = %s.ip_id`, query.WebsiteID, logAlias)
	joinClause := fmt.Sprintf(`
        JOIN "%s_dim_ip" ip ON ip.id = %s.ip_id
        JOIN "%s_dim_url" u ON u.id = %s.url_id
        JOIN "%s_dim_referer" r ON r.id = %s.referer_id
        JOIN "%s_dim_ua" ua ON ua.id = %s.ua_id
        JOIN "%s_dim_location" loc ON loc.id = %s.location_id`,
		query.WebsiteID, logAlias,
		query.WebsiteID, logAlias,
		query.WebsiteID, logAlias,
		query.WebsiteID, logAlias,
		query.WebsiteID, logAlias,
	)
	column := func(name string) string {
		switch name {
		case "ip":
			return "ip.ip"
		case "url":
			return "u.url"
		case "referer":
			return "r.referer"
		case "user_browser":
			return "ua.browser"
		case "user_os":
			return "ua.os"
		case "user_device":
			return "ua.device"
		case "domestic_location":
			return "loc.domestic"
		case "global_location":
			return "loc.global"
		default:
			return fmt.Sprintf("%s.%s", logAlias, name)
		}
	}
	baseColumn := func(name string) string {
		return fmt.Sprintf("%s.%s", logAlias, name)
	}
	fastPathColumn := func(name string) string {
		return fmt.Sprintf("base.%s", name)
	}
	options := logsQueryOptions{
		filter:            filter,
		timeRange:         timeRange,
		timeStart:         timeStart,
		timeEnd:           timeEnd,
		statusCode:        statusCode,
		statusClass:       statusClass,
		excludeInternal:   excludeInternal,
		excludeSpider:     excludeSpider,
		excludeForeign:    excludeForeign,
		ipFilter:          ipFilter,
		locationFilter:    locationFilter,
		urlFilter:         urlFilter,
		pageviewOnly:      pageviewOnly,
		includeNewVisitor: includeNewVisitor,
		newVisitorFilter:  newVisitorFilter,
		newRangeStart:     newRangeStart,
		newRangeEnd:       newRangeEnd,
	}
	fullConditions, fullArgs, err := buildLogsConditions(column, options)
	if err != nil {
		return result, err
	}
	baseConditions, baseArgs, err := buildLogsConditions(baseColumn, options)
	if err != nil {
		return result, err
	}
	fastPath := canUseFastLogsPath(sortField, options, distinctIP)
	exactCount := shouldUseExactLogsCount(options)
	queryLimit := pageSize
	if !exactCount {
		queryLimit++
	}
	orderClause := buildLogsOrderClause(column(sortField), sortOrder, fmt.Sprintf("%s.id", logAlias))
	baseOrderClause := buildLogsOrderClause(baseColumn(sortField), sortOrder, baseColumn("id"))
	fastPathOrderClause := buildLogsOrderClause(fastPathColumn(sortField), sortOrder, fastPathColumn("id"))

	// 构建查询语句
	var queryBuilder strings.Builder
	var args []interface{}
	selectFields := []string{
		"id", "ip", "timestamp", "method", "url", "status_code",
		"bytes_sent", "request_length", "request_time_ms", "upstream_response_time_ms",
		"upstream_addr", "host", "request_id", "referer", "user_browser", "user_os", "user_device",
		"domestic_location", "global_location", "pageview_flag",
	}
	selectColumns := make([]string, 0, len(selectFields))
	for _, field := range selectFields {
		selectColumns = append(selectColumns, fmt.Sprintf("%s AS %s", column(field), field))
	}
	selectColumnsWithAlias := strings.Join(selectColumns, ", ")
	selectColumnsRaw := strings.Join(selectFields, ", ")

	if !distinctIP {
		if fastPath {
			baseSelectColumns := strings.Join([]string{
				baseColumn("id"),
				baseColumn("ip_id"),
				baseColumn("timestamp"),
				baseColumn("method"),
				baseColumn("url_id"),
				baseColumn("status_code"),
				baseColumn("bytes_sent"),
				baseColumn("request_length"),
				baseColumn("request_time_ms"),
				baseColumn("upstream_response_time_ms"),
				baseColumn("upstream_addr"),
				baseColumn("host"),
				baseColumn("request_id"),
				baseColumn("referer_id"),
				baseColumn("ua_id"),
				baseColumn("location_id"),
				baseColumn("pageview_flag"),
			}, ", ")
			queryBuilder.WriteString(fmt.Sprintf(`
        WITH base AS (
            SELECT
                %s
            FROM "%s" %s`,
				baseSelectColumns, tableName, logAlias))
			if len(baseConditions) > 0 {
				queryBuilder.WriteString(" WHERE ")
				queryBuilder.WriteString(strings.Join(baseConditions, " AND "))
			}
			queryBuilder.WriteString(fmt.Sprintf(" ORDER BY %s", baseOrderClause))
			queryBuilder.WriteString(" LIMIT ? OFFSET ?")
			queryBuilder.WriteString(fmt.Sprintf(`
        )
        SELECT
            %s
        FROM base
        JOIN "%s_dim_ip" ip ON ip.id = base.ip_id
        JOIN "%s_dim_url" u ON u.id = base.url_id
        JOIN "%s_dim_referer" r ON r.id = base.referer_id
        JOIN "%s_dim_ua" ua ON ua.id = base.ua_id
        JOIN "%s_dim_location" loc ON loc.id = base.location_id`,
				strings.Join([]string{
					"base.id AS id",
					"ip.ip AS ip",
					"base.timestamp AS timestamp",
					"base.method AS method",
					"u.url AS url",
					"base.status_code AS status_code",
					"base.bytes_sent AS bytes_sent",
					"base.request_length AS request_length",
					"base.request_time_ms AS request_time_ms",
					"base.upstream_response_time_ms AS upstream_response_time_ms",
					"base.upstream_addr AS upstream_addr",
					"base.host AS host",
					"base.request_id AS request_id",
					"r.referer AS referer",
					"ua.browser AS user_browser",
					"ua.os AS user_os",
					"ua.device AS user_device",
					"loc.domestic AS domestic_location",
					"loc.global AS global_location",
					"base.pageview_flag AS pageview_flag",
				}, ", "),
				query.WebsiteID,
				query.WebsiteID,
				query.WebsiteID,
				query.WebsiteID,
				query.WebsiteID,
			))
			queryBuilder.WriteString(fmt.Sprintf(" ORDER BY %s", fastPathOrderClause))
			args = append(args, baseArgs...)
		} else if includeNewVisitor {
			queryBuilder.WriteString(fmt.Sprintf(`
        SELECT
            %s,
            CASE WHEN fs.first_ts >= ? AND fs.first_ts < ? THEN 1 ELSE 0 END AS is_new_visitor
        FROM "%s" %s
        %s
        %s`,
				selectColumnsWithAlias, tableName, logAlias, joinClause, firstSeenJoin))
			args = append(args, newRangeStart, newRangeEnd)
			args = append(args, fullArgs...)
		} else {
			queryBuilder.WriteString(fmt.Sprintf(`
        SELECT
            %s
        FROM "%s" %s
        %s`,
				selectColumnsWithAlias, tableName, logAlias, joinClause))
			args = append(args, fullArgs...)
		}
	}
	if distinctIP {
		var baseQuery strings.Builder
		if includeNewVisitor {
			baseQuery.WriteString(fmt.Sprintf(`
        WITH base AS (
            SELECT
                %s,
                CASE WHEN fs.first_ts >= ? AND fs.first_ts < ? THEN 1 ELSE 0 END AS is_new_visitor
        FROM "%s" %s
        %s
        %s`,
				selectColumnsWithAlias, tableName, logAlias, joinClause, firstSeenJoin))
			if len(fullConditions) > 0 {
				baseQuery.WriteString(" WHERE ")
				baseQuery.WriteString(strings.Join(fullConditions, " AND "))
			}
			baseQuery.WriteString("\n        )")
		} else {
			baseQuery.WriteString(fmt.Sprintf(`
        WITH base AS (
            SELECT
                %s
            FROM "%s" %s
            %s`,
				selectColumnsWithAlias, tableName, logAlias, joinClause))
			if len(fullConditions) > 0 {
				baseQuery.WriteString(" WHERE ")
				baseQuery.WriteString(strings.Join(fullConditions, " AND "))
			}
			baseQuery.WriteString("\n        )")
		}

		queryBuilder.WriteString(baseQuery.String())
		outerSelect := selectColumnsRaw
		if includeNewVisitor {
			outerSelect = outerSelect + ", is_new_visitor"
		}
		queryBuilder.WriteString(fmt.Sprintf(`
        SELECT %s FROM (
            SELECT base.*, ROW_NUMBER() OVER (PARTITION BY ip ORDER BY timestamp DESC, id DESC) AS rn
            FROM base
        )
        WHERE rn = 1`, outerSelect))
		queryBuilder.WriteString(fmt.Sprintf(" ORDER BY %s", buildLogsOrderClause(sortField, sortOrder, "id")))
		queryBuilder.WriteString(" LIMIT ? OFFSET ?")
		if includeNewVisitor {
			args = append([]interface{}{newRangeStart, newRangeEnd}, fullArgs...)
		} else {
			args = append(args, fullArgs...)
		}
		args = append(args, queryLimit, offset)
	} else if !fastPath {
		if len(fullConditions) > 0 {
			queryBuilder.WriteString(" WHERE ")
			queryBuilder.WriteString(strings.Join(fullConditions, " AND "))
		}

		queryBuilder.WriteString(fmt.Sprintf(" ORDER BY %s", orderClause))

		queryBuilder.WriteString(" LIMIT ? OFFSET ?")
		args = append(args, queryLimit, offset)
	} else {
		args = append(args, queryLimit, offset)
	}

	selectStartedAt := time.Now()
	queryStr := sqlutil.ReplacePlaceholders(queryBuilder.String())
	rows, err := m.repo.GetDB().Query(queryStr, args...)
	if err != nil {
		return result, fmt.Errorf("查询日志失败: %v", err)
	}
	defer rows.Close()
	var selectDuration time.Duration

	// 处理结果
	logs := make([]LogEntry, 0)
	for rows.Next() {
		var log LogEntry
		var pageviewFlag int
		var isNewVisitor int
		var err error

		if includeNewVisitor {
			err = rows.Scan(&log.ID, &log.IP, &log.Timestamp, &log.Method, &log.URL, &log.StatusCode,
				&log.BytesSent, &log.RequestLength, &log.RequestTimeMs, &log.UpstreamResponseTimeMs,
				&log.UpstreamAddr, &log.Host, &log.RequestID, &log.Referer, &log.UserBrowser, &log.UserOS, &log.UserDevice,
				&log.DomesticLocation, &log.GlobalLocation, &pageviewFlag, &isNewVisitor)
		} else {
			err = rows.Scan(&log.ID, &log.IP, &log.Timestamp, &log.Method, &log.URL, &log.StatusCode,
				&log.BytesSent, &log.RequestLength, &log.RequestTimeMs, &log.UpstreamResponseTimeMs,
				&log.UpstreamAddr, &log.Host, &log.RequestID, &log.Referer, &log.UserBrowser, &log.UserOS, &log.UserDevice,
				&log.DomesticLocation, &log.GlobalLocation, &pageviewFlag)
		}

		if err != nil {
			return result, fmt.Errorf("解析日志行失败: %v", err)
		}

		// 处理时间
		log.Time = time.Unix(log.Timestamp, 0).Format("2006-01-02 15:04:05")

		// 处理 pageview_flag (数据库中存储为 0/1)
		log.PageviewFlag = pageviewFlag == 1
		if includeNewVisitor {
			log.IsNewVisitor = isNewVisitor == 1
		}

		logs = append(logs, log)
	}
	selectDuration = time.Since(selectStartedAt)

	var total int
	countNeedsJoin := needsLogsJoinForFilters(options) || (includeNewVisitor && newVisitorFilter != "all")
	countDuration := time.Duration(0)
	hasMore := false
	if exactCount {
		// 查询总记录数
		var countQuery strings.Builder
		if countNeedsJoin {
			countQuery.WriteString(fmt.Sprintf(`
        SELECT %s
        FROM "%s" %s
        %s
        %s`,
				countSelect(distinctIP), tableName, logAlias, joinClause, firstSeenJoin))
		} else {
			countQuery.WriteString(fmt.Sprintf(`SELECT %s FROM "%s" %s`, countSelect(distinctIP), tableName, logAlias))
		}

		countConditions := fullConditions
		countArgs := fullArgs
		if !countNeedsJoin {
			countConditions = baseConditions
			countArgs = baseArgs
		}
		if len(countConditions) > 0 {
			countQuery.WriteString(" WHERE ")
			countQuery.WriteString(strings.Join(countConditions, " AND "))
		}

		countStartedAt := time.Now()
		countQueryStr := sqlutil.ReplacePlaceholders(countQuery.String())
		err = m.repo.GetDB().QueryRow(countQueryStr, countArgs...).Scan(&total)
		if err != nil {
			return result, fmt.Errorf("获取日志总数失败: %v", err)
		}
		countDuration = time.Since(countStartedAt)
	} else if len(logs) > pageSize {
		hasMore = true
		logs = logs[:pageSize]
	}

	// 设置返回结果
	result.Logs = logs
	result.Pagination.Total = total
	result.Pagination.Page = page
	result.Pagination.PageSize = pageSize
	result.Pagination.Exact = exactCount
	if exactCount {
		result.Pagination.Pages = (total + pageSize - 1) / pageSize
		result.Pagination.HasMore = page < result.Pagination.Pages
	} else {
		result.Pagination.HasMore = hasMore
	}
	logSlowLogsQuery(query.WebsiteID, page, pageSize, sortField, sortOrder, options, distinctIP, fastPath, exactCount, !countNeedsJoin, len(logs), total, selectDuration, countDuration, time.Since(queryStartedAt))

	return result, nil
}

func countSelect(distinctIP bool) string {
	if distinctIP {
		return "COUNT(DISTINCT l.ip_id)"
	}
	return "COUNT(*)"
}

func shouldUseExactLogsCount(opts logsQueryOptions) bool {
	// 宽查询直接做精确 COUNT(*)/COUNT(DISTINCT) 很容易扫全表，优先退化为 has_more 分页。
	return opts.timeRange != "" ||
		opts.timeStart > 0 ||
		opts.timeEnd > 0 ||
		opts.filter != "" ||
		opts.ipFilter != "" ||
		opts.locationFilter != "" ||
		opts.urlFilter != "" ||
		opts.statusCode > 0 ||
		opts.includeNewVisitor
}

func buildLogsConditions(column func(string) string, opts logsQueryOptions) ([]string, []interface{}, error) {
	const botDeviceLabel = "蜘蛛"
	conditions := make([]string, 0, 8)
	args := make([]interface{}, 0, 8)

	if opts.filter != "" {
		conditions = append(conditions, fmt.Sprintf("(%s LIKE ? OR %s LIKE ? OR %s LIKE ? OR %s LIKE ?)",
			column("url"), column("ip"), column("referer"), column("domestic_location")))
		filterArg := "%" + opts.filter + "%"
		args = append(args, filterArg, filterArg, filterArg, filterArg)
	}
	if opts.timeRange != "" {
		startTime, endTime, err := timeutil.TimePeriod(opts.timeRange)
		if err != nil {
			return nil, nil, fmt.Errorf("解析时间范围失败: %v", err)
		}
		conditions = append(conditions, fmt.Sprintf("%s >= ? AND %s < ?", column("timestamp"), column("timestamp")))
		args = append(args, startTime.Unix(), endTime.Unix())
	}
	if opts.timeStart > 0 {
		conditions = append(conditions, fmt.Sprintf("%s >= ?", column("timestamp")))
		args = append(args, opts.timeStart)
	}
	if opts.timeEnd > 0 {
		conditions = append(conditions, fmt.Sprintf("%s <= ?", column("timestamp")))
		args = append(args, opts.timeEnd)
	}
	if opts.ipFilter != "" {
		conditions = append(conditions, fmt.Sprintf("%s LIKE ?", column("ip")))
		args = append(args, "%"+opts.ipFilter+"%")
	}
	if opts.locationFilter != "" {
		conditions = append(conditions, fmt.Sprintf("(%s LIKE ? OR %s LIKE ?)",
			column("domestic_location"), column("global_location")))
		locationArg := "%" + opts.locationFilter + "%"
		args = append(args, locationArg, locationArg)
	}
	if opts.urlFilter != "" {
		conditions = append(conditions, fmt.Sprintf("%s LIKE ?", column("url")))
		args = append(args, "%"+opts.urlFilter+"%")
	}
	if opts.statusCode > 0 {
		conditions = append(conditions, fmt.Sprintf("%s = ?", column("status_code")))
		args = append(args, opts.statusCode)
	} else if opts.statusClass != "" {
		switch strings.ToLower(opts.statusClass) {
		case "2xx":
			conditions = append(conditions, fmt.Sprintf("%s >= 200 AND %s < 300", column("status_code"), column("status_code")))
		case "3xx":
			conditions = append(conditions, fmt.Sprintf("%s >= 300 AND %s < 400", column("status_code"), column("status_code")))
		case "4xx":
			conditions = append(conditions, fmt.Sprintf("%s >= 400 AND %s < 500", column("status_code"), column("status_code")))
		case "5xx":
			conditions = append(conditions, fmt.Sprintf("%s >= 500 AND %s < 600", column("status_code"), column("status_code")))
		}
	}
	if opts.excludeInternal {
		internalCondition, internalArgs := buildInternalIPCondition(column("ip"))
		conditions = append(conditions, fmt.Sprintf("NOT %s", internalCondition))
		args = append(args, internalArgs...)
	}
	if opts.excludeSpider {
		conditions = append(conditions, fmt.Sprintf("%s <> ?", column("user_device")))
		args = append(args, botDeviceLabel)
	}
	if opts.excludeForeign {
		conditions = append(conditions, fmt.Sprintf("(%s = ? OR LOWER(%s) = ?)", column("global_location"), column("global_location")))
		args = append(args, "中国", "china")
	}
	if opts.pageviewOnly {
		conditions = append(conditions, fmt.Sprintf("%s = 1", column("pageview_flag")))
	}
	if opts.includeNewVisitor {
		if opts.newVisitorFilter == "new" {
			conditions = append(conditions, "fs.first_ts >= ? AND fs.first_ts < ?")
			args = append(args, opts.newRangeStart, opts.newRangeEnd)
		} else if opts.newVisitorFilter == "returning" {
			conditions = append(conditions, "fs.first_ts < ?")
			args = append(args, opts.newRangeStart)
		}
	}

	return conditions, args, nil
}

func needsLogsJoinForFilters(opts logsQueryOptions) bool {
	return opts.filter != "" ||
		opts.ipFilter != "" ||
		opts.locationFilter != "" ||
		opts.urlFilter != "" ||
		opts.excludeInternal ||
		opts.excludeSpider ||
		opts.excludeForeign
}

func canUseFastLogsPath(sortField string, opts logsQueryOptions, distinctIP bool) bool {
	if distinctIP || opts.includeNewVisitor || needsLogsJoinForFilters(opts) {
		return false
	}
	switch sortField {
	case "timestamp", "status_code", "bytes_sent", "request_length", "request_time_ms", "upstream_response_time_ms":
		return true
	default:
		return false
	}
}

func buildLogsOrderClause(primary, sortOrder, secondary string) string {
	if secondary == "" || secondary == primary {
		return fmt.Sprintf("%s %s", primary, sortOrder)
	}
	return fmt.Sprintf("%s %s, %s %s", primary, sortOrder, secondary, sortOrder)
}

func logSlowLogsQuery(
	websiteID string,
	page int,
	pageSize int,
	sortField string,
	sortOrder string,
	opts logsQueryOptions,
	distinctIP bool,
	fastPath bool,
	exactCount bool,
	countFastPath bool,
	rows int,
	total int,
	selectDuration time.Duration,
	countDuration time.Duration,
	totalDuration time.Duration,
) {
	if totalDuration <= 300*time.Millisecond {
		return
	}

	fields := logrus.Fields{
		"website_id":       websiteID,
		"page":             page,
		"page_size":        pageSize,
		"sort_field":       sortField,
		"sort_order":       sortOrder,
		"distinct_ip":      distinctIP,
		"fast_path":        fastPath,
		"count_exact":      exactCount,
		"count_fast_path":  countFastPath,
		"rows":             rows,
		"total":            total,
		"select_ms":        selectDuration.Milliseconds(),
		"count_ms":         countDuration.Milliseconds(),
		"total_ms":         totalDuration.Milliseconds(),
		"time_range":       opts.timeRange,
		"time_start":       opts.timeStart,
		"time_end":         opts.timeEnd,
		"status_code":      opts.statusCode,
		"status_class":     opts.statusClass,
		"pageview_only":    opts.pageviewOnly,
		"exclude_internal": opts.excludeInternal,
		"exclude_spider":   opts.excludeSpider,
		"exclude_foreign":  opts.excludeForeign,
		"new_visitor":      opts.newVisitorFilter,
	}
	if opts.filter != "" {
		fields["filter"] = opts.filter
	}
	if opts.ipFilter != "" {
		fields["ip_filter"] = opts.ipFilter
	}
	if opts.locationFilter != "" {
		fields["location_filter"] = opts.locationFilter
	}
	if opts.urlFilter != "" {
		fields["url_filter"] = opts.urlFilter
	}

	logrus.WithFields(fields).Warn("日志列表查询耗时较高")
}

func parseTimeFilter(value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	if unixValue, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		if unixValue > 1_000_000_000_000 {
			return unixValue / 1000, nil
		}
		return unixValue, nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, trimmed, time.Local)
		if err == nil {
			return parsed.Unix(), nil
		}
	}
	return 0, fmt.Errorf("不支持的时间格式")
}

func resolveNewVisitorRange(timeRange string, timeStart, timeEnd int64) (int64, int64, error) {
	if timeStart > 0 && timeEnd > 0 {
		return timeStart, timeEnd, nil
	}
	if timeRange != "" {
		startTime, endTime, err := timeutil.TimePeriod(timeRange)
		if err != nil {
			return 0, 0, fmt.Errorf("解析时间范围失败: %v", err)
		}
		return startTime.Unix(), endTime.Unix(), nil
	}
	if timeStart > 0 {
		return timeStart, time.Now().Unix(), nil
	}
	if timeEnd > 0 {
		return 0, timeEnd, nil
	}
	return 0, 0, nil
}

func resolveQueryRange(timeRange string, timeStart, timeEnd int64) (int64, int64, error) {
	var rangeStart int64
	var rangeEnd int64
	if timeRange != "" {
		startTime, endTime, err := timeutil.TimePeriod(timeRange)
		if err != nil {
			return 0, 0, fmt.Errorf("解析时间范围失败: %v", err)
		}
		rangeStart = startTime.Unix()
		rangeEnd = endTime.Unix()
	}
	if timeStart > 0 {
		if rangeStart == 0 || timeStart > rangeStart {
			rangeStart = timeStart
		}
	}
	if timeEnd > 0 {
		if rangeEnd == 0 || timeEnd < rangeEnd {
			rangeEnd = timeEnd
		}
	}
	return rangeStart, rangeEnd, nil
}

func computeParsingPending(
	status ingest.WebsiteParseStatus,
	rangeStart, rangeEnd int64,
) (bool, *TimeRange) {
	logMin := status.LogMinTs
	logMax := status.LogMaxTs
	if logMin <= 0 || logMax <= 0 || logMax < logMin {
		return false, nil
	}

	if rangeStart <= 0 {
		rangeStart = logMin
	}
	if rangeEnd <= 0 {
		rangeEnd = logMax
	}

	if rangeEnd < logMin || rangeStart > logMax {
		return false, nil
	}

	if len(status.ParsedHourBuckets) > 0 {
		progress := computeBucketProgress(status.ParsedHourBuckets, logMin, logMax, rangeStart, rangeEnd)
		if progress >= 100 {
			return false, nil
		}
	}

	parsedMin := status.ParsedMinTs
	parsedMax := status.ParsedMaxTs
	if parsedMin == 0 && status.RecentCutoffTs > 0 {
		parsedMin = status.RecentCutoffTs
	}

	if parsedMin == 0 && parsedMax == 0 {
		return true, &TimeRange{Start: rangeStart, End: rangeEnd}
	}

	pending := false
	pendingStart := int64(0)
	pendingEnd := int64(0)

	if parsedMin > 0 && rangeStart < parsedMin {
		pending = true
		pendingStart = maxInt64(rangeStart, logMin)
		pendingEnd = minInt64(rangeEnd, parsedMin-1)
	}
	if parsedMax > 0 && rangeEnd > parsedMax {
		pending = true
		pendingStart = maxInt64(pendingStart, maxInt64(parsedMax+1, rangeStart))
		pendingEnd = maxInt64(pendingEnd, minInt64(rangeEnd, logMax))
	}

	if !pending {
		return false, nil
	}
	if pendingStart == 0 {
		pendingStart = rangeStart
	}
	if pendingEnd == 0 {
		pendingEnd = rangeEnd
	}
	if pendingEnd < pendingStart {
		return true, nil
	}
	return true, &TimeRange{Start: pendingStart, End: pendingEnd}
}

func computeParsingProgress(
	status ingest.WebsiteParseStatus,
	rangeStart, rangeEnd int64,
) int {
	logMin := status.LogMinTs
	logMax := status.LogMaxTs
	if logMin <= 0 || logMax <= 0 || logMax < logMin {
		return 0
	}

	if rangeStart <= 0 {
		rangeStart = logMin
	}
	if rangeEnd <= 0 {
		rangeEnd = logMax
	}
	if rangeEnd < rangeStart {
		return 0
	}

	rangeStart = maxInt64(rangeStart, logMin)
	rangeEnd = minInt64(rangeEnd, logMax)
	if rangeEnd < rangeStart {
		return 0
	}

	parsedMin := status.ParsedMinTs
	parsedMax := status.ParsedMaxTs
	if parsedMin == 0 && status.RecentCutoffTs > 0 {
		parsedMin = status.RecentCutoffTs
	}
	if parsedMin == 0 && parsedMax == 0 {
		return 0
	}
	if parsedMin == 0 {
		parsedMin = parsedMax
	}
	if parsedMax == 0 {
		parsedMax = parsedMin
	}
	if parsedMin > parsedMax {
		parsedMin, parsedMax = parsedMax, parsedMin
	}

	coverStart := maxInt64(rangeStart, parsedMin)
	coverEnd := minInt64(rangeEnd, parsedMax)
	coveredLen := int64(0)
	if coverEnd >= coverStart {
		coveredLen = coverEnd - coverStart + 1
	}

	rangeLen := rangeEnd - rangeStart + 1
	if rangeLen <= 0 {
		return 0
	}

	progress := int((coveredLen * 100) / rangeLen)
	if progress < 0 {
		return 0
	}
	if progress > 100 {
		return 100
	}
	return progress
}

func computePendingProgress(
	status ingest.WebsiteParseStatus,
	rangeStart, rangeEnd int64,
) int {
	if len(status.ParsedHourBuckets) > 0 {
		progress := computeBucketProgress(status.ParsedHourBuckets, status.LogMinTs, status.LogMaxTs, rangeStart, rangeEnd)
		if progress >= 0 {
			return progress
		}
	}
	if status.BackfillTotalBytes > 0 {
		processed := status.BackfillProcessedBytes
		if processed < 0 {
			processed = 0
		}
		if processed > status.BackfillTotalBytes {
			processed = status.BackfillTotalBytes
		}
		progress := int((processed * 100) / status.BackfillTotalBytes)
		if progress < 0 {
			return 0
		}
		if progress > 100 {
			return 100
		}
		return progress
	}
	return computeParsingProgress(status, rangeStart, rangeEnd)
}

func computeBucketProgress(buckets map[int64]bool, logMin, logMax, rangeStart, rangeEnd int64) int {
	if len(buckets) == 0 {
		return -1
	}
	if logMin <= 0 || logMax <= 0 || logMax < logMin {
		return -1
	}
	if rangeStart <= 0 {
		rangeStart = logMin
	}
	if rangeEnd <= 0 {
		rangeEnd = logMax
	}
	rangeStart = maxInt64(rangeStart, logMin)
	rangeEnd = minInt64(rangeEnd, logMax)
	if rangeEnd < rangeStart {
		return -1
	}

	startHour := (rangeStart / 3600) * 3600
	endHour := (rangeEnd / 3600) * 3600
	if endHour < startHour {
		return -1
	}

	totalBuckets := int64((endHour-startHour)/3600 + 1)
	if totalBuckets <= 0 {
		return -1
	}

	parsedBuckets := int64(0)
	for bucket := startHour; bucket <= endHour; bucket += 3600 {
		if buckets[bucket] {
			parsedBuckets++
		}
	}

	progress := int((parsedBuckets * 100) / totalBuckets)
	if progress < 0 {
		return 0
	}
	if progress > 100 {
		return 100
	}
	return progress
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func buildInternalIPCondition(column string) (string, []interface{}) {
	patterns := []string{
		"10.%", "127.%", "192.168.%",
		"172.16.%", "172.17.%", "172.18.%", "172.19.%", "172.20.%",
		"172.21.%", "172.22.%", "172.23.%", "172.24.%", "172.25.%",
		"172.26.%", "172.27.%", "172.28.%", "172.29.%", "172.30.%", "172.31.%",
		"fc%", "fd%", "fe80:%", "::1",
	}

	clauses := make([]string, 0, len(patterns))
	args := make([]interface{}, 0, len(patterns))
	for _, pattern := range patterns {
		if pattern == "::1" {
			clauses = append(clauses, fmt.Sprintf("%s = ?", column))
		} else {
			clauses = append(clauses, fmt.Sprintf("%s LIKE ?", column))
		}
		args = append(args, pattern)
	}
	return fmt.Sprintf("(%s)", strings.Join(clauses, " OR ")), args
}
