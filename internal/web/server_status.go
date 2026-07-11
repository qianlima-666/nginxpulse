package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qianlima-666/nginxpulse/internal/config"
)

const serverStatusMaxBodyBytes = 2 * 1024 * 1024

func registerServerStatusRoute(router *gin.Engine) {
	router.GET("/api/server-status", func(c *gin.Context) {
		cfg := config.ReadConfig()
		statusCfg := cfg.System.ServerStatus
		timeout := parseServerStatusDuration(statusCfg.Timeout, 5*time.Second)
		refreshInterval := parseServerStatusDuration(statusCfg.RefreshInterval, 30*time.Second)

		if !statusCfg.Enabled {
			c.JSON(http.StatusOK, gin.H{
				"enabled":                  false,
				"status":                   "disabled",
				"refresh_interval_seconds": durationSeconds(refreshInterval),
			})
			return
		}

		if statusCfg.MockEnabled {
			c.JSON(http.StatusOK, mockServerStatusResponse(refreshInterval))
			return
		}

		client := &http.Client{Timeout: timeout}
		metrics, metricsErr := fetchServerStatusJSON(c.Request.Context(), client, statusCfg.MetricsURL)
		disks, disksErr := fetchServerStatusJSON(c.Request.Context(), client, statusCfg.DisksURL)

		errors := make([]string, 0, 2)
		if metricsErr != nil {
			errors = append(errors, fmt.Sprintf("metrics: %v", metricsErr))
		}
		if disksErr != nil {
			errors = append(errors, fmt.Sprintf("disks: %v", disksErr))
		}

		status := "ok"
		if len(errors) == 1 {
			status = "partial"
		} else if len(errors) > 1 {
			status = "error"
		} else if upstreamStatus(metrics) != "ok" || upstreamStatus(disks) != "ok" {
			status = "warning"
		}

		c.JSON(http.StatusOK, gin.H{
			"enabled":                  true,
			"status":                   status,
			"updated_at":               firstString(metrics["updated_at"], disks["updated_at"]),
			"metrics":                  mapValue(metrics, "metrics"),
			"missing_metrics":          stringSliceValue(metrics, "missing_metrics"),
			"disk_count":               diskCount(disks),
			"disks":                    diskList(disks),
			"errors":                   errors,
			"refresh_interval_seconds": durationSeconds(refreshInterval),
		})
	})
}

