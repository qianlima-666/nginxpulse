---
title: "完整字段参考"
---

# 完整字段参考

本文是配置字段查询手册。新用户建议先看[配置说明](Configuration)，需要查字段含义时再回到本页。

## websites[]
- `name` (string, 必填): 站点名称，站点 ID 由该字段生成；改名会产生新站点并重新解析。
- `logPath` (string): 日志路径，支持通配符 `*`。配置 `sources` 后会被忽略。
- `domains` (string[]): 站点域名列表，用于判断站内来源，建议填写。
- `logType` (string): 内置日志类型，详见[支持的日志格式](Supported-Log-Formats)。
- `logFormat` (string): 自定义日志格式，适合 Nginx `$变量` 格式。
- `logRegex` (string): 自定义正则，必须包含命名分组。
- `timeLayout` (string): 自定义时间解析格式，留空使用内置默认值。
- `sources` (array): 多来源配置，详见[日志来源配置](Log-Sources)。
- `whitelist` (object): 白名单命中提示配置。

## websites[].whitelist
白名单按站点生效，示例：
```json
"whitelist": {
  "enabled": true,
  "ips": ["1.1.1.1", "10.0.0.0/8", "1.1.1.10-1.1.1.100"],
  "cities": ["上海", "Hangzhou"],
  "nonMainland": false
}
```

字段说明：
- `enabled` (bool): 白名单总开关。
- `ips` (string[]): IP 规则，支持单 IP、CIDR、IP 段。
- `cities` (string[]): 城市规则，包含匹配，忽略常见行政后缀差异。
- `nonMainland` (bool): 非大陆访问规则，包含海外及港澳台。

校验规则：
- `enabled=true` 时，`ips`、`cities`、`nonMainland` 至少需要配置一类。
- `ips` 中每一项必须是合法 IP/CIDR/IP 段，IP 段起始地址不能大于结束地址。

当前行为：
- 命中白名单时会写入系统通知，分类为 `whitelist`。
- 白名单只用于标记和告警提示，不会阻止日志解析和入库。

## system
- `logDestination`: `file` 或 `stdout`，默认 `file`。
- `taskInterval`: 定期任务间隔，默认 `1m`，最小 5s。
- `backfillMaxDurationPerRun`: 历史回填单轮最长时长，Go duration，默认 `8s`。实际单轮时长取 `taskInterval/3` 与该值中的较小值。
- `backfillMaxBytesPerRun`: 历史回填单轮最大字节数，默认 `33554432`（32 MiB）。
- `httpSourceTimeout`: 远程 HTTP 日志读取超时，Go duration，默认 `2m`。
- `logRetentionDays`: 保留天数，默认 30。只清理已入库访问数据，不删除原始日志文件。
- `parseBatchSize`: 单批解析条数，默认 100。
- `ipGeoCacheLimit`: IP 归属地缓存上限，默认 1000000。
- `ipGeoApiUrl`: IP 归属地远端 API 地址，默认 `http://ip-api.com/batch`。自定义 API 需遵循[IP 归属地解析](IP-Geo)中的协议定义。
- `demoMode`: 是否演示模式，默认 `false`。
- `setupPassword`: 系统配置页面与配置保存接口的独立密码。保存后以 bcrypt 哈希形式写入配置文件，留空表示不启用；读取配置接口不会返回该值。旧版本已保存的明文密码仍可校验，并会在下次保存配置时自动迁移为哈希。
- `setupPasswordClear`: 保存时用于显式清空当前系统配置密码的开关，默认 `false`。
- `accessKeys`: 访问密钥列表，默认空。
- `language`: `zh-CN` 或 `en-US`，默认 `zh-CN`。
- `webBasePath`: 前端访问前缀，仅支持单段路径。
- `mobilePwaEnabled`: 是否启用移动端 PWA。
- `serverStatus`: 服务器状态模块配置。开启后概况页会显示温度、风扇和磁盘 SMART 健康状态。
- `alertPush`: 系统通知外部推送配置。

### logRetentionDays 生效说明
- 修改后需重启服务进程/容器，解析器才会按新值过滤入库日志。
- 如需立即按新值重建历史数据，重启后点击“重新解析”。
- 如果设置了 `LOG_RETENTION_DAYS` 环境变量，会覆盖配置文件中的值。

