# Supported Log Formats

This page lists the built-in `logType` values, aliases, config examples, and sample log lines supported by NginxPulse. If your format is not listed, use `logFormat` or `logRegex`.

## Overview
| logType | Aliases | Default time layout | Notes |
|---|---|---|---|
| `nginx` | - | `02/Jan/2006:15:04:05 -0700` | Nginx combined/access log, with optional trace fields |
| `tengine` | - | `02/Jan/2006:15:04:05 -0700` | Parsed as Nginx |
| `apache` | `httpd`, `apache-httpd` | `02/Jan/2006:15:04:05 -0700` | Apache combined log |
| `caddy` | - | auto-detects `ts`/`time`/`timestamp` | Caddy JSON access log |
| `nginx-proxy-manager` | `npm` | timestamp inside `[...]` | Nginx Proxy Manager access log |
| `iis` | `iis-w3c`, `w3c-iis` | `2006-01-02 15:04:05` | IIS W3C Extended |
| `haproxy` | `haproxy-ingress` | `02/Jan/2006:15:04:05.000` | HAProxy HTTP log |
| `traefik` | `traefik-ingress` | `02/Jan/2006:15:04:05 -0700` | Traefik common log |
| `envoy` | - | RFC3339/RFC3339Nano | Envoy access log |
| `nginx-ingress` | `ingress-nginx` | `02/Jan/2006:15:04:05 -0700` | NGINX Ingress Controller |
| `safeline` | `safeline-waf`, `raywaf`, `ray-waf`, `leichi`, `leichi-waf` | `02/Jan/2006:15:04:05 -0700` | SafeLine WAF |
| `zoraxy` | `zoraxy-router` | `2006-01-02 15:04:05.000000` | Zoraxy router request log |

Generic config:
```json
{
  "name": "site-name",
  "logPath": "/path/to/access.log",
  "logType": "nginx"
}
```

## Nginx
Config:
```json
{
  "name": "nginx-site",
  "logPath": "/var/log/nginx/access.log",
  "logType": "nginx"
}
```

Sample:
```text
203.0.113.10 - - [24/Apr/2026:10:05:34 +0800] "GET /index.html HTTP/1.1" 200 512 "https://example.com/" "Mozilla/5.0"
```

Extended trace sample:
```text
203.0.113.10 - - [24/Apr/2026:10:05:34 +0800] "GET /orders?id=42 HTTP/2.0" 200 512 "-" "curl/8.0.1" 128 0.245 0.200 10.0.0.2:8080 app.example.com req-123
```

Parsed fields: IP, time, method, URL, status, response bytes, Referer, User-Agent. Optional trace fields can include request length, request time, upstream time, upstream address, host, and request ID.

## Tengine
Tengine is parsed with the Nginx default parser:
```json
{
  "name": "tengine-site",
  "logPath": "/var/log/tengine/access.log",
  "logType": "tengine"
}
```

Sample:
```text
203.0.113.11 - - [24/Apr/2026:10:05:35 +0800] "POST /api/login HTTP/1.1" 302 64 "-" "Mozilla/5.0"
```

## Apache httpd
Config:
```json
{
  "name": "apache-site",
  "logPath": "/var/log/apache2/access.log",
  "logType": "apache"
}
```

Sample:
```text
203.0.113.12 - - [24/Apr/2026:10:05:36 +0800] "GET /docs HTTP/1.1" 200 2048 "https://example.com/" "Mozilla/5.0"
```

## Caddy
Caddy uses JSON access logs:
```json
{
  "name": "caddy-site",
  "logPath": "/var/log/caddy/access.json",
  "logType": "caddy"
}
```

Sample:
```json
{"level":"info","ts":1777000000.123456,"request":{"remote_ip":"203.0.113.13","method":"GET","uri":"/api/items","host":"api.example.com","headers":{"User-Agent":["curl/8.0.1"],"Referer":["https://example.com/"]}},"status":200,"size":128,"duration":0.034}
```

Notes: time is read from `ts`, `time`, or `timestamp`; duration is read from `duration`, `duration_ms`, or `duration_ns`.

## Nginx Proxy Manager
Config:
```json
{
  "name": "npm-site",
  "logPath": "/data/logs/proxy-host-1_access.log",
  "logType": "nginx-proxy-manager"
}
```

Sample:
```text
[24/Apr/2026:10:05:37 +0800] - 200 200 - GET http app.example.com "/assets/app.js" [Client 203.0.113.14] [Length 4096] [Gzip -] [Sent-to 10.0.0.4] "Mozilla/5.0" "https://app.example.com/"
```

## IIS W3C Extended
Config:
```json
{
  "name": "iis-site",
  "logPath": "/var/log/iis/u_ex*.log",
  "logType": "iis"
}
```