func mockServerStatusResponse(refreshInterval time.Duration) gin.H {
	now := time.Now().Format(time.RFC3339)
	return gin.H{
		"enabled":                  true,
		"status":                   "warning",
		"updated_at":               now,
		"refresh_interval_seconds": durationSeconds(refreshInterval),
		"metrics": gin.H{
			"cpu_temp_celsius":   52.4,
			"board_temp_celsius": 34.8,
			"nvme_temp_celsius":  62.7,
			"cpu_fan_rpm":        940,
			"chassis_fan1_rpm":   720,
		},
		"missing_metrics": []string{},
		"disk_count":      5,
		"disks": []gin.H{
			{
				"name":                     "nvme1n1",
				"path":                     "/dev/nvme1n1",
				"smartctl_path":            "/dev/nvme1",
				"type":                     "nvme",
				"model":                    "Samsung 980 PRO 2TB",
				"serial":                   "S6AXNS0T900123A",
				"firmware_version":         "5B2QGXA7",
				"smartctl_exit_status":     0,
				"size_bytes":               2000398934016,
				"smart_available":          true,
				"smart_enabled":            true,
				"health_passed":            true,
				"temperature_celsius":      62.7,
				"percentage_used":          18,
				"percentage_remaining":     82,
				"media_errors":             0,
				"error_log_entries":        3,
				"unsafe_shutdowns":         8,
				"power_on_hours":           4680,
				"power_cycles":             72,
				"data_units_read_bytes":    12884901888000,
				"data_units_written_bytes": 7421703487488,
			},
			{
				"name":                     "sdb",
				"path":                     "/dev/sdb",
				"smartctl_path":            "/dev/sdb",
				"type":                     "sat",
				"model":                    "WD Blue SA510 1TB",
				"serial":                   "WD-WXK2A2390001",
				"firmware_version":         "52040100",
				"smartctl_exit_status":     0,
				"size_bytes":               1000204886016,
				"smart_available":          true,
				"smart_enabled":            true,
				"health_passed":            true,
				"temperature_celsius":      41,
				"percentage_used":          64,
				"percentage_remaining":     36,
				"media_errors":             2,
				"error_log_entries":        9,
				"unsafe_shutdowns":         1,
				"power_on_hours":           12840,
				"power_cycles":             122,
				"data_units_read_bytes":    9455799992320,
				"data_units_written_bytes": 6120328396800,
			},
			{
				"name":                     "sdc",
				"path":                     "/dev/sdc",
				"smartctl_path":            "/dev/sdc",
				"type":                     "sat",
				"model":                    "Seagate IronWolf 4TB",
				"serial":                   "ZFN19ABC",
				"firmware_version":         "SC60",
				"smartctl_exit_status":     0,
				"size_bytes":               4000787030016,
				"smart_available":          true,
				"smart_enabled":            true,
				"health_passed":            false,
				"temperature_celsius":      48,
				"percentage_used":          92,
				"percentage_remaining":     8,
				"media_errors":             14,
				"error_log_entries":        31,
				"unsafe_shutdowns":         3,
				"power_on_hours":           30220,
				"power_cycles":             86,
				"data_units_read_bytes":    38654705664000,
				"data_units_written_bytes": 24696061952000,
			},
			{
				"name":                     "nvme0n1",
				"path":                     "/dev/nvme0n1",
				"smartctl_path":            "/dev/nvme0",
				"type":                     "nvme",
				"model":                    "Fanxiang S500Pro 1TB",
				"serial":                   "26030261611000611",
				"firmware_version":         "GT18c634",
				"smartctl_exit_status":     0,
				"size_bytes":               1024209543168,
				"smart_available":          true,
				"smart_enabled":            true,
				"health_passed":            true,
				"temperature_celsius":      39.9,
				"percentage_used":          0,
				"percentage_remaining":     100,
				"media_errors":             0,
				"error_log_entries":        0,
				"unsafe_shutdowns":         11,
				"power_on_hours":           27,
				"power_cycles":             34,
				"data_units_read_bytes":    2825585664000,
				"data_units_written_bytes": 849481216000,
			},
			{
				"name":                     "sdd",
				"path":                     "/dev/sdd",
				"smartctl_path":            "/dev/sdd",
				"type":                     "sat",
				"model":                    "Crucial MX500 500GB",
				"serial":                   "2306E6AA77C0",
				"firmware_version":         "M3CR046",
				"smartctl_exit_status":     0,
				"size_bytes":               500107862016,
				"smart_available":          true,
				"smart_enabled":            true,
				"health_passed":            true,
				"temperature_celsius":      36,
				"percentage_used":          22,
				"percentage_remaining":     78,
				"media_errors":             0,
				"error_log_entries":        1,
				"unsafe_shutdowns":         0,
				"power_on_hours":           6210,
				"power_cycles":             210,
				"data_units_read_bytes":    1932735283200,
				"data_units_written_bytes": 1181116006400,
			},
		},
		"errors": []string{},
	}
}

func fetchServerStatusJSON(ctx context.Context, client *http.Client, rawURL string) (map[string]any, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return map[string]any{}, fmt.Errorf("接口地址未配置")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return map[string]any{}, fmt.Errorf("接口地址格式无效")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return map[string]any{}, fmt.Errorf("仅支持 http(s) 接口")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trimmed, nil)
	if err != nil {
		return map[string]any{}, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return map[string]any{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return map[string]any{}, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body := io.LimitReader(resp.Body, serverStatusMaxBodyBytes)
	decoder := json.NewDecoder(body)
	decoder.UseNumber()
	data := map[string]any{}
	if err := decoder.Decode(&data); err != nil {
		return map[string]any{}, err
	}
	return data, nil
}

func parseServerStatusDuration(value string, fallback time.Duration) time.Duration {
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func durationSeconds(value time.Duration) int {
	seconds := int(value.Seconds())
	if seconds < 1 {
		return 1
	}
	return seconds
}

func upstreamStatus(data map[string]any) string {
	value, _ := data["status"].(string)
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "ok"
	}
	return value
}

func firstString(values ...any) string {
	for _, value := range values {
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			return text
		}
	}
	return ""
}

func mapValue(data map[string]any, key string) map[string]any {
	value, ok := data[key].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return value
}

func stringSliceValue(data map[string]any, key string) []string {
	raw, ok := data[key].([]any)
	if !ok {
		return nil
	}
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok {
			values = append(values, text)
		}
	}
	return values
}

func diskCount(data map[string]any) int {
	if value, ok := data["disk_count"]; ok {
		if count, ok := numberToInt(value); ok {
			return count
		}
	}
	return len(diskList(data))
}

func diskList(data map[string]any) []map[string]any {
	raw, ok := data["disks"].([]any)
	if !ok {
		return []map[string]any{}
	}
	disks := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if disk, ok := item.(map[string]any); ok {
			disks = append(disks, disk)
		}
	}
	return disks
}

func numberToInt(value any) (int, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return int(parsed), true
		}
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case int64:
		return int(typed), true
	}
	return 0, false
}
