package ingest

import (
	"fmt"
	"strings"

	"github.com/qianlima-666/nginxpulse/internal/config"
	"github.com/qianlima-666/nginxpulse/internal/store"
	"github.com/sirupsen/logrus"
)

func (p *LogParser) notifySystem(
	level, category, title, message, fingerprint string,
	metadata map[string]interface{},
) {
	if p == nil || p.repo == nil {
		return
	}
	entry := store.SystemNotification{
		Level:       level,
		Category:    category,
		Title:       title,
		Message:     message,
		Fingerprint: fingerprint,
		Metadata:    metadata,
	}
	if _, err := p.repo.CreateSystemNotification(entry); err != nil {
		logrus.WithError(err).Warn("写入系统通知失败")
		return
	}
	if p.alertDispatcher != nil {
		go p.alertDispatcher.Send(entry)
	}
}

func (p *LogParser) notifyFileIO(websiteID, filePath, action string, err error) {
	if err == nil {
		return
	}
	path := strings.TrimSpace(filePath)
	normalized := path
	if normalized != "" {
		normalized = normalizeLogPath(normalized)
	}
	title := "文件/IO 异常"
	message := fmt.Sprintf("%s失败：%s", action, err.Error())
	fingerprint := fmt.Sprintf("file_io:%s:%s:%s", websiteID, action, normalized)
	metadata := map[string]interface{}{
		"website_id": websiteID,
		"file_path":  path,
		"action":     action,
		"error":      err.Error(),
	}
	if websiteID != "" {
		if site, ok := config.GetWebsiteByID(websiteID); ok {
			metadata["website_name"] = site.Name
		}
	}
	p.notifySystem("warning", "file_io", title, message, fingerprint, metadata)
}

func (p *LogParser) notifyLogParsing(websiteID, filePath, action string, err error) {
	if err == nil {
		return
	}
	path := strings.TrimSpace(filePath)
	normalized := path
	if normalized != "" {
		normalized = normalizeLogPath(normalized)
	}
	title := "日志解析异常"
	message := fmt.Sprintf("%s失败：%s", action, err.Error())
	fingerprint := fmt.Sprintf("log_parsing:%s:%s:%s", websiteID, action, normalized)
	metadata := map[string]interface{}{
		"website_id": websiteID,
		"file_path":  path,
		"action":     action,
		"error":      err.Error(),
	}
	if websiteID != "" {
		if site, ok := config.GetWebsiteByID(websiteID); ok {
			metadata["website_name"] = site.Name
		}
	}
	p.notifySystem("warning", "log_parsing", title, message, fingerprint, metadata)
}

func (p *LogParser) notifyDatabaseWrite(websiteID, action string, err error) {
	if err == nil {
		return
	}
	title := "数据库写入失败"
	message := fmt.Sprintf("%s失败：%s", action, err.Error())
	fingerprint := fmt.Sprintf("db_write:%s:%s", websiteID, action)
	metadata := map[string]interface{}{
		"website_id": websiteID,
		"action":     action,
		"error":      err.Error(),
	}
	if websiteID != "" {
		if site, ok := config.GetWebsiteByID(websiteID); ok {
			metadata["website_name"] = site.Name
		}
	}
	p.notifySystem("warning", "db_write", title, message, fingerprint, metadata)
}
