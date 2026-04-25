---
title: "支持的日志格式"
---

# 支持的日志格式

本文列出 NginxPulse 内置支持的 `logType`、别名、示例配置和示例日志。若你的日志格式不在列表中，可使用 `logFormat` 或 `logRegex` 自定义解析规则。

## 总览
| logType | 别名 | 默认时间格式 | 说明 |
|---|---|---|---|
| `nginx` | - | `02/Jan/2006:15:04:05 -0700` | Nginx combined/access log，支持部分扩展追踪字段 |
| `tengine` | - | `02/Jan/2006:15:04:05 -0700` | 按 Nginx 默认格式解析 |
| `apache` | `httpd`, `apache-httpd` | `02/Jan/2006:15:04:05 -0700` | Apache combined log |
| `caddy` | - | 自动识别 `ts`/`time`/`timestamp` | Caddy JSON access log |
| `nginx-proxy-manager` | `npm` | NPM 日志中的 `[...]` 时间 | Nginx Proxy Manager access log |
| `iis` | `iis-w3c`, `w3c-iis` | `2006-01-02 15:04:05` | IIS W3C Extended |
| `haproxy` | `haproxy-ingress` | `02/Jan/2006:15:04:05.000` | HAProxy HTTP log |
| `traefik` | `traefik-ingress` | `02/Jan/2006:15:04:05 -0700` | Traefik common log |
| `envoy` | - | RFC3339/RFC3339Nano | Envoy access log |
| `nginx-ingress` | `ingress-nginx` | `02/Jan/2006:15:04:05 -0700` | NGINX Ingress Controller |
| `safeline` | `safeline-waf`, `raywaf`, `ray-waf`, `leichi`, `leichi-waf` | `02/Jan/2006:15:04:05 -0700` | SafeLine/雷池 WAF |
| `zoraxy` | `zoraxy-router` | `2006-01-02 15:04:05.000000` | Zoraxy router request log |

通用配置示例：
```json
{
  "name": "site-name",
  "logPath": "/path/to/access.log",
  "logType": "nginx"
}
```

## Nginx
配置：
```json
{
  "name": "nginx-site",
  "logPath": "/var/log/nginx/access.log",
  "logType": "nginx"
}
```

示例日志：
```text
203.0.113.10 - - [24/Apr/2026:10:05:34 +0800] "GET /index.html HTTP/1.1" 200 512 "https://example.com/" "Mozilla/5.0"
```

扩展追踪字段示例：
```text
203.0.113.10 - - [24/Apr/2026:10:05:34 +0800] "GET /orders?id=42 HTTP/2.0" 200 512 "-" "curl/8.0.1" 128 0.245 0.200 10.0.0.2:8080 app.example.com req-123
```

可解析字段：IP、时间、方法、URL、状态码、响应字节、Referer、User-Agent；扩展字段可包含请求长度、请求耗时、上游耗时、上游地址、Host、Request ID。

## Tengine
Tengine 默认按 Nginx 格式解析：
```json
{
  "name": "tengine-site",
  "logPath": "/var/log/tengine/access.log",
  "logType": "tengine"
}
```

示例日志：
```text
203.0.113.11 - - [24/Apr/2026:10:05:35 +0800] "POST /api/login HTTP/1.1" 302 64 "-" "Mozilla/5.0"
```

## Apache httpd
配置：
```json
{
  "name": "apache-site",
  "logPath": "/var/log/apache2/access.log",
  "logType": "apache"
}
```

示例日志：
```text
203.0.113.12 - - [24/Apr/2026:10:05:36 +0800] "GET /docs HTTP/1.1" 200 2048 "https://example.com/" "Mozilla/5.0"
```

## Caddy
Caddy 使用 JSON access log：
```json
{
  "name": "caddy-site",
  "logPath": "/var/log/caddy/access.json",
  "logType": "caddy"
}
```

示例日志：
```json
{"level":"info","ts":1777000000.123456,"request":{"remote_ip":"203.0.113.13","method":"GET","uri":"/api/items","host":"api.example.com","headers":{"User-Agent":["curl/8.0.1"],"Referer":["https://example.com/"]}},"status":200,"size":128,"duration":0.034}
```

注意：时间可从 `ts`、`time` 或 `timestamp` 读取；耗时可从 `duration`、`duration_ms` 或 `duration_ns` 读取。

## Nginx Proxy Manager
配置：
```json
{
  "name": "npm-site",
  "logPath": "/data/logs/proxy-host-1_access.log",
  "logType": "nginx-proxy-manager"
}
```

示例日志：
```text
[24/Apr/2026:10:05:37 +0800] - 200 200 - GET http app.example.com "/assets/app.js" [Client 203.0.113.14] [Length 4096] [Gzip -] [Sent-to 10.0.0.4] "Mozilla/5.0" "https://app.example.com/"
```

## IIS W3C Extended
配置：
```json
{
  "name": "iis-site",
  "logPath": "/var/log/iis/u_ex*.log",
  "logType": "iis"
}
```

