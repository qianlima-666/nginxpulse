package analytics

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/likaia/nginxpulse/internal/sqlutil"
	"github.com/likaia/nginxpulse/internal/store"
	"github.com/likaia/nginxpulse/internal/timeutil"
)

const (
	developerTrendDays       = 7
	developerSlowThresholdMs = int64(1000)
	developerIssueAvgSlowMs  = 500.0
	developerIssueLimit      = 8
)

type DeveloperMetric struct {
	Current       float64  `json:"current"`
	Previous      float64  `json:"previous"`
	Delta         float64  `json:"delta"`
	ChangeRate    *float64 `json:"changeRate"`
	ShareCurrent  *float64 `json:"shareCurrent,omitempty"`
	SharePrevious *float64 `json:"sharePrevious,omitempty"`
}

type DeveloperDailySummary struct {
	TotalRequests       int64           `json:"totalRequests"`
	AvgRequestSizeBytes DeveloperMetric `json:"avgRequestSizeBytes"`
	Status5xx           DeveloperMetric `json:"status5xx"`
	Status4xx           DeveloperMetric `json:"status4xx"`
	AvgRequestTimeMs    DeveloperMetric `json:"avgRequestTimeMs"`
	AvgUpstreamTimeMs   DeveloperMetric `json:"avgUpstreamTimeMs"`
	SlowRequests        DeveloperMetric `json:"slowRequests"`
	SlowRequestRate     DeveloperMetric `json:"slowRequestRate"`
}

type DeveloperDailyTrend struct {
	Labels            []string  `json:"labels"`
	Status4xx         []int64   `json:"status4xx"`
	Status5xx         []int64   `json:"status5xx"`
	AvgRequestTimeMs  []float64 `json:"avgRequestTimeMs"`
	AvgUpstreamTimeMs []float64 `json:"avgUpstreamTimeMs"`
	SlowRequestRate   []float64 `json:"slowRequestRate"`
}

type DeveloperDailyURLIssue struct {
	URL                   string `json:"url"`
	Requests              int64  `json:"requests"`
	Errors5xx             int64  `json:"errors5xx"`
	Errors5xxDelta        int64  `json:"errors5xxDelta"`
	SlowRequests          int64  `json:"slowRequests"`
	AvgRequestTimeMs      int64  `json:"avgRequestTimeMs"`
	AvgRequestTimeDeltaMs int64  `json:"avgRequestTimeDeltaMs"`
	MaxRequestTimeMs      int64  `json:"maxRequestTimeMs"`
}

type DeveloperDailyStats struct {
	CurrentDate     string                   `json:"currentDate"`
	PreviousDate    string                   `json:"previousDate"`
	SlowThresholdMs int64                    `json:"slowThresholdMs"`
	Summary         DeveloperDailySummary    `json:"summary"`
	Trend           DeveloperDailyTrend      `json:"trend"`
	TopIssues       []DeveloperDailyURLIssue `json:"topIssues"`
}

func (s DeveloperDailyStats) GetType() string {
	return "developer_daily"
}

type DeveloperDailyStatsManager struct {
	repo *store.Repository
}

func NewDeveloperDailyStatsManager(userRepoPtr *store.Repository) *DeveloperDailyStatsManager {
	return &DeveloperDailyStatsManager{
		repo: userRepoPtr,
	}
}

type developerDaySnapshot struct {
	totalRequests       int64
	status4xx           int64
	status5xx           int64
	slowRequests        int64
	timedRequests       int64
	upstreamTimed       int64
	sizeRequests        int64
	avgRequestTimeMs    float64
	avgUpstreamTimeMs   float64
	avgRequestSizeBytes float64
}

type developerIssueRow struct {
	url              string
	requests         int64
	errors5xx        int64
	slowRequests     int64
	avgRequestTimeMs float64
	maxRequestTimeMs int64
}

