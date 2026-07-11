package analytics

import (
	"fmt"
	"strings"

	"github.com/qianlima-666/nginxpulse/internal/sqlutil"
	"github.com/qianlima-666/nginxpulse/internal/store"
	"github.com/qianlima-666/nginxpulse/internal/timeutil"
)

type RefererIPGroupStats struct {
	Key      []string  `json:"key"`
	UV       []int     `json:"uv"`
	Share    []float64 `json:"share"`
	Domestic []string  `json:"domestic"`
	Global   []string  `json:"global"`
	TotalUV  int       `json:"total_uv"`
}

type RefererIPBatchStats struct {
	All      RefererIPGroupStats `json:"all"`
	Search   RefererIPGroupStats `json:"search"`
	Direct   RefererIPGroupStats `json:"direct"`
	External RefererIPGroupStats `json:"external"`
}

func (s RefererIPBatchStats) GetType() string {
	return "referer_ip_batch"
}

type RefererIPBatchStatsManager struct {
	repo *store.Repository
}

func NewRefererIPBatchStatsManager(repo *store.Repository) *RefererIPBatchStatsManager {
	return &RefererIPBatchStatsManager{repo: repo}
}

func (m *RefererIPBatchStatsManager) Query(query StatsQuery) (StatsResult, error) {
	result := RefererIPBatchStats{}
	timeRange := query.ExtraParam["timeRange"].(string)
	limit, _ := query.ExtraParam["limit"].(int)
	if limit <= 0 {
		limit = 10
	}
	urlFilter := ""
	if value, ok := query.ExtraParam["urlFilter"].(string); ok {
		urlFilter = strings.TrimSpace(value)
	}

	startTime, endTime, err := timeutil.TimePeriod(timeRange)
	if err != nil {
		return result, err
	}

	all, err := m.queryGroup(query.WebsiteID, startTime.Unix(), endTime.Unix(), limit, "all", urlFilter)
	if err != nil {
		return result, err
	}
	search, err := m.queryGroup(query.WebsiteID, startTime.Unix(), endTime.Unix(), limit, "search", urlFilter)
	if err != nil {
		return result, err
	}
	direct, err := m.queryGroup(query.WebsiteID, startTime.Unix(), endTime.Unix(), limit, "direct", urlFilter)
	if err != nil {
		return result, err
	}
	external, err := m.queryGroup(query.WebsiteID, startTime.Unix(), endTime.Unix(), limit, "external", urlFilter)
	if err != nil {
		return result, err
	}

	result.All = all
	result.Search = search
	result.Direct = direct
	result.External = external
	return result, nil
}

func (m *RefererIPBatchStatsManager) queryGroup(
	websiteID string,
	startUnix int64,
	endUnix int64,
	limit int,
	sourceKind string,
	urlFilter string,
) (RefererIPGroupStats, error) {
	result := RefererIPGroupStats{
		Key:      make([]string, 0),
		UV:       make([]int, 0),
		Share:    make([]float64, 0),
		Domestic: make([]string, 0),
		Global:   make([]string, 0),
	}

	sourceCondition := buildRefererSourceCondition(sourceKind, "r.referer")
	extraCondition := ""
	if sourceCondition != "" {
		extraCondition = " AND " + sourceCondition
	}
	urlJoin := ""
	args := []interface{}{startUnix, endUnix}
	if urlFilter != "" {
		urlJoin = fmt.Sprintf(`JOIN "%s_dim_url" filter_url ON filter_url.id = l.url_id`, websiteID)
		extraCondition += " AND filter_url.url LIKE ?"
		args = append(args, "%"+urlFilter+"%")
	}

	totalQuery := sqlutil.ReplacePlaceholders(fmt.Sprintf(`
        SELECT COUNT(*)
        FROM "%[1]s_nginx_logs" l
        JOIN "%[1]s_dim_referer" r ON r.id = l.referer_id
        %[2]s
        WHERE l.pageview_flag = 1 AND l.timestamp >= ? AND l.timestamp < ?%[3]s`,
		websiteID, urlJoin, extraCondition))

	if err := m.repo.GetDB().QueryRow(totalQuery, args...).Scan(&result.TotalUV); err != nil {
		return result, fmt.Errorf("查询来源IP总量失败: %v", err)
	}

	querySQL := sqlutil.ReplacePlaceholders(fmt.Sprintf(`
        WITH filtered AS (
            SELECT l.ip_id, ip.ip, l.location_id
            FROM "%[1]s_nginx_logs" l
            JOIN "%[1]s_dim_ip" ip ON ip.id = l.ip_id
            JOIN "%[1]s_dim_referer" r ON r.id = l.referer_id
            %[2]s
            WHERE l.pageview_flag = 1 AND l.timestamp >= ? AND l.timestamp < ?%[3]s
        ),
        ip_counts AS (
            SELECT ip_id, ip, COUNT(*) AS uv
            FROM filtered
            GROUP BY ip_id, ip
        ),
        top_ips AS (
            SELECT ip_id, ip, uv
            FROM ip_counts
            ORDER BY uv DESC, ip ASC
            LIMIT ?
        ),
        location_rank AS (
            SELECT
                f.ip_id,
                COALESCE(loc.domestic, '-') AS domestic,
                COALESCE(loc.global, '-') AS global,
                COUNT(*) AS location_count,
                ROW_NUMBER() OVER (
                    PARTITION BY f.ip_id
                    ORDER BY COUNT(*) DESC, COALESCE(loc.global, '-') ASC, COALESCE(loc.domestic, '-') ASC
                ) AS rn
            FROM filtered f
            LEFT JOIN "%[1]s_dim_location" loc ON loc.id = f.location_id
            JOIN top_ips t ON t.ip_id = f.ip_id
            GROUP BY f.ip_id, COALESCE(loc.domestic, '-'), COALESCE(loc.global, '-')
        )
        SELECT
            t.ip,
            t.uv,
            COALESCE(lr.domestic, '-') AS domestic,
            COALESCE(lr.global, '-') AS global
        FROM top_ips t
        LEFT JOIN location_rank lr ON lr.ip_id = t.ip_id AND lr.rn = 1
        ORDER BY t.uv DESC, t.ip ASC`,
		websiteID, urlJoin, extraCondition))

	queryArgs := append([]interface{}{}, args...)
	queryArgs = append(queryArgs, limit)
	rows, err := m.repo.GetDB().Query(querySQL, queryArgs...)
	if err != nil {
		return result, fmt.Errorf("查询来源IP排行失败: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			ip       string
			uv       int
			domestic string
			global   string
		)
		if err := rows.Scan(&ip, &uv, &domestic, &global); err != nil {
			return result, fmt.Errorf("解析来源IP排行失败: %v", err)
		}
		result.Key = append(result.Key, ip)
		result.UV = append(result.UV, uv)
		result.Domestic = append(result.Domestic, domestic)
		result.Global = append(result.Global, global)
		if result.TotalUV <= 0 {
			result.Share = append(result.Share, 0)
		} else {
			result.Share = append(result.Share, float64(uv)/float64(result.TotalUV))
		}
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("遍历来源IP排行失败: %v", err)
	}

	return result, nil
}
