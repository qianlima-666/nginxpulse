package alertpush

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/qianlima-666/nginxpulse/internal/config"
	"github.com/qianlima-666/nginxpulse/internal/store"
	"github.com/sirupsen/logrus"
)

const (
	defaultTimeout = 5 * time.Second
	maxRespBody    = 4096
)

type Dispatcher struct {
	cfg    config.AlertPushConfig
	client *http.Client
}

type ChannelResult struct {
	Enabled bool   `json:"enabled"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func NewDispatcher(cfg *config.AlertPushConfig) *Dispatcher {
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	timeout := defaultTimeout
	if raw := strings.TrimSpace(cfg.Timeout); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			timeout = parsed
		}
	}

	cloned := *cfg
	return &Dispatcher{
		cfg:    cloned,
		client: &http.Client{Timeout: timeout},
	}
}

func (d *Dispatcher) Send(entry store.SystemNotification) {
	if d == nil {
		return
	}
	results := d.SendWithResult(entry, nil)
	for channel, result := range results {
		if result.Success {
			continue
		}
		logrus.WithField("channel", channel).Warnf("告警推送失败: %s", result.Error)
	}
}

func (d *Dispatcher) SendWithResult(entry store.SystemNotification, channels []string) map[string]ChannelResult {
	results := make(map[string]ChannelResult)
	if d == nil {
		return results
	}

	selected, hasSelected := normalizeChannels(channels)
	text := formatMessage(entry)

	trySend := func(channel string, enabled bool, sendFn func() error) {
		if hasSelected {
			if _, ok := selected[channel]; !ok {
				return
			}
		} else if !enabled {
			return
		}

		result := ChannelResult{Enabled: enabled}
		if !enabled {
			result.Error = "channel disabled"
			results[channel] = result
			return
		}
		if err := sendFn(); err != nil {
			result.Error = err.Error()
			results[channel] = result
			return
		}
		result.Success = true
		results[channel] = result
	}

	trySend(
		"feishu",
		d.cfg.Feishu.Enabled && strings.TrimSpace(d.cfg.Feishu.Webhook) != "",
		func() error { return d.sendFeishu(text) },
	)
	trySend(
		"dingtalk",
		d.cfg.DingTalk.Enabled && strings.TrimSpace(d.cfg.DingTalk.Webhook) != "",
		func() error { return d.sendDingTalk(text) },
	)
	trySend(
		"wecom",
		d.cfg.WeCom.Enabled && strings.TrimSpace(d.cfg.WeCom.Webhook) != "",
		func() error { return d.sendWeCom(text) },
	)
	trySend(
		"email",
		d.cfg.Email.Enabled,
		func() error { return d.sendEmail(entry, text) },
	)

	return results
}

func (d *Dispatcher) sendFeishu(text string) error {
	payload := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": text,
		},
	}
	return d.sendWebhook("feishu", strings.TrimSpace(d.cfg.Feishu.Webhook), payload)
}

func (d *Dispatcher) sendDingTalk(text string) error {
	webhook, err := withDingTalkSign(strings.TrimSpace(d.cfg.DingTalk.Webhook), strings.TrimSpace(d.cfg.DingTalk.Secret))
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": text,
		},
	}
	return d.sendWebhook("dingtalk", webhook, payload)
}

func (d *Dispatcher) sendWeCom(text string) error {
	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": text,
		},
	}
	return d.sendWebhook("wecom", strings.TrimSpace(d.cfg.WeCom.Webhook), payload)
}

func (d *Dispatcher) sendWebhook(provider, webhook string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, webhook, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxRespBody))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s webhook 响应状态 %d: %s", provider, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if len(respBody) == 0 {
		return nil
	}

	var ack map[string]interface{}
	if err := json.Unmarshal(respBody, &ack); err != nil {
		return nil
	}
	if v, ok := ack["code"]; ok && toInt(v) != 0 {
		return fmt.Errorf("%s webhook 返回错误 code=%d: %s", provider, toInt(v), strings.TrimSpace(string(respBody)))
	}
	if v, ok := ack["errcode"]; ok && toInt(v) != 0 {
		return fmt.Errorf("%s webhook 返回错误 errcode=%d: %s", provider, toInt(v), strings.TrimSpace(string(respBody)))
	}
	return nil
}

func (d *Dispatcher) sendEmail(entry store.SystemNotification, text string) error {
	emailCfg := d.cfg.Email
	host := strings.TrimSpace(emailCfg.Host)
	from := strings.TrimSpace(emailCfg.From)
	recipients := normalizeRecipients(emailCfg.To)

	if host == "" || emailCfg.Port <= 0 || from == "" || len(recipients) == 0 {
		return fmt.Errorf("邮件配置不完整")
	}

	addr := fmt.Sprintf("%s:%d", host, emailCfg.Port)
	subject := fmt.Sprintf("[NginxPulse][%s][%s] %s",
		strings.ToUpper(strings.TrimSpace(entry.Level)),
		strings.TrimSpace(entry.Category),
		strings.TrimSpace(entry.Title),
	)
	msg := buildEmailMessage(from, recipients, subject, text)

	var auth smtp.Auth
	username := strings.TrimSpace(emailCfg.Username)
	if username != "" {
		auth = smtp.PlainAuth("", username, strings.TrimSpace(emailCfg.Password), host)
	}

	if emailCfg.UseTLS {
		return sendMailTLS(addr, host, auth, from, recipients, msg)
	}
	return smtp.SendMail(addr, auth, from, recipients, msg)
}

func sendMailTLS(addr, host string, auth smtp.Auth, from string, to []string, message []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return err
			}
		}
	}

	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	return client.Quit()
}

func buildEmailMessage(from string, to []string, subject, body string) []byte {
	headers := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", strings.Join(to, ", ")),
		fmt.Sprintf("Subject: %s", subject),
		fmt.Sprintf("Date: %s", time.Now().Format(time.RFC1123Z)),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}
	return []byte(strings.Join(headers, "\r\n"))
}

func formatMessage(entry store.SystemNotification) string {
	occurrences := entry.Occurrences
	if occurrences <= 0 {
		occurrences = 1
	}

	metadata := "-"
	if len(entry.Metadata) > 0 {
		if encoded, err := json.Marshal(entry.Metadata); err == nil {
			metadata = string(encoded)
		}
	}

	title := strings.TrimSpace(entry.Title)
	if title == "" {
		title = "系统通知"
	}
	message := strings.TrimSpace(entry.Message)
	if message == "" {
		message = "-"
	}
	level := strings.TrimSpace(entry.Level)
	if level == "" {
		level = "info"
	}

	return fmt.Sprintf(
		"NginxPulse 告警通知\n标题: %s\n级别: %s\n分类: %s\n消息: %s\n重复次数: %d\n时间: %s\n指纹: %s\n元数据: %s",
		title,
		level,
		strings.TrimSpace(entry.Category),
		message,
		occurrences,
		time.Now().Format("2006-01-02 15:04:05"),
		strings.TrimSpace(entry.Fingerprint),
		metadata,
	)
}

func normalizeRecipients(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func withDingTalkSign(rawWebhook, secret string) (string, error) {
	if secret == "" {
		return rawWebhook, nil
	}

	timestamp := time.Now().UnixMilli()
	toSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(toSign)); err != nil {
		return "", err
	}
	sign := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))

	sep := "?"
	if strings.Contains(rawWebhook, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%stimestamp=%d&sign=%s", rawWebhook, sep, timestamp, sign), nil
}

func normalizeChannels(channels []string) (map[string]struct{}, bool) {
	result := make(map[string]struct{})
	for _, channel := range channels {
		normalized := strings.ToLower(strings.TrimSpace(channel))
		switch normalized {
		case "feishu", "dingtalk", "wecom", "email":
			result[normalized] = struct{}{}
		}
	}
	return result, len(result) > 0
}

func toInt(value interface{}) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float32:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}
