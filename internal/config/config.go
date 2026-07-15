package config

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	globalConfig *Config
	websiteIDMap sync.Map
)

const (
	DataDir                          = "./var/nginxpulse_data"
	ConfigFile                       = "./configs/nginxpulse_config.json"
	DefaultIPGeoAPIURL               = "http://ip-api.com/batch"
	DefaultBackfillMaxDurationPerRun = 8 * time.Second
	DefaultBackfillMaxBytesPerRun    = int64(32 * 1024 * 1024)
)

type Config struct {
	System   SystemConfig    `json:"system"`
	Server   ServerConfig    `json:"server"`
	Database DatabaseConfig  `json:"database"`
	Websites []WebsiteConfig `json:"websites"`
	PVFilter PVFilterConfig  `json:"pvFilter"`
}

type WebsiteConfig struct {
	Name              string           `json:"name"`
	LogPath           string           `json:"logPath"`
	Domains           []string         `json:"domains,omitempty"`
	LogType           string           `json:"logType,omitempty"`
	LogFormat         string           `json:"logFormat,omitempty"`
	LogRegex          string           `json:"logRegex,omitempty"`
	TimeLayout        string           `json:"timeLayout,omitempty"`
	Sources           []SourceConfig   `json:"sources,omitempty"`
	Whitelist         *WhitelistConfig `json:"whitelist,omitempty"`
	AutoDiscoverHosts bool             `json:"autoDiscoverHosts,omitempty"`
	CustomLabel       string           `json:"customLabel,omitempty"`
	Tags              []string         `json:"tags,omitempty"`
}

type SourceConfig struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	Mode         string            `json:"mode,omitempty"`
	PollInterval string            `json:"pollInterval,omitempty"`
	Path         string            `json:"path,omitempty"`
	Pattern      string            `json:"pattern,omitempty"`
	Compression  string            `json:"compression,omitempty"`
	Parse        *ParseConfig      `json:"parse,omitempty"`
	Host         string            `json:"host,omitempty"`
	Port         int               `json:"port,omitempty"`
	User         string            `json:"user,omitempty"`
	Auth         *SourceAuth       `json:"auth,omitempty"`
	URL          string            `json:"url,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	RangePolicy  string            `json:"rangePolicy,omitempty"`
	Index        *HTTPIndexConfig  `json:"index,omitempty"`
	Endpoint     string            `json:"endpoint,omitempty"`
	Region       string            `json:"region,omitempty"`
	Bucket       string            `json:"bucket,omitempty"`
	Prefix       string            `json:"prefix,omitempty"`
	AccessKey    string            `json:"accessKey,omitempty"`
	SecretKey    string            `json:"secretKey,omitempty"`
}

type SourceAuth struct {
	KeyFile    string `json:"keyFile,omitempty"`
	Password   string `json:"password,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}

type HTTPIndexConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	JSONMap map[string]string `json:"jsonMap,omitempty"`
}

type ParseConfig struct {
	LogType    string `json:"logType,omitempty"`
	LogFormat  string `json:"logFormat,omitempty"`
	LogRegex   string `json:"logRegex,omitempty"`
	TimeLayout string `json:"timeLayout,omitempty"`
}

type WhitelistConfig struct {
	Enabled     bool     `json:"enabled"`
	IPs         []string `json:"ips,omitempty"`
	Cities      []string `json:"cities,omitempty"`
	NonMainland bool     `json:"nonMainland"`
}

type SystemConfig struct {
	LogDestination            string             `json:"logDestination"`
	TaskInterval              string             `json:"taskInterval"` // "5m" "25s"
	BackfillMaxDurationPerRun string             `json:"backfillMaxDurationPerRun,omitempty"`
	BackfillMaxBytesPerRun    int64              `json:"backfillMaxBytesPerRun,omitempty"`
	HTTPSourceTimeout         string             `json:"httpSourceTimeout,omitempty"`
	LogRetentionDays          int                `json:"logRetentionDays"`
	ParseBatchSize            int                `json:"parseBatchSize"`
	IPGeoCacheLimit           int                `json:"ipGeoCacheLimit"`
	IPGeoAPIURL               string             `json:"ipGeoApiUrl"`
	AlertPush                 *AlertPushConfig   `json:"alertPush,omitempty"`
	DemoMode                  bool               `json:"demoMode"`
	SetupPassword             string             `json:"setupPassword,omitempty"`
	SetupPasswordClear        bool               `json:"setupPasswordClear,omitempty"`
	AccessKeys                []string           `json:"accessKeys"`
	AccessKeyExpireDays       int                `json:"accessKeyExpireDays"`
	Language                  string             `json:"language"`
	WebBasePath               string             `json:"webBasePath,omitempty"`
	MobilePWAEnabled          bool               `json:"mobilePwaEnabled"`
	ServerStatus              ServerStatusConfig `json:"serverStatus,omitempty"`
}

