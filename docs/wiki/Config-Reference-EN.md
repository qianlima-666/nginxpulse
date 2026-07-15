# Config Reference

This page is the field reference. New users should start with [Configuration](Configuration-EN), then come back here when they need details.

## websites[]
- `name` (string, required): site name. The site ID is derived from this field; renaming creates a new site and reparses logs.
- `logPath` (string): log path, supports `*` glob. Ignored when `sources` is configured.
- `domains` (string[]): site domains, recommended for same-site referer detection.
- `logType` (string): built-in log type. See [Supported Log Formats](Supported-Log-Formats-EN).
- `logFormat` (string): custom log format, suitable for Nginx `$variable` formats.
- `logRegex` (string): custom regex, must include named groups.
- `timeLayout` (string): custom time layout. Empty means built-in default.
- `sources` (array): multi-source config. See [Log Sources](Log-Sources-EN).
- `whitelist` (object): whitelist hit notification config.

## websites[].whitelist
Whitelist rules are site-scoped:
```json
"whitelist": {
  "enabled": true,
  "ips": ["1.1.1.1", "10.0.0.0/8", "1.1.1.10-1.1.1.100"],
  "cities": ["Shanghai", "Hangzhou"],
  "nonMainland": false
}
```

Fields:
- `enabled` (bool): main switch.
- `ips` (string[]): IP rules. Supports single IP, CIDR, and IP ranges.
- `cities` (string[]): city rules, matched by containment with common administrative suffix normalization.
- `nonMainland` (bool): non-mainland visits, including overseas, Hong Kong, Macao, and Taiwan.

Validation:
- When `enabled=true`, at least one of `ips`, `cities`, or `nonMainland` must be configured.
- Each `ips` entry must be a valid IP/CIDR/range, and range start cannot be greater than range end.

Current behavior:
- Whitelist hits create system notifications with category `whitelist`.
- Whitelist currently marks and alerts only. It does not block parsing or database writes.

## system
- `logDestination`: `file` or `stdout`, default `file`.
- `taskInterval`: periodic task interval, default `1m`, minimum 5s.
- `backfillMaxDurationPerRun`: max historical backfill duration per run, Go duration, default `8s`.
- `backfillMaxBytesPerRun`: max historical backfill bytes per run, default `33554432` (32 MiB).
- `httpSourceTimeout`: remote HTTP log read timeout, Go duration, default `2m`.
- `logRetentionDays`: retention days, default 30. Cleans parsed access data only; does not delete raw log files.
- `parseBatchSize`: parse batch size, default 100.
- `ipGeoCacheLimit`: max IP geo cache entries, default 1000000.
- `ipGeoApiUrl`: remote IP geo API URL, default `http://ip-api.com/batch`. Custom APIs must follow the contract in [IP Geo](IP-Geo-EN).
- `demoMode`: demo mode, default `false`.
- `setupPassword`: standalone password for the Settings page and config-save API. Empty means disabled.
- `setupPasswordClear`: explicit switch used on save to clear the current settings password, default `false`.
- `accessKeys`: access key list, default empty.
- `language`: `zh-CN` or `en-US`, default `zh-CN`.
- `webBasePath`: frontend base path, one path segment only.
- `mobilePwaEnabled`: enable mobile PWA.
- `alertPush`: external system notification push config.

### logRetentionDays Notes
- Restart the service/container after changing it.
- To rebuild existing historical data with the new value, restart and click “Reparse”.
- `LOG_RETENTION_DAYS` overrides this field.

### webBasePath
Example:
```json
{
  "system": {
    "webBasePath": "nginxpulse"
  }
}
```

Result:
- Web: `/nginxpulse/`
- Mobile: `/nginxpulse/m/`
- API: `/nginxpulse/api/`

Limits:
- Only one path segment is supported, such as `nginxpulse`.
- Restart after changing it.
- The root path `/` is disabled when this is set.

### Mobile Tabbar URL Override
Mobile `/m/` supports temporary URL overrides:
- `tabbarBottom` has higher priority than `tabbar`.
- Truthy bottom values: `1`, `true`, `yes`, `on`, `bottom`.
- Falsy top values: `0`, `false`, `no`, `off`, `top`.
- When present, the value is stored locally and remains active during later navigation.

Example:
```bash
# force bottom navigation
https://example.com/m/?tabbarBottom=true
# force top navigation
https://example.com/m/?tabbarBottom=false
```

### alertPush
External system notification push config. Supports Feishu, DingTalk, WeCom bot, and email. Restart after changing it.

Example:
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

## database
- `driver`: must be `postgres`.
- `dsn`: PostgreSQL DSN, required.
- `maxOpenConns`: max open connections.
- `maxIdleConns`: max idle connections.
- `connMaxLifetime`: max connection lifetime, Go duration.

Example:
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
- `Port`: API/Web listen port, default `:8089`.

Example:
```json
{
  "server": {
    "Port": ":8089"
  }
}
```

## pvFilter
PV filters decide which requests count as page views.

- `statusCodeInclude`: status codes counted as PV, default `[200]`.
- `excludePatterns`: URL regex list to exclude.
- `excludeIPs`: IP list to exclude.

Example:
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

## Environment Overrides
Supported env vars:
- `CONFIG_JSON`: full config JSON string.
- `WEBSITES`: websites array JSON string only.
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
- `SERVER_PORT`
- `PV_STATUS_CODES`
- `PV_EXCLUDE_PATTERNS`
- `PV_EXCLUDE_IPS`
- `DB_DRIVER`
- `DB_DSN`
- `DB_MAX_OPEN_CONNS`
- `DB_MAX_IDLE_CONNS`
- `DB_CONN_MAX_LIFETIME`

Example:
```bash
export CONFIG_JSON="$(cat configs/nginxpulse_config.json)"
export LOG_PARSE_BATCH_SIZE=1000
export DB_DSN="postgres://nginxpulse:nginxpulse@127.0.0.1:5432/nginxpulse?sslmode=disable"
```
