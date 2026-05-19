package analytics

import (
	"fmt"
	"strings"
	"time"

	"github.com/likaia/nginxpulse/internal/sqlutil"
	"github.com/likaia/nginxpulse/internal/store"
	"github.com/likaia/nginxpulse/internal/timeutil"
)

type StatPoint struct {
	PV int `json:"pv"` // 页面浏览量
	UV int `json:"uv"` // 独立访客数
}

type TimeSeriesStats struct {
	Labels    []string `json:"labels"`
	Visitors  []int    `json:"visitors"`
	Pageviews []int    `json:"pageviews"`
	PvMinusUv []int    `json:"pvMinusUv"` // PV - UV
}

// TimeSeriesStats 实现 StatsResult 接口
func (s TimeSeriesStats) GetType() string {
	return "timeseries"
}

type TimeSeriesStatsManager struct {
	repo *store.Repository
}

// NewTimeSeriesStatsManager 创建一个新的 TimeSeriesStatsManager 实例
func NewTimeSeriesStatsManager(userRepoPtr *store.Repository) *TimeSeriesStatsManager {
	return &TimeSeriesStatsManager{
		repo: userRepoPtr,
	}
}

// 实现 StatsManager 接口
func (s *TimeSeriesStatsManager) Query(query StatsQuery) (StatsResult, error) {
	timeRange := query.ExtraParam["timeRange"].(string)
	viewType := query.ExtraParam["viewType"].(string)
	urlFilter := ""
	if value, ok := query.ExtraParam["urlFilter"].(string); ok {
		urlFilter = strings.TrimSpace(value)
	}
	timePoints, labels := timeutil.TimePointsAndLabels(timeRange, viewType)
	result := TimeSeriesStats{
		Labels:    labels,
		Visitors:  make([]int, len(timePoints)),
		Pageviews: make([]int, len(timePoints)),
		PvMinusUv: make([]int, len(timePoints)),
	}

	statPoints, err := s.statsByTimePointsForWebsite(query.WebsiteID, timePoints, viewType, urlFilter)
	if err != nil {
		return result, fmt.Errorf("获取图表数据失败: %v", err)
	}
	for i, point := range statPoints {
		result.Pageviews[i] = point.PV
		result.Visitors[i] = point.UV
		result.PvMinusUv[i] = point.PV - point.UV
	}

	return result, nil
}

// statsByTimePointsForWebsite 根据多个时间点批量查询统计数据
func (s *TimeSeriesStatsManager) statsByTimePointsForWebsite(
	websiteID string, timePoints []time.Time, viewType string, urlFilter string) ([]StatPoint, error) {

	timePointsSize := len(timePoints)
	results := make([]StatPoint, timePointsSize)

	if timePointsSize == 0 {
		return results, nil
	}

	if urlFilter != "" {
		return s.statsByLogBuckets(websiteID, timePoints, viewType, results, urlFilter)
	}

	if viewType == "hourly" {
		return s.statsByHourlyBuckets(websiteID, timePoints, results)
	}

	return s.statsByDailyBuckets(websiteID, timePoints, results)
}