type ServerStatusConfig struct {
	Enabled         bool   `json:"enabled"`
	MockEnabled     bool   `json:"mockEnabled,omitempty"`
	MetricsURL      string `json:"metricsUrl,omitempty"`
	DisksURL        string `json:"disksUrl,omitempty"`
	Timeout         string `json:"timeout,omitempty"`
	RefreshInterval string `json:"refreshInterval,omitempty"`
}

type AlertPushConfig struct {
	Enabled  bool                `json:"enabled"`
	Timeout  string              `json:"timeout,omitempty"`
	Feishu   AlertWebhookConfig  `json:"feishu,omitempty"`
	DingTalk AlertDingTalkConfig `json:"dingtalk,omitempty"`
	WeCom    AlertWebhookConfig  `json:"wecom,omitempty"`
	Email    AlertEmailConfig    `json:"email,omitempty"`
}

type AlertWebhookConfig struct {
	Enabled bool   `json:"enabled"`
	Webhook string `json:"webhook,omitempty"`
}

type AlertDingTalkConfig struct {
	Enabled bool   `json:"enabled"`
	Webhook string `json:"webhook,omitempty"`
	Secret  string `json:"secret,omitempty"`
}

type AlertEmailConfig struct {
	Enabled  bool     `json:"enabled"`
	Host     string   `json:"host,omitempty"`
	Port     int      `json:"port,omitempty"`
	Username string   `json:"username,omitempty"`
	Password string   `json:"password,omitempty"`
	From     string   `json:"from,omitempty"`
	To       []string `json:"to,omitempty"`
	UseTLS   bool     `json:"useTLS,omitempty"`
}

type ServerConfig struct {
	Port string `json:"Port"`
}

type DatabaseConfig struct {
	Driver          string `json:"driver"`
	DSN             string `json:"dsn"`
	MaxOpenConns    int    `json:"maxOpenConns"`
	MaxIdleConns    int    `json:"maxIdleConns"`
	ConnMaxLifetime string `json:"connMaxLifetime"`
}

type PVFilterConfig struct {
	StatusCodeInclude []int    `json:"statusCodeInclude"`
	ExcludePatterns   []string `json:"excludePatterns"`
	ExcludeIPs        []string `json:"excludeIPs"`
}

// ReadRawConfig 读取配置（支持环境变量覆盖与默认值）但不初始化全局变量
func ReadRawConfig() (*Config, error) {
	return loadConfig()
}

// ReadConfig 读取配置文件并返回配置，同时初始化 ID 映射
func ReadConfig() *Config {
	if globalConfig != nil {
		return globalConfig
	}

	cfg, err := loadConfig()
	if err != nil {
		panic(err)
	}

	// 初始化 ID 映射
	for _, website := range cfg.Websites {
		id := generateID(website.Name)
		websiteIDMap.Store(id, website)
	}

	globalConfig = cfg
	return globalConfig
}

// GetWebsiteByID 根据 ID 获取对应的 WebsiteConfig
func GetWebsiteByID(id string) (WebsiteConfig, bool) {
	value, ok := websiteIDMap.Load(id)
	if ok {
		return value.(WebsiteConfig), true
	}
	return WebsiteConfig{}, false
}

// GetAllWebsiteIDs 获取所有网站的 ID 列表
func GetAllWebsiteIDs() []string {
	var ids []string
	websiteIDMap.Range(func(key, value interface{}) bool {
		if website, ok := value.(WebsiteConfig); ok && website.AutoDiscoverHosts {
			return true
		}
		ids = append(ids, key.(string))
		return true
	})
	return ids
}

func GetAutoDiscoverWebsiteTemplates() []WebsiteConfig {
	cfg := ReadConfig()
	templates := make([]WebsiteConfig, 0)
	for _, website := range cfg.Websites {
		if website.AutoDiscoverHosts {
			templates = append(templates, website)
		}
	}
	return templates
}

