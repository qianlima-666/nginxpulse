package config

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidateOptions struct {
	CheckPaths  bool
	CheckRemote bool
}

type ValidationResult struct {
	Errors   []FieldError `json:"errors"`
	Warnings []FieldError `json:"warnings"`
}

func ValidateConfig(cfg *Config, opts ValidateOptions) ValidationResult {
	result := ValidationResult{}
	addError := func(field, msg string) {
		result.Errors = append(result.Errors, FieldError{Field: field, Message: msg})
	}
	addWarning := func(field, msg string) {
		result.Warnings = append(result.Warnings, FieldError{Field: field, Message: msg})
	}

	if cfg == nil {
		addError("config", "配置不能为空")
		return result
	}

	if len(cfg.Websites) == 0 {
		addError("websites", "至少需要配置一个站点")
	}

	for i, site := range cfg.Websites {
		sitePrefix := fmt.Sprintf("websites[%d]", i)
		if strings.TrimSpace(site.Name) == "" {
			addError(sitePrefix+".name", "站点名称不能为空")
		}

		if len(site.Sources) == 0 {
			if strings.TrimSpace(site.LogPath) == "" {
				addError(sitePrefix+".logPath", "日志路径不能为空")
			} else if opts.CheckPaths {
				if err := validatePath(site.LogPath); err != nil {
					addError(sitePrefix+".logPath", err.Error())
				}
			}
			continue
		}

		seen := map[string]struct{}{}
		for sidx, src := range site.Sources {
			srcPrefix := fmt.Sprintf("%s.sources[%d]", sitePrefix, sidx)
			id := strings.TrimSpace(src.ID)
			if id == "" {
				addError(srcPrefix+".id", "source.id 不能为空")
			} else if _, ok := seen[id]; ok {
				addError(srcPrefix+".id", "source.id 重复")
			} else {
				seen[id] = struct{}{}
			}

			stype := strings.ToLower(strings.TrimSpace(src.Type))
			if stype == "" {
				addError(srcPrefix+".type", "source.type 不能为空")
				continue
			}

			switch stype {
			case "local":
				if strings.TrimSpace(src.Path) == "" && strings.TrimSpace(src.Pattern) == "" {
					addError(srcPrefix, "local 需要 path 或 pattern")
				} else if opts.CheckPaths {
					if src.Path != "" {
						if err := validatePath(src.Path); err != nil {
							addError(srcPrefix+".path", err.Error())
						}
					}
					if src.Pattern != "" {
						if err := validatePath(src.Pattern); err != nil {
							addError(srcPrefix+".pattern", err.Error())
						}
					}
				}
			case "sftp":
				if strings.TrimSpace(src.Host) == "" {
					addError(srcPrefix+".host", "sftp.host 不能为空")
				}
				if strings.TrimSpace(src.User) == "" {
					addError(srcPrefix+".user", "sftp.user 不能为空")
				}
				if src.Auth == nil || (strings.TrimSpace(src.Auth.KeyFile) == "" && strings.TrimSpace(src.Auth.Password) == "") {
					addError(srcPrefix+".auth", "sftp 需要 keyFile 或 password")
				}
				if strings.TrimSpace(src.Path) == "" && strings.TrimSpace(src.Pattern) == "" {
					addError(srcPrefix, "sftp 需要 path 或 pattern")
				} else if opts.CheckRemote {
					addWarning(srcPrefix, "远端路径校验会在后续版本支持")
				}
			case "http":
				if strings.TrimSpace(src.URL) == "" {
					addError(srcPrefix+".url", "http.url 不能为空")
				}
				if src.Index != nil && strings.TrimSpace(src.Index.URL) == "" {
					addError(srcPrefix+".index.url", "http.index.url 不能为空")
				}
			case "s3":
				if strings.TrimSpace(src.Bucket) == "" {
					addError(srcPrefix+".bucket", "s3.bucket 不能为空")
				}
				if (strings.TrimSpace(src.AccessKey) == "") != (strings.TrimSpace(src.SecretKey) == "") {
					addError(srcPrefix+".accessKey", "s3 accessKey/secretKey 需同时配置")
				}
			case "agent":
				// no-op
			default:
				addError(srcPrefix+".type", "不支持的 source.type")
			}
		}

		if site.Whitelist != nil {
			whitelistPrefix := sitePrefix + ".whitelist"
			if site.Whitelist.Enabled && len(site.Whitelist.IPs) == 0 && len(site.Whitelist.Cities) == 0 && !site.Whitelist.NonMainland {
				addError(whitelistPrefix, "白名单已启用，但未配置任何规则")
			}
			if len(site.Whitelist.IPs) > 0 {
				for _, raw := range site.Whitelist.IPs {
					if err := validateWhitelistIP(raw); err != nil {
						addError(whitelistPrefix+".ips", fmt.Sprintf("白名单 IP/IP 段格式不正确: %s", raw))
						break
					}
				}
			}
		}
	}

	if strings.TrimSpace(cfg.Database.Driver) == "" {
		addError("database.driver", "数据库驱动不能为空")
	} else if strings.TrimSpace(cfg.Database.Driver) != "postgres" {
		addError("database.driver", "仅支持 postgres 驱动")
	}
	if strings.TrimSpace(cfg.Database.DSN) == "" {
		addError("database.dsn", "数据库 DSN 不能为空")
	}
	if cfg.System.LogRetentionDays <= 0 {
		addError("system.logRetentionDays", "logRetentionDays 必须大于 0")
	}
	if cfg.System.ParseBatchSize <= 0 {
		addError("system.parseBatchSize", "parseBatchSize 必须大于 0")
	}
	if strings.TrimSpace(cfg.System.BackfillMaxDurationPerRun) != "" {
		duration, err := time.ParseDuration(strings.TrimSpace(cfg.System.BackfillMaxDurationPerRun))
		if err != nil {
			addError("system.backfillMaxDurationPerRun", "backfillMaxDurationPerRun 格式无效，示例：8s、30s")
		} else if duration <= 0 {
			addError("system.backfillMaxDurationPerRun", "backfillMaxDurationPerRun 必须大于 0")
		}
	}
	if cfg.System.BackfillMaxBytesPerRun <= 0 {
		addError("system.backfillMaxBytesPerRun", "backfillMaxBytesPerRun 必须大于 0")
	}
	if cfg.System.IPGeoCacheLimit <= 0 {
		addError("system.ipGeoCacheLimit", "ipGeoCacheLimit 必须大于 0")
	}
	if cfg.System.AlertPush != nil {
		alert := cfg.System.AlertPush
		if strings.TrimSpace(alert.Timeout) != "" {
			timeout, err := time.ParseDuration(strings.TrimSpace(alert.Timeout))
			if err != nil {
				addError("system.alertPush.timeout", "alertPush.timeout 格式无效，示例：3s、10s")
			} else if timeout <= 0 {
				addError("system.alertPush.timeout", "alertPush.timeout 必须大于 0")
			}
		}

		validateWebhook := func(field, label, webhook string, enabled bool) {
			if !enabled {
				return
			}
			if strings.TrimSpace(webhook) == "" {
				addError(field, label+" webhook 不能为空")
				return
			}
			if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(webhook)), "http://") &&
				!strings.HasPrefix(strings.ToLower(strings.TrimSpace(webhook)), "https://") {
				addError(field, label+" webhook 必须是 http(s) 地址")
			}
		}

		validateWebhook("system.alertPush.feishu.webhook", "飞书", alert.Feishu.Webhook, alert.Feishu.Enabled)
		validateWebhook("system.alertPush.dingtalk.webhook", "钉钉", alert.DingTalk.Webhook, alert.DingTalk.Enabled)
		validateWebhook("system.alertPush.wecom.webhook", "企微", alert.WeCom.Webhook, alert.WeCom.Enabled)

		if alert.Email.Enabled {
			if strings.TrimSpace(alert.Email.Host) == "" {
				addError("system.alertPush.email.host", "邮件 SMTP host 不能为空")
			}
			if alert.Email.Port <= 0 {
				addError("system.alertPush.email.port", "邮件 SMTP port 必须大于 0")
			}
			if strings.TrimSpace(alert.Email.From) == "" {
				addError("system.alertPush.email.from", "邮件 from 不能为空")
			}
			recipients := 0
			for _, value := range alert.Email.To {
				if strings.TrimSpace(value) != "" {
					recipients++
				}
			}
			if recipients == 0 {
				addError("system.alertPush.email.to", "邮件收件人不能为空")
			}
		}
	}
	if cfg.System.AccessKeyExpireDays <= 0 {
		addError("system.accessKeyExpireDays", "accessKeyExpireDays 必须大于 0")
	}
	if strings.TrimSpace(cfg.System.HTTPSourceTimeout) != "" {
		timeout, err := time.ParseDuration(strings.TrimSpace(cfg.System.HTTPSourceTimeout))
		if err != nil {
			addError("system.httpSourceTimeout", "httpSourceTimeout 格式无效，示例：30s、2m")
		} else if timeout <= 0 {
			addError("system.httpSourceTimeout", "httpSourceTimeout 必须大于 0")
		}
	}
	if cfg.System.ServerStatus.Enabled {
		validateHTTPURL := func(field, label, value string) {
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				addError(field, label+"不能为空")
				return
			}
			parsed, err := url.Parse(trimmed)
			if err != nil || parsed.Host == "" {
				addError(field, label+"格式无效")
				return
			}
			if parsed.Scheme != "http" && parsed.Scheme != "https" {
				addError(field, label+"必须是 http(s) 地址")
			}
		}
		if !cfg.System.ServerStatus.MockEnabled {
			validateHTTPURL("system.serverStatus.metricsUrl", "服务器状态接口", cfg.System.ServerStatus.MetricsURL)
			validateHTTPURL("system.serverStatus.disksUrl", "磁盘状态接口", cfg.System.ServerStatus.DisksURL)
		}
		if timeout, err := time.ParseDuration(strings.TrimSpace(cfg.System.ServerStatus.Timeout)); err != nil {
			addError("system.serverStatus.timeout", "serverStatus.timeout 格式无效，示例：5s、10s")
		} else if timeout <= 0 {
			addError("system.serverStatus.timeout", "serverStatus.timeout 必须大于 0")
		}
		if interval, err := time.ParseDuration(strings.TrimSpace(cfg.System.ServerStatus.RefreshInterval)); err != nil {
			addError("system.serverStatus.refreshInterval", "serverStatus.refreshInterval 格式无效，示例：30s、1m")
		} else if interval <= 0 {
			addError("system.serverStatus.refreshInterval", "serverStatus.refreshInterval 必须大于 0")
		}
	}
	if basePath := NormalizeWebBasePath(cfg.System.WebBasePath); basePath != "" {
		if strings.Contains(basePath, "/") {
			addError("system.webBasePath", "webBasePath 仅支持单段路径")
		} else if !regexp.MustCompile("^[a-zA-Z0-9_-]+$").MatchString(basePath) {
			addError("system.webBasePath", "webBasePath 仅支持字母、数字、下划线或短横线")
		} else {
			reserved := map[string]struct{}{
				"api":            {},
				"m":              {},
				"assets":         {},
				"favicon.svg":    {},
				"brand-mark":     {},
				"brand-mark.svg": {},
				"app-config.js":  {},
				"health":         {},
			}
			if _, ok := reserved[strings.ToLower(basePath)]; ok {
				addError("system.webBasePath", "webBasePath 与系统保留路径冲突")
			}
		}
	}

	if len(cfg.PVFilter.StatusCodeInclude) == 0 {
		addError("pvFilter.statusCodeInclude", "statusCodeInclude 不能为空")
	}
	if len(cfg.PVFilter.ExcludePatterns) == 0 {
		addError("pvFilter.excludePatterns", "excludePatterns 不能为空")
	}

	return result
}