func (m *DeveloperDailyStatsManager) Query(query StatsQuery) (StatsResult, error) {
	result := DeveloperDailyStats{
		SlowThresholdMs: developerSlowThresholdMs,
		Summary: DeveloperDailySummary{
			AvgRequestSizeBytes: DeveloperMetric{},
			Status5xx:           DeveloperMetric{},
			Status4xx:           DeveloperMetric{},
			AvgRequestTimeMs:    DeveloperMetric{},
			AvgUpstreamTimeMs:   DeveloperMetric{},
			SlowRequests:        DeveloperMetric{},
			SlowRequestRate:     DeveloperMetric{},
		},
		Trend: DeveloperDailyTrend{
			Labels:            make([]string, 0, developerTrendDays),
			Status4xx:         make([]int64, 0, developerTrendDays),
			Status5xx:         make([]int64, 0, developerTrendDays),
			AvgRequestTimeMs:  make([]float64, 0, developerTrendDays),
			AvgUpstreamTimeMs: make([]float64, 0, developerTrendDays),
			SlowRequestRate:   make([]float64, 0, developerTrendDays),
		},
		TopIssues: make([]DeveloperDailyURLIssue, 0),
	}

	timeRange, ok := query.ExtraParam["timeRange"].(string)
	if !ok || timeRange == "" {
		return result, fmt.Errorf("timeRange 参数缺失")
	}

	startTime, _, err := timeutil.TimePeriod(timeRange)
	if err != nil {
		return result, fmt.Errorf("解析时间范围失败: %v", err)
	}
	currentStart := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
	currentEnd := currentStart.AddDate(0, 0, 1)

	prevStart, prevEnd := previousTimeRange(timeRange)
	if prevStart.IsZero() || prevEnd.IsZero() {
		prevStart = currentStart.AddDate(0, 0, -1)
		prevEnd = prevStart.AddDate(0, 0, 1).Add(-time.Second)
	}
	prevDayStart := time.Date(prevStart.Year(), prevStart.Month(), prevStart.Day(), 0, 0, 0, 0, prevStart.Location())
	prevDayEnd := prevDayStart.AddDate(0, 0, 1)

	result.CurrentDate = currentStart.Format("2006-01-02")
	result.PreviousDate = prevDayStart.Format("2006-01-02")

	currentSnapshot, err := m.queryDaySnapshot(query.WebsiteID, currentStart, currentEnd)
	if err != nil {
		return result, err
	}
	prevSnapshot, err := m.queryDaySnapshot(query.WebsiteID, prevDayStart, prevDayEnd)
	if err != nil {
		return result, err
	}

	result.Summary.TotalRequests = currentSnapshot.totalRequests
	result.Summary.AvgRequestSizeBytes = buildDeveloperMetric(currentSnapshot.avgRequestSizeBytes, prevSnapshot.avgRequestSizeBytes)
	result.Summary.Status5xx = buildDeveloperMetric(float64(currentSnapshot.status5xx), float64(prevSnapshot.status5xx))
	result.Summary.Status5xx.ShareCurrent = ratioPointer(currentSnapshot.status5xx, currentSnapshot.totalRequests)
	result.Summary.Status5xx.SharePrevious = ratioPointer(prevSnapshot.status5xx, prevSnapshot.totalRequests)
	result.Summary.Status4xx = buildDeveloperMetric(float64(currentSnapshot.status4xx), float64(prevSnapshot.status4xx))
	result.Summary.Status4xx.ShareCurrent = ratioPointer(currentSnapshot.status4xx, currentSnapshot.totalRequests)
	result.Summary.Status4xx.SharePrevious = ratioPointer(prevSnapshot.status4xx, prevSnapshot.totalRequests)
	result.Summary.AvgRequestTimeMs = buildDeveloperMetric(currentSnapshot.avgRequestTimeMs, prevSnapshot.avgRequestTimeMs)
	result.Summary.AvgUpstreamTimeMs = buildDeveloperMetric(currentSnapshot.avgUpstreamTimeMs, prevSnapshot.avgUpstreamTimeMs)
	result.Summary.SlowRequests = buildDeveloperMetric(float64(currentSnapshot.slowRequests), float64(prevSnapshot.slowRequests))
	result.Summary.SlowRequests.ShareCurrent = ratioPointer(currentSnapshot.slowRequests, currentSnapshot.totalRequests)
	result.Summary.SlowRequests.SharePrevious = ratioPointer(prevSnapshot.slowRequests, prevSnapshot.totalRequests)
	result.Summary.SlowRequestRate = buildDeveloperMetric(
		ratioValue(currentSnapshot.slowRequests, currentSnapshot.totalRequests),
		ratioValue(prevSnapshot.slowRequests, prevSnapshot.totalRequests),
	)

	for offset := developerTrendDays - 1; offset >= 0; offset-- {
		dayStart := currentStart.AddDate(0, 0, -offset)
		dayEnd := dayStart.AddDate(0, 0, 1)
		snapshot, err := m.queryDaySnapshot(query.WebsiteID, dayStart, dayEnd)
		if err != nil {
			return result, err
		}
		result.Trend.Labels = append(result.Trend.Labels, timeutil.FormatDateWithWeekday(dayStart, true))
		result.Trend.Status4xx = append(result.Trend.Status4xx, snapshot.status4xx)
		result.Trend.Status5xx = append(result.Trend.Status5xx, snapshot.status5xx)
		result.Trend.AvgRequestTimeMs = append(result.Trend.AvgRequestTimeMs, snapshot.avgRequestTimeMs)
		result.Trend.AvgUpstreamTimeMs = append(result.Trend.AvgUpstreamTimeMs, snapshot.avgUpstreamTimeMs)
		result.Trend.SlowRequestRate = append(result.Trend.SlowRequestRate, ratioValue(snapshot.slowRequests, snapshot.totalRequests))
	}

	topIssues, err := m.queryTopIssues(query.WebsiteID, currentStart, currentEnd, prevDayStart, prevDayEnd)
	if err != nil {
		return result, err
	}
	result.TopIssues = topIssues

	return result, nil
}