func RegisterRuntimeWebsite(website WebsiteConfig) (string, bool) {
	cfg := ReadConfig()
	website.AutoDiscoverHosts = false
	website.Name = strings.TrimSpace(website.Name)
	if website.Name == "" {
		return "", false
	}
	if len(website.Domains) == 0 {
		website.Domains = []string{website.Name}
	}

	if id, ok := findWebsiteIDByNameOrDomain(website.Name, website.Domains); ok {
		return id, false
	}

	baseID := generateID(website.Name)
	id := baseID
	for i := 1; ; i++ {
		if _, exists := websiteIDMap.Load(id); !exists {
			break
		}
		id = fmt.Sprintf("%s_%d", baseID, i)
	}

	websiteIDMap.Store(id, website)
	globalConfig = cfg
	globalConfig.Websites = append(globalConfig.Websites, website)
	return id, true
}

func UnregisterRuntimeWebsite(id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	value, ok := websiteIDMap.Load(id)
	if !ok {
		return
	}
	target, ok := value.(WebsiteConfig)
	if !ok {
		websiteIDMap.Delete(id)
		return
	}
	websiteIDMap.Delete(id)
	if globalConfig == nil {
		return
	}
	websites := globalConfig.Websites[:0]
	for _, website := range globalConfig.Websites {
		if sameWebsiteIdentity(website, target) {
			continue
		}
		websites = append(websites, website)
	}
	globalConfig.Websites = websites
}

func sameWebsiteIdentity(a, b WebsiteConfig) bool {
	if normalizeDomainForConfig(a.Name) != normalizeDomainForConfig(b.Name) {
		return false
	}
	if len(a.Domains) != len(b.Domains) {
		return false
	}
	for i := range a.Domains {
		if normalizeDomainForConfig(a.Domains[i]) != normalizeDomainForConfig(b.Domains[i]) {
			return false
		}
	}
	return true
}

func findWebsiteIDByNameOrDomain(name string, domains []string) (string, bool) {
	normalizedName := normalizeDomainForConfig(name)
	normalizedDomains := make(map[string]struct{}, len(domains))
	for _, domain := range domains {
		if normalized := normalizeDomainForConfig(domain); normalized != "" {
			normalizedDomains[normalized] = struct{}{}
		}
	}

	var foundID string
	websiteIDMap.Range(func(key, value interface{}) bool {
		website, ok := value.(WebsiteConfig)
		if !ok {
			return true
		}
		if website.AutoDiscoverHosts {
			return true
		}
		if normalizeDomainForConfig(website.Name) == normalizedName {
			foundID = key.(string)
			return false
		}
		for _, domain := range website.Domains {
			if _, ok := normalizedDomains[normalizeDomainForConfig(domain)]; ok {
				foundID = key.(string)
				return false
			}
		}
		return true
	})
	return foundID, foundID != ""
}

func normalizeDomainForConfig(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimSuffix(value, ".")
	return value
}

func GetIPGeoAPIURL() string {
	cfg := ReadConfig()
	value := strings.TrimSpace(cfg.System.IPGeoAPIURL)
	if value == "" {
		return DefaultIPGeoAPIURL
	}
	return value
}

func GetHTTPSourceTimeout() time.Duration {
	cfg := ReadConfig()
	value := strings.TrimSpace(cfg.System.HTTPSourceTimeout)
	if value == "" {
		return 2 * time.Minute
	}
	timeout, err := time.ParseDuration(value)
	if err != nil || timeout <= 0 {
		return 2 * time.Minute
	}
	return timeout
}

func GetBackfillMaxDurationPerRun() time.Duration {
	cfg := ReadConfig()
	value := strings.TrimSpace(cfg.System.BackfillMaxDurationPerRun)
	if value == "" {
		return DefaultBackfillMaxDurationPerRun
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return DefaultBackfillMaxDurationPerRun
	}
	return duration
}

func GetBackfillMaxBytesPerRun() int64 {
	cfg := ReadConfig()
	value := cfg.System.BackfillMaxBytesPerRun
	if value <= 0 {
		return DefaultBackfillMaxBytesPerRun
	}
	return value
}

// ParseInterval 解析间隔配置字符串，支持分钟(m)和秒(s)单位
func ParseInterval(intervalStr string, defaultInterval time.Duration) time.Duration {
	if intervalStr == "" {
		return defaultInterval
	}

	// 尝试解析配置的时间间隔
	duration, err := time.ParseDuration(intervalStr)
	if err != nil {
		logrus.WithField("interval", intervalStr).Info(
			"无效的时间间隔配置，使用默认值")
		return defaultInterval
	}

	minInterval := 5 * time.Second
	if duration < minInterval {
		logrus.WithField("interval", intervalStr).Info(
			"配置的时间间隔过短，已调整为最小值5秒")
		return minInterval
	}

	return duration
}

// generateID 根据输入字符串生成唯一 ID
func generateID(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:2])
}