func validateWhitelistIP(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if strings.Contains(trimmed, "/") {
		if _, _, err := net.ParseCIDR(trimmed); err != nil {
			return err
		}
		return nil
	}
	if strings.Contains(trimmed, "-") {
		parts := strings.SplitN(trimmed, "-", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid range")
		}
		start := net.ParseIP(strings.TrimSpace(parts[0]))
		end := net.ParseIP(strings.TrimSpace(parts[1]))
		if start == nil || end == nil {
			return fmt.Errorf("invalid range")
		}
		if (start.To4() == nil) != (end.To4() == nil) {
			return fmt.Errorf("mixed ip versions")
		}
		if compareIP(start, end) > 0 {
			return fmt.Errorf("range start > end")
		}
		return nil
	}
	if net.ParseIP(trimmed) == nil {
		return fmt.Errorf("invalid ip")
	}
	return nil
}

func compareIP(a, b net.IP) int {
	aa := a.To16()
	bb := b.To16()
	if aa == nil || bb == nil {
		return 0
	}
	return bytes.Compare(aa, bb)
}

func validatePath(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("路径不能为空")
	}
	if strings.Contains(value, "*") {
		matches, err := filepath.Glob(value)
		if err != nil || len(matches) == 0 {
			return fmt.Errorf("日志路径未匹配到任何文件")
		}
		return nil
	}
	if _, err := os.Stat(value); err != nil {
		return fmt.Errorf("日志路径不存在或不可访问")
	}
	return nil
}