func (m *DeveloperDailyStatsManager) queryDaySnapshot(
	websiteID string,
	startTime, endTime time.Time,
) (developerDaySnapshot, error) {
	result := developerDaySnapshot{}
	tableName := fmt.Sprintf(`"%s_nginx_logs"`, websiteID)

	query := sqlutil.ReplacePlaceholders(fmt.Sprintf(`
		SELECT
			COUNT(*) AS total_requests,
			SUM(CASE WHEN status_code >= 400 AND status_code < 500 THEN 1 ELSE 0 END) AS status_4xx,
			SUM(CASE WHEN status_code >= 500 AND status_code < 600 THEN 1 ELSE 0 END) AS status_5xx,
			SUM(CASE WHEN request_time_ms >= ? THEN 1 ELSE 0 END) AS slow_requests,
			SUM(CASE WHEN request_time_ms > 0 THEN 1 ELSE 0 END) AS timed_requests,
			SUM(CASE WHEN upstream_response_time_ms > 0 THEN 1 ELSE 0 END) AS upstream_timed_requests,
			SUM(CASE WHEN request_length > 0 THEN 1 ELSE 0 END) AS size_requests,
			COALESCE(AVG(NULLIF(request_time_ms, 0)), 0) AS avg_request_time_ms,
			COALESCE(AVG(NULLIF(upstream_response_time_ms, 0)), 0) AS avg_upstream_time_ms,
			COALESCE(AVG(NULLIF(request_length, 0)), 0) AS avg_request_size_bytes
		FROM %s
		WHERE timestamp >= ? AND timestamp < ?`,
		tableName))

	var avgRequestTimeMs sql.NullFloat64
	var avgUpstreamTimeMs sql.NullFloat64
	var avgRequestSizeBytes sql.NullFloat64
	row := m.repo.GetDB().QueryRow(query, developerSlowThresholdMs, startTime.Unix(), endTime.Unix())
	if err := row.Scan(
		&result.totalRequests,
		&result.status4xx,
		&result.status5xx,
		&result.slowRequests,
		&result.timedRequests,
		&result.upstreamTimed,
		&result.sizeRequests,
		&avgRequestTimeMs,
		&avgUpstreamTimeMs,
		&avgRequestSizeBytes,
	); err != nil {
		return result, fmt.Errorf("查询开发者日报摘要失败: %v", err)
	}

	result.avgRequestTimeMs = nullableFloat(avgRequestTimeMs)
	result.avgUpstreamTimeMs = nullableFloat(avgUpstreamTimeMs)
	result.avgRequestSizeBytes = nullableFloat(avgRequestSizeBytes)
	return result, nil
}

