---
title: "Log Sources"
---

# Log Sources

This page explains where logs come from: local files, globs, compressed logs, remote pulling, object storage, and Push Agent.

## Choosing A Source
| Scenario | Recommended config |
|---|---|
| Logs are local or inside the container | `websites[].logPath` |
| Logs are rotated daily | `logPath` glob |
| `.gz` compressed logs | `logPath` or `sources[].compression` |
| Multiple hosts or remote directories | `websites[].sources` + `sftp` |
| Logs exposed over HTTP | `websites[].sources` + `http` |
| S3-compatible object storage | `websites[].sources` + `s3` |
| Agent-pushed logs | `websites[].sources` + `agent` |

## logPath
The simplest setup:
```json
{
  "name": "Main Site",
  "logPath": "/var/log/nginx/access.log",
  "domains": ["example.com"],
  "logType": "nginx"
}
```

Glob support:
```json
{
  "name": "Main Site",
  "logPath": "/var/log/nginx/access-*.log",
  "domains": ["example.com"],
  "logType": "nginx"
}
```

`.gz` support:
```json
{
  "name": "Main Site",
  "logPath": "/var/log/nginx/access-*.log.gz",
  "domains": ["example.com"],
  "logType": "nginx"
}
```

Notes:
- In container deployments, `logPath` must be the path inside the container.
- If `sources` is configured, `logPath` is ignored.
- Site ID is derived from `websites[].name`; renaming creates a new site and reparses logs.

## Docker-Mounted Logs
Host log directory:
```text
/var/log/nginx/access.log
```

Docker mount:
```yaml
volumes:
  - /var/log/nginx:/share/logs/nginx:ro
```

NginxPulse config:
```json
{
  "name": "Main Site",
  "logPath": "/share/logs/nginx/access.log",
  "domains": ["example.com"],
  "logType": "nginx"
}
```

## sources
When logs are not convenient to mount locally or inside the container, use `sources` instead of `logPath`.

Common fields:
- `id` (string, required): unique source ID, recommended globally unique.
- `type` (string, required): `local` | `sftp` | `http` | `s3` | `agent`.
- `mode` (string): `poll` | `stream` | `hybrid`, default `poll`.
- `pollInterval` (string): reserved in the current version.
- `compression` (string): `auto` | `gz` | `none`, default `auto`.
- `parse` (object): per-source parsing override. Supports `logType`, `logFormat`, `logRegex`, and `timeLayout`.

## local source
Use this when one site needs multiple local files or patterns.

Either `path` or `pattern` is required:
```json
{
  "id": "local-main",
  "type": "local",
  "path": "/var/log/nginx/access.log",
  "pattern": "",
  "compression": "auto"
}
```

Glob example:
```json
{
  "id": "local-rotated",
  "type": "local",
  "pattern": "/var/log/nginx/access-*.log.gz",
  "compression": "auto"
}
```

## sftp source
Use this to pull logs from remote servers.

Key fields:
- `host` and `user` are required.
- `auth` supports `keyFile`, `passphrase`, or `password`.
- Either `path` or `pattern` is required.
- `auth.keyFile` must be an absolute path accessible on the machine or container running NginxPulse.

```json
{
  "id": "sftp-main",
  "type": "sftp",
  "host": "10.0.0.10",
  "port": 22,
  "user": "nginx",
  "auth": {
    "keyFile": "/home/nginxpulse/.ssh/nginxpulse_sftp",
    "passphrase": "",
    "password": ""
  },
  "path": "/var/log/nginx/access.log",
  "pattern": "",
  "compression": "auto"
}
```

## http source
Use this when log files are exposed over HTTP. Always use auth or IP allowlists to avoid leaking logs.

Single-file example:
```json
{
  "id": "http-main",
  "type": "http",
  "url": "https://logs.example.com/logs/access.log",
  "headers": { "Authorization": "Bearer TOKEN" },
  "rangePolicy": "auto",
  "compression": "auto"
}
```

Index-list example:
```json
{
  "id": "http-index",
  "type": "http",
  "url": "https://logs.example.com/logs/access.log",
  "index": {
    "url": "https://logs.example.com/logs/index.json",
    "method": "GET",
    "headers": { "Authorization": "Bearer TOKEN" },
    "jsonMap": {
      "items": "items",
      "path": "path",
      "size": "size",
      "mtime": "mtime",
      "etag": "etag",
      "compressed": "compressed"
    }
  }
}
```

Nginx directory exposure example:
```nginx
location /logs/ {
  alias /var/log/nginx/;
  autoindex on;
  # basic auth / IP allowlist recommended
}
```

## s3 source
Use this for AWS S3 or S3-compatible object storage.

Key fields:
- `bucket` is required.
- Empty `endpoint` means AWS default endpoint.
- `accessKey`/`secretKey` are optional depending on runtime credentials.

```json
{
  "id": "s3-main",
  "type": "s3",
  "endpoint": "https://s3.amazonaws.com",
  "region": "ap-northeast-1",
  "bucket": "my-bucket",
  "prefix": "nginx/",
  "pattern": "*.log.gz",
  "accessKey": "AKIA...",
  "secretKey": "SECRET...",
  "compression": "gz"
}
```

## agent source
Use this for logs pushed by NginxPulse Agent.

Notes:
- Agent here means the log collector process, not an AI LLM agent.
- `id` must match `sourceID` in the Agent config.
- This source is used to match parse overrides for pushed logs. It is not periodically scanned by the server.
- Agent installation, deployment, and `/api/ingest/logs` setup are documented in [Agent Collection](Agent-EN).

```json
{
  "id": "agent-main",
  "type": "agent"
}
```

## Complete sources Example
Place this directly inside one `websites[]` item:
```json
{
  "name": "Main Site",
  "domains": ["example.com", "www.example.com"],
  "logType": "nginx",
  "sources": [
    {
      "id": "sftp-main",
      "type": "sftp",
      "mode": "poll",
      "host": "192.168.6.131",
      "port": 22,
      "user": "root",
      "auth": {
        "keyFile": "/home/nginxpulse/.ssh/nginxpulse_sftp",
        "passphrase": "",
        "password": ""
      },
      "path": "/var/log/nginx/access.log",
      "pattern": "/var/log/nginx/access-*.log.gz",
      "pollInterval": "5s",
      "compression": "auto"
    }
  ]
}
```

## Per-Source Parse Override
If one site receives different log formats, configure `parse` on each source:
```json
{
  "id": "sftp-zoraxy",
  "type": "sftp",
  "host": "10.0.0.20",
  "user": "zoraxy",
  "auth": { "keyFile": "/home/nginxpulse/.ssh/id_ed25519" },
  "path": "/opt/zoraxy/log/zoraxy.log",
  "parse": {
    "logType": "zoraxy"
  }
}
```

## Related Docs
- [Configuration](Configuration-EN)
- [Supported Log Formats](Supported-Log-Formats-EN)
- [Agent Collection](Agent-EN)
- [Log Parsing](Log-Parsing-EN)