字段顺序：
```text
date time s-ip cs-method cs-uri-stem cs-uri-query s-port cs-username c-ip cs(User-Agent) cs(Referer) sc-status sc-substatus sc-win32-status time-taken
```

示例日志：
```text
2026-04-24 10:05:38 10.0.0.10 GET /index.html a=1&b=2 443 - 203.0.113.15 Mozilla/5.0+(Windows+NT+10.0;+Win64;+x64) https://example.com/ 200 0 0 36
```

注意：
- `#Software`、`#Version`、`#Fields` 等元数据行会自动跳过。
- 当 `cs-uri-query` 不是 `-` 时，会自动拼接为 `path?query`。
- IIS W3C 时间通常是 UTC。

## HAProxy
配置：
```json
{
  "name": "haproxy-site",
  "logPath": "/var/log/haproxy.log",
  "logType": "haproxy"
}
```

示例日志：
```text
Apr 24 10:05:39 lb haproxy[1234]: 203.0.113.16:53124 [24/Apr/2026:10:05:39.123] fe_http be_app/app1 0/0/1/12/13 200 1024 - - ---- 1/1/0/0/0 0/0 "GET /health HTTP/1.1"
```

## Traefik
配置：
```json
{
  "name": "traefik-site",
  "logPath": "/var/log/traefik/access.log",
  "logType": "traefik"
}
```

示例日志：
```text
203.0.113.17 - - [24/Apr/2026:10:05:40 +0800] "GET / HTTP/1.1" 200 612 "-" "Mozilla/5.0" 1 "app@docker" "http://10.0.0.5:8080" 12ms
```

## Envoy
配置：
```json
{
  "name": "envoy-site",
  "logPath": "/var/log/envoy/access.log",
  "logType": "envoy"
}
```

示例日志：
```text
[2026-04-24T10:05:41.123Z] "GET /api HTTP/2" 200 - 0 256 15 14 "203.0.113.18" "curl/8.0.1" "req-abc" "api.example.com" "10.0.0.6:8080"
```

## NGINX Ingress Controller
配置：
```json
{
  "name": "ingress-site",
  "logPath": "/var/log/nginx/ingress.log",
  "logType": "nginx-ingress"
}
```

示例日志：
```text
203.0.113.19 - - [24/Apr/2026:10:05:42 +0800] "GET /orders?id=42 HTTP/2.0" 200 512 "-" "curl/8.0.1" 128 0.245 [default-app-80] [] 10.0.0.7:8080 512 0.200 200 req-123
```

## SafeLine / 雷池 WAF
配置：
```json
{
  "name": "safeline-site",
  "logPath": "/var/log/safeline/access.log",
  "logType": "safeline"
}
```

空格分隔示例：
```text
192.168.1.242 - - [24/Apr/2026:10:05:43 +0800] "app.example.com" "GET /api/rules HTTP/2.0" 200 36 "-" "Mozilla/5.0" "-"
```

管道分隔示例：
```text
203.0.113.20 | - | 24/Apr/2026:10:05:43 +0800 | "app.example.com" | "POST /api/v1/tunnel HTTP/1.1" | 468 | 14862 | "-" | "Mozilla/5.0"
```

## Zoraxy
配置：
```json
{
  "name": "zoraxy-site",
  "logPath": "/opt/zoraxy/log/*.log",
  "logType": "zoraxy"
}
```

格式：
```text
[time] [router:<class>] [origin:<host>] [client: <ip>] [useragent: <ua>] METHOD URI STATUS
```

示例日志：
```text
[2026-04-24 10:05:44.123456] [router:host-http] [origin:app.example.com] [client: 203.0.113.21] [useragent: Mozilla/5.0] GET /index.html 200
```

注意：
- 只解析 `[router:...]` 请求日志；Zoraxy 的 `[system:...]` 系统日志会自动跳过。
- Zoraxy 日志没有响应字节数和 Referer 字段，这些字段会保持默认值。

## 自定义格式
如果内置格式不匹配，可优先使用 `logFormat`：
```json
{
  "name": "custom-site",
  "logPath": "/var/log/custom/access.log",
  "logType": "nginx",
  "logFormat": "$remote_addr [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\""
}
```

也可以使用 `logRegex`，必须包含命名分组：
```json
{
  "name": "regex-site",
  "logPath": "/var/log/custom/access.log",
  "logRegex": "^(?P<ip>\\S+) \\[(?P<time>[^\\]]+)\\] \"(?P<request>[^\"]*)\" (?P<status>\\d{3}) (?P<bytes>\\d+|-)$"
}
```

必要字段：
- IP: `ip`, `remote_addr`, `client_ip`, `http_x_forwarded_for`
- Time: `time`, `time_local`, `time_iso8601`
- Status: `status`
- URL 或 Request: `url`, `request_uri`, `uri`, `path`, `request`, `request_line`