func (m *DeveloperDailyStatsManager) queryTopIssues(
	websiteID string,
	currentStart, currentEnd time.Time,
	prevStart, prevEnd time.Time,
) ([]DeveloperDailyURLIssue, error) {
	tableName := fmt.Sprintf(`"%s_nginx_logs"`, websiteID)
	query := sqlutil.ReplacePlaceholders(fmt.Sprintf(`
		SELECT
			url,
			COUNT(*) AS requests,
			SUM(CASE WHEN status_code >= 500 AND status_code < 600 THEN 1 ELSE 0 END) AS errors_5xx,
			SUM(CASE WHEN request_time_ms >= ? THEN 1 ELSE 0 END) AS slow_requests,
			COALESCE(AVG(NULLIF(request_time_ms, 0)), 0) AS avg_request_time_ms,
			COALESCE(MAX(request_time_ms), 0) AS max_request_time_ms
		FROM %s
		WHERE timestamp >= ? AND timestamp < ?
		GROUP BY url
		HAVING
			SUM(CASE WHEN status_code >= 500 AND status_code < 600 THEN 1 ELSE 0 END) > 0
			OR SUM(CASE WHEN request_time_ms >= ? THEN 1 ELSE 0 END) > 0
			OR COALESCE(AVG(NULLIF(request_time_ms, 0)), 0) >= ?
		ORDER BY errors_5xx DESC, slow_requests DESC, avg_request_time_ms DESC, requests DESC
		LIMIT ?`,
		tableName))

	rows, err := m.repo.GetDB().Query(
		query,
		developerSlowThresholdMs,
		currentStart.Unix(),
		currentEnd.Unix(),
		developerSlowThresholdMs,
		developerIssueAvgSlowMs,
		developerIssueLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("查询开发者日报问题 URL 失败: %v", err)
	}
	defer rows.Close()

	issues := make([]developerIssueRow, 0, developerIssueLimit)
	for rows.Next() {
		item, err := scanDeveloperIssueRow(rows)
		if err != nil {
			return nil, fmt.Errorf("解析开发者日报问题 URL 失败: %v", err)
		}
		issues = append(issues, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历开发者日报问题 URL 失败: %v", err)
	}

	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].errors5xx != issues[j].errors5xx {
			return issues[i].errors5xx > issues[j].errors5xx
		}
		if issues[i].slowRequests != issues[j].slowRequests {
			return issues[i].slowRequests > issues[j].slowRequests
		}
		if issues[i].avgRequestTimeMs != issues[j].avgRequestTimeMs {
			return issues[i].avgRequestTimeMs > issues[j].avgRequestTimeMs
		}
		return issues[i].requests > issues[j].requests
	})

	results := make([]DeveloperDailyURLIssue, 0, len(issues))
	for _, item := range issues {
		prevItem, err := m.queryIssueForURL(websiteID, prevStart, prevEnd, item.url)
		if err != nil {
			return nil, err
		}
		results = append(results, DeveloperDailyURLIssue{
			URL:                   item.url,
			Requests:              item.requests,
			Errors5xx:             item.errors5xx,
			Errors5xxDelta:        item.errors5xx - prevItem.errors5xx,
			SlowRequests:          item.slowRequests,
			AvgRequestTimeMs:      roundFloat(item.avgRequestTimeMs),
			AvgRequestTimeDeltaMs: roundFloat(item.avgRequestTimeMs - prevItem.avgRequestTimeMs),
			MaxRequestTimeMs:      item.maxRequestTimeMs,
		})
	}

	return results, nil
}

func (m *DeveloperDailyStatsManager) queryIssueForURL(
	websiteID string,
	startTime, endTime time.Time,
	url string,
) (developerIssueRow, error) {
	result := developerIssueRow{}
	if url == "" {
		return result, nil
	}
	tableName := fmt.Sprintf(`"%s_nginx_logs"`, websiteID)
	query := sqlutil.ReplacePlaceholders(fmt.Sprintf(`
		SELECT
			url,
			COUNT(*) AS requests,
			SUM(CASE WHEN status_code >= 500 AND status_code < 600 THEN 1 ELSE 0 END) AS errors_5xx,
			SUM(CASE WHEN request_time_ms >= ? THEN 1 ELSE 0 END) AS slow_requests,
			COALESCE(AVG(NULLIF(request_time_ms, 0)), 0) AS avg_request_time_ms,
			COALESCE(MAX(request_time_ms), 0) AS max_request_time_ms
		FROM %s
		WHERE timestamp >= ? AND timestamp < ? AND url = ?
		GROUP BY url`,
		tableName))

	row := m.repo.GetDB().QueryRow(query, developerSlowThresholdMs, startTime.Unix(), endTime.Unix(), url)
	item, err := scanDeveloperIssueRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return result, nil
		}
		return result, fmt.Errorf("查询开发者日报问题 URL 对比失败: %v", err)
	}
	return item, nil
}

type developerIssueScanner interface {
	Scan(dest ...interface{}) error
}

func scanDeveloperIssueRow(scanner developerIssueScanner) (developerIssueRow, error) {
	item := developerIssueRow{}
	var avgRequestTimeMs sql.NullFloat64
	err := scanner.Scan(
		&item.url,
		&item.requests,
		&item.errors5xx,
		&item.slowRequests,
		&avgRequestTimeMs,
		&item.maxRequestTimeMs,
	)
	if err != nil {
		return item, err
	}
	item.avgRequestTimeMs = nullableFloat(avgRequestTimeMs)
	return item, nil
}

func buildDeveloperMetric(current, previous float64) DeveloperMetric {
	metric := DeveloperMetric{
		Current:  current,
		Previous: previous,
		Delta:    current - previous,
	}
	if previous == 0 {
		if current == 0 {
			zero := 0.0
			metric.ChangeRate = &zero
		}
		return metric
	}
	changeRate := (current - previous) / previous
	metric.ChangeRate = &changeRate
	return metric
}

func ratioPointer(value, total int64) *float64 {
	if total <= 0 {
		return nil
	}
	ratio := float64(value) / float64(total)
	return &ratio
}

func ratioValue(value, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(value) / float64(total)
}

func nullableFloat(value sql.NullFloat64) float64 {
	if !value.Valid {
		return 0
	}
	return value.Float64
}

func roundFloat(value float64) int64 {
	return int64(math.Round(value))
}