### webBasePath
示例：
```json
{
  "system": {
    "webBasePath": "nginxpulse"
  }
}
```

效果：
- Web：`/nginxpulse/`
- Mobile：`/nginxpulse/m/`
- API：`/nginxpulse/api/`

限制：
- 只支持单段路径，如 `nginxpulse`。
- 修改后需要重启服务。
- 设置后根路径 `/` 将不可访问。

### 移动端底部导航栏 URL 覆盖
移动端 `/m/` 支持通过 URL 参数临时覆盖导航栏位置：
- 参数优先级：`tabbarBottom` 高于 `tabbar`。
- 真值（底部导航）：`1`、`true`、`yes`、`on`、`bottom`。
- 假值（顶部导航）：`0`、`false`、`no`、`off`、`top`。
- URL 中出现上述参数时，会写入本地存储并在后续页面跳转中持续生效。

示例：
```bash
# 强制底部导航
https://example.com/m/?tabbarBottom=true
# 强制顶部导航
https://example.com/m/?tabbarBottom=false
```

### alertPush
系统通知外部推送配置，支持飞书、钉钉、企微机器人和邮件。修改后需重启服务生效。

示例：
```json
{
  "enabled": true,
  "timeout": "5s",
  "feishu": { "enabled": true, "webhook": "https://open.feishu.cn/open-apis/bot/v2/hook/xxx" },
  "dingtalk": { "enabled": false, "webhook": "", "secret": "" },
  "wecom": { "enabled": false, "webhook": "" },
  "email": {
    "enabled": false,
    "host": "smtp.example.com",
    "port": 465,
    "username": "alert@example.com",
    "password": "password",
    "from": "alert@example.com",
    "to": ["ops@example.com"],
    "useTLS": true
  }
}
```

### serverStatus
服务器状态模块在概况页展示主机传感器和磁盘 SMART 信息。该模块由运行参数/系统配置开启；在 UI 中对应“系统配置”里的“运行参数”步骤。NginxPulse 后端会代替浏览器请求传感器接口和磁盘接口，因此接口地址只需要能被 NginxPulse 服务端访问。

示例：
```json
{
  "system": {
    "serverStatus": {
      "enabled": true,
      "mockEnabled": false,
      "metricsUrl": "http://192.168.6.150:9911/json",
      "disksUrl": "http://192.168.6.150:9911/disks",
      "timeout": "5s",
      "refreshInterval": "30s"
    }
  }
}
```

字段说明：
- `enabled` (bool): 是否开启服务器状态模块，默认 `false`。
- `mockEnabled` (bool): 是否使用后端内置模拟数据。仅用于预览 UI 或开发调试；开启后不会请求 `metricsUrl` 和 `disksUrl`。
- `metricsUrl` (string): 传感器接口地址，必须是 `http` 或 `https`。返回 CPU 温度、主板温度、NVMe 温度、风扇转速等指标。
- `disksUrl` (string): 磁盘 SMART 接口地址，必须是 `http` 或 `https`。返回磁盘列表、健康状态、温度、寿命、错误计数等指标。
- `timeout` (string): 后端请求上游接口的超时时间，Go duration 格式，如 `5s`、`10s`。
- `refreshInterval` (string): 概况页自动刷新服务器状态的间隔，Go duration 格式，如 `30s`、`1m`。

请求行为：
- NginxPulse 通过 `GET` 请求 `metricsUrl` 和 `disksUrl`，并发送 `Accept: application/json`。
- 上游接口 HTTP 状态码必须是 `2xx`。
- 单个接口响应体最大读取 2 MiB。
- `metricsUrl` 或 `disksUrl` 任一失败时，概况页会显示“部分可用”；两个都失败时显示“异常”。
- 上游返回中的 `status` 字段不是 `ok` 时，概况页会显示“需关注”。

#### 传感器接口返回结构
传感器接口需要返回 JSON 对象。推荐结构如下：

```json
{
  "status": "ok",
  "updated_at": "2026-04-25T12:30:00+08:00",
  "metrics": {
    "cpu_temp_celsius": 52.4,
    "board_temp_celsius": 34.8,
    "nvme_temp_celsius": 62.7,
    "cpu_fan_rpm": 940,
    "chassis_fan1_rpm": 720
  },
  "missing_metrics": []
}
```