func (s *TimeSeriesStatsManager) statsByLogBuckets(
	websiteID string,
	timePoints []time.Time,
	viewType string,
	results []StatPoint,
	urlFilter string,
) ([]StatPoint, error) {
	bucketIndex := make(map[interface{}]int, len(timePoints))
	for i, point := range timePoints {
		if viewType == "hourly" {
			bucketIndex[hourBucket(point)] = i
		} else {
			bucketIndex[dayBucket(point)] = i
		}
	}

	startTime := timePoints[0]
	endTime := timePoints[len(timePoints)-1]
	if viewType == "hourly" {
		endTime = endTime.Add(time.Hour)
	} else {
		endTime = endTime.AddDate(0, 0, 1)
	}

	query := sqlutil.ReplacePlaceholders(fmt.Sprintf(`
        SELECT l.timestamp, l.ip_id
        FROM "%[1]s_nginx_logs" l
        JOIN "%[1]s_dim_url" u ON u.id = l.url_id
        WHERE l.pageview_flag = 1 AND l.timestamp >= ? AND l.timestamp < ? AND u.url LIKE ?`,
		websiteID))

	rows, err := s.repo.GetDB().Query(query, startTime.Unix(), endTime.Unix(), "%"+urlFilter+"%")
	if err != nil {
		return results, err
	}
	defer rows.Close()

	uvSets := make([]map[int64]struct{}, len(results))
	for rows.Next() {
		var timestamp int64
		var ipID int64
		if err := rows.Scan(&timestamp, &ipID); err != nil {
			return results, err
		}

		pointTime := time.Unix(timestamp, 0)
		var key interface{}
		if viewType == "hourly" {
			key = hourBucket(pointTime)
		} else {
			key = dayBucket(pointTime)
		}
		idx, ok := bucketIndex[key]
		if !ok {
			continue
		}
		results[idx].PV++
		if uvSets[idx] == nil {
			uvSets[idx] = make(map[int64]struct{})
		}
		uvSets[idx][ipID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return results, err
	}
	for idx, ips := range uvSets {
		results[idx].UV = len(ips)
	}

	return results, nil
}

func (s *TimeSeriesStatsManager) statsByHourlyBuckets(
	websiteID string, timePoints []time.Time, results []StatPoint) ([]StatPoint, error) {

	bucketIndex := make(map[int64]int, len(timePoints))
	startBucket := hourBucket(timePoints[0])
	endBucket := hourBucket(timePoints[len(timePoints)-1])
	for i, point := range timePoints {
		bucket := hourBucket(point)
		results[i] = StatPoint{}
		bucketIndex[bucket] = i
	}

	rows, err := s.repo.GetDB().Query(sqlutil.ReplacePlaceholders(fmt.Sprintf(
		`SELECT bucket, pv FROM "%s_agg_hourly" WHERE bucket >= ? AND bucket <= ?`,
		websiteID,
	)), startBucket, endBucket)
	if err != nil {
		return results, err
	}
	defer rows.Close()
	for rows.Next() {
		var bucket int64
		var pv int
		if err := rows.Scan(&bucket, &pv); err != nil {
			return results, err
		}
		if idx, ok := bucketIndex[bucket]; ok {
			results[idx].PV = pv
		}
	}
	if err := rows.Err(); err != nil {
		return results, err
	}

	uvRows, err := s.repo.GetDB().Query(sqlutil.ReplacePlaceholders(fmt.Sprintf(
		`SELECT bucket, COUNT(*) FROM "%s_agg_hourly_ip" WHERE bucket >= ? AND bucket <= ? GROUP BY bucket`,
		websiteID,
	)), startBucket, endBucket)
	if err != nil {
		return results, err
	}
	defer uvRows.Close()
	for uvRows.Next() {
		var bucket int64
		var uv int
		if err := uvRows.Scan(&bucket, &uv); err != nil {
			return results, err
		}
		if idx, ok := bucketIndex[bucket]; ok {
			results[idx].UV = uv
		}
	}
	if err := uvRows.Err(); err != nil {
		return results, err
	}

	return results, nil
}

func (s *TimeSeriesStatsManager) statsByDailyBuckets(
	websiteID string, timePoints []time.Time, results []StatPoint) ([]StatPoint, error) {

	dayIndex := make(map[string]int, len(timePoints))
	startDay := dayBucket(timePoints[0])
	endDay := dayBucket(timePoints[len(timePoints)-1])
	for i, point := range timePoints {
		day := dayBucket(point)
		results[i] = StatPoint{}
		dayIndex[day] = i
	}

	rows, err := s.repo.GetDB().Query(sqlutil.ReplacePlaceholders(fmt.Sprintf(
		`SELECT day, pv FROM "%s_agg_daily" WHERE day >= ? AND day <= ?`,
		websiteID,
	)), startDay, endDay)
	if err != nil {
		return results, err
	}
	defer rows.Close()
	for rows.Next() {
		var day time.Time
		var pv int
		if err := rows.Scan(&day, &pv); err != nil {
			return results, err
		}
		dayKey := day.Format("2006-01-02")
		if idx, ok := dayIndex[dayKey]; ok {
			results[idx].PV = pv
		}
	}
	if err := rows.Err(); err != nil {
		return results, err
	}

	uvRows, err := s.repo.GetDB().Query(sqlutil.ReplacePlaceholders(fmt.Sprintf(
		`SELECT day, COUNT(*) FROM "%s_agg_daily_ip" WHERE day >= ? AND day <= ? GROUP BY day`,
		websiteID,
	)), startDay, endDay)
	if err != nil {
		return results, err
	}
	defer uvRows.Close()
	for uvRows.Next() {
		var day time.Time
		var uv int
		if err := uvRows.Scan(&day, &uv); err != nil {
			return results, err
		}
		dayKey := day.Format("2006-01-02")
		if idx, ok := dayIndex[dayKey]; ok {
			results[idx].UV = uv
		}
	}
	if err := uvRows.Err(); err != nil {
		return results, err
	}

	return results, nil
}

func hourBucket(ts time.Time) int64 {
	local := ts.In(time.Local)
	start := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), 0, 0, 0, local.Location())
	return start.Unix()
}

func dayBucket(ts time.Time) string {
	return ts.In(time.Local).Format("2006-01-02")
}