Field order:
```text
date time s-ip cs-method cs-uri-stem cs-uri-query s-port cs-username c-ip cs(User-Agent) cs(Referer) sc-status sc-substatus sc-win32-status time-taken
```

Sample:
```text
2026-04-24 10:05:38 10.0.0.10 GET /index.html a=1&b=2 443 - 203.0.113.15 Mozilla/5.0+(Windows+NT+10.0;+Win64;+x64) https://example.com/ 200 0 0 36
```

Notes:
- Metadata lines such as `#Software`, `#Version`, and `#Fields` are skipped automatically.
- When `cs-uri-query` is not `-`, it is appended as `path?query`.
- IIS W3C timestamps are usually UTC.

## HAProxy
Config:
```json
{
  "name": "haproxy-site",
  "logPath": "/var/log/haproxy.log",
  "logType": "haproxy"
}
```

Sample:
```text
Apr 24 10:05:39 lb haproxy[1234]: 203.0.113.16:53124 [24/Apr/2026:10:05:39.123] fe_http be_app/app1 0/0/1/12/13 200 1024 - - ---- 1/1/0/0/0 0/0 "GET /health HTTP/1.1"
```

## Traefik
Config:
```json
{
  "name": "traefik-site",
  "logPath": "/var/log/traefik/access.log",
  "logType": "traefik"
}
```

Sample:
```text
203.0.113.17 - - [24/Apr/2026:10:05:40 +0800] "GET / HTTP/1.1" 200 612 "-" "Mozilla/5.0" 1 "app@docker" "http://10.0.0.5:8080" 12ms
```

## Envoy
Config:
```json
{
  "name": "envoy-site",
  "logPath": "/var/log/envoy/access.log",
  "logType": "envoy"
}
```

Sample:
```text
[2026-04-24T10:05:41.123Z] "GET /api HTTP/2" 200 - 0 256 15 14 "203.0.113.18" "curl/8.0.1" "req-abc" "api.example.com" "10.0.0.6:8080"
```

## NGINX Ingress Controller
Config:
```json
{
  "name": "ingress-site",
  "logPath": "/var/log/nginx/ingress.log",
  "logType": "nginx-ingress"
}
```

Sample:
```text
203.0.113.19 - - [24/Apr/2026:10:05:42 +0800] "GET /orders?id=42 HTTP/2.0" 200 512 "-" "curl/8.0.1" 128 0.245 [default-app-80] [] 10.0.0.7:8080 512 0.200 200 req-123
```

## SafeLine WAF
Config:
```json
{
  "name": "safeline-site",
  "logPath": "/var/log/safeline/access.log",
  "logType": "safeline"
}
```

Space-separated sample:
```text
192.168.1.242 - - [24/Apr/2026:10:05:43 +0800] "app.example.com" "GET /api/rules HTTP/2.0" 200 36 "-" "Mozilla/5.0" "-"
```

Pipe-separated sample:
```text
203.0.113.20 | - | 24/Apr/2026:10:05:43 +0800 | "app.example.com" | "POST /api/v1/tunnel HTTP/1.1" | 468 | 14862 | "-" | "Mozilla/5.0"
```

## Zoraxy
Config:
```json
{
  "name": "zoraxy-site",
  "logPath": "/opt/zoraxy/log/*.log",
  "logType": "zoraxy"
}
```

Format:
```text
[time] [router:<class>] [origin:<host>] [client: <ip>] [useragent: <ua>] METHOD URI STATUS
```

Sample:
```text
[2026-04-24 10:05:44.123456] [router:host-http] [origin:app.example.com] [client: 203.0.113.21] [useragent: Mozilla/5.0] GET /index.html 200
```

Notes:
- Only `[router:...]` request logs are parsed; Zoraxy `[system:...]` system logs are skipped automatically.
- Zoraxy logs do not include response bytes or Referer, so those fields keep their default values.

## Custom Formats
If no built-in parser matches your logs, prefer `logFormat`:
```json
{
  "name": "custom-site",
  "logPath": "/var/log/custom/access.log",
  "logType": "nginx",
  "logFormat": "$remote_addr [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\""
}
```

You can also use `logRegex`, which must include named groups:
```json
{
  "name": "regex-site",
  "logPath": "/var/log/custom/access.log",
  "logRegex": "^(?P<ip>\\S+) \\[(?P<time>[^\\]]+)\\] \"(?P<request>[^\"]*)\" (?P<status>\\d{3}) (?P<bytes>\\d+|-)$"
}
```

Required fields:
- IP: `ip`, `remote_addr`, `client_ip`, `http_x_forwarded_for`
- Time: `time`, `time_local`, `time_iso8601`
- Status: `status`
- URL or Request: `url`, `request_uri`, `uri`, `path`, `request`, `request_line`