顶层字段：
- `status` (string, 可选): 上游状态。建议返回 `ok`；返回其他值时，NginxPulse 会把服务器状态标记为“需关注”。
- `updated_at` (string, 可选): 数据更新时间。建议使用 RFC3339/ISO 8601 时间字符串。
- `metrics` (object): 传感器指标集合。
- `missing_metrics` (string[], 可选): 无法采集的指标名称列表，会在概况页提示。

`metrics` 字段：
- `cpu_temp_celsius` (number): CPU 温度，单位摄氏度，用于 CPU 温度卡片和健康度计算。
- `board_temp_celsius` (number): 主板或机箱环境温度，单位摄氏度。
- `nvme_temp_celsius` (number): NVMe 温度，单位摄氏度。页面会和磁盘列表中的最高温度一起取较高值展示。
- `cpu_fan_rpm` (number): CPU 风扇转速，单位 RPM。
- `chassis_fan1_rpm` (number): 机箱风扇转速，单位 RPM。

说明：
- 所有 `metrics` 字段都可以缺省；缺省时对应位置会显示为空值。
- 如果有多个机箱风扇，目前首页只读取 `chassis_fan1_rpm`。
- 温度和风扇值建议返回数字，不要带单位字符串。

#### 磁盘接口返回结构
磁盘接口需要返回 JSON 对象。推荐结构如下：

```json
{
  "status": "ok",
  "updated_at": "2026-04-25T12:30:00+08:00",
  "disk_count": 2,
  "disks": [
    {
      "name": "nvme0n1",
      "path": "/dev/nvme0n1",
      "smartctl_path": "/dev/nvme0",
      "type": "nvme",
      "model": "Samsung 980 PRO 2TB",
      "serial": "S6AXNS0T900123A",
      "firmware_version": "5B2QGXA7",
      "smartctl_exit_status": 0,
      "size_bytes": 2000398934016,
      "smart_available": true,
      "smart_enabled": true,
      "health_passed": true,
      "temperature_celsius": 62.7,
      "percentage_used": 18,
      "percentage_remaining": 82,
      "media_errors": 0,
      "error_log_entries": 3,
      "unsafe_shutdowns": 8,
      "power_on_hours": 4680,
      "power_cycles": 72,
      "data_units_read_bytes": 12884901888000,
      "data_units_written_bytes": 7421703487488
    },
    {
      "name": "sdb",
      "path": "/dev/sdb",
      "smartctl_path": "/dev/sdb",
      "type": "sat",
      "model": "WD Blue SA510 1TB",
      "serial": "WD-WXK2A2390001",
      "firmware_version": "52040100",
      "smartctl_exit_status": 0,
      "size_bytes": 1000204886016,
      "smart_available": true,
      "smart_enabled": true,
      "health_passed": true,
      "temperature_celsius": 41,
      "percentage_used": 64,
      "percentage_remaining": 36,
      "media_errors": 2,
      "error_log_entries": 9,
      "unsafe_shutdowns": 1,
      "power_on_hours": 12840,
      "power_cycles": 122,
      "data_units_read_bytes": 9455799992320,
      "data_units_written_bytes": 6120328396800
    }
  ]
}
```

顶层字段：
- `status` (string, 可选): 上游状态。建议返回 `ok`；返回其他值时，NginxPulse 会把服务器状态标记为“需关注”。
- `updated_at` (string, 可选): 数据更新时间。传感器接口和磁盘接口都返回时，页面优先使用传感器接口的时间。
- `disk_count` (number, 可选): 磁盘总数。缺省时使用 `disks.length`。
- `disks` (array): 磁盘 SMART 信息列表。

`disks[]` 字段：
- `name` (string): 设备名，如 `nvme0n1`、`sdb`。
- `path` (string): 操作系统中的设备路径，如 `/dev/nvme0n1`。
- `smartctl_path` (string): `smartctl` 查询路径，如 `/dev/nvme0`。
- `type` (string): 磁盘类型，如 `nvme`、`sat`。
- `model` (string): 磁盘型号，首页和弹窗中优先显示。
- `serial` (string): 序列号，用于排查具体设备。
- `firmware_version` (string): 固件版本。
- `smartctl_exit_status` (number): `smartctl` 退出码，便于定位采集异常。
- `size_bytes` (number): 磁盘容量，单位字节。
- `smart_available` (bool): 设备是否支持 SMART。
- `smart_enabled` (bool): SMART 是否已启用。
- `health_passed` (bool): SMART 健康检查是否通过。为 `false` 时会被标记为异常。
- `temperature_celsius` (number): 磁盘温度，单位摄氏度。
- `percentage_used` (number): 已使用寿命百分比，常见于 NVMe。
- `percentage_remaining` (number): 剩余寿命百分比。页面优先用它展示健康度；低于 20 会标记为异常，低于 60 会标记为关注。
- `media_errors` (number): 介质错误计数，大于 0 会标记为异常。
- `error_log_entries` (number): 错误日志条目数量，大于 0 会标记为关注。
- `unsafe_shutdowns` (number): 异常断电次数。
- `power_on_hours` (number): 通电小时数。
- `power_cycles` (number): 通电次数。
- `data_units_read_bytes` (number): 累计读取量，单位字节。
- `data_units_written_bytes` (number): 累计写入量，单位字节。

说明：
- 字段可以按设备能力缺省；缺省字段在页面中显示为空值。
- `percentage_remaining`、`health_passed`、`temperature_celsius`、`media_errors`、`error_log_entries` 会影响风险排序和状态颜色。
- 如果只能拿到 `percentage_used`，建议同时在接口中计算并返回 `percentage_remaining = 100 - percentage_used`。

## database
- `driver`: 固定为 `postgres`。
- `dsn`: PostgreSQL DSN，必填。
- `maxOpenConns`: 最大连接数。
- `maxIdleConns`: 最大空闲连接数。
- `connMaxLifetime`: 连接最大生命周期，Go duration。

示例：
```json
{
  "database": {
    "driver": "postgres",
    "dsn": "postgres://nginxpulse:nginxpulse@127.0.0.1:5432/nginxpulse?sslmode=disable",
    "maxOpenConns": 10,
    "maxIdleConns": 5,
    "connMaxLifetime": "30m"
  }
}
```

## server
- `Port`: API/Web 监听端口，默认 `:8089`。

示例：
```json
{
  "server": {
    "Port": ":8089"
  }
}
```

## pvFilter
PV 过滤规则用于决定哪些请求计入页面浏览量。

- `statusCodeInclude`: 计入 PV 的状态码数组，默认 `[200]`。
- `excludePatterns`: 排除 URL 的正则数组。
- `excludeIPs`: 排除 IP 列表。

示例：
```json
{
  "pvFilter": {
    "statusCodeInclude": [200],
    "excludePatterns": [
      "favicon.ico$",
      "robots.txt$",
      "sitemap.xml$",
      "^/health$",
      "^/_(?:nuxt|next)/"
    ],
    "excludeIPs": ["127.0.0.1", "::1"]
  }
}
```

## 环境变量覆盖
以下环境变量可覆盖配置：
- `CONFIG_JSON`: 完整配置 JSON 字符串。
- `WEBSITES`: 仅网站数组 JSON 字符串。
- `LOG_DEST`
- `TASK_INTERVAL`
- `HTTP_SOURCE_TIMEOUT`
- `LOG_RETENTION_DAYS`
- `LOG_PARSE_BATCH_SIZE`
- `IP_GEO_CACHE_LIMIT`
- `IP_GEO_API_URL`
- `DEMO_MODE`
- `ACCESS_KEYS`
- `APP_LANGUAGE`
- `SERVER_STATUS_ENABLED`
- `SERVER_STATUS_MOCK_ENABLED`
- `SERVER_STATUS_METRICS_URL`
- `SERVER_STATUS_DISKS_URL`
- `SERVER_STATUS_TIMEOUT`
- `SERVER_STATUS_REFRESH_INTERVAL`
- `SERVER_PORT`
- `PV_STATUS_CODES`
- `PV_EXCLUDE_PATTERNS`
- `PV_EXCLUDE_IPS`
- `DB_DRIVER`
- `DB_DSN`
- `DB_MAX_OPEN_CONNS`
- `DB_MAX_IDLE_CONNS`
- `DB_CONN_MAX_LIFETIME`

示例：
```bash
export CONFIG_JSON="$(cat configs/nginxpulse_config.json)"
export LOG_PARSE_BATCH_SIZE=1000
export DB_DSN="postgres://nginxpulse:nginxpulse@127.0.0.1:5432/nginxpulse?sslmode=disable"
```
