# 日志来源配置

本文专门说明日志从哪里来：本地文件、通配符、压缩日志、远端拉取、对象存储和 Push Agent。

## 选择方式
| 场景 | 推荐配置 |
|---|---|
| 日志在本机或容器内 | `websites[].logPath` |
| 日志按天切割 | `logPath` 通配符 |
| `.gz` 压缩日志 | `logPath` 或 `sources[].compression` |
| 多台机器/远端目录 | `websites[].sources` + `sftp` |
| HTTP 暴露日志 | `websites[].sources` + `http` |
| S3/兼容对象存储 | `websites[].sources` + `s3` |
| Agent 主动推送 | `websites[].sources` + `agent` |

## logPath
最简单的配置方式：
```json
{
  "name": "主站",
  "logPath": "/var/log/nginx/access.log",
  "domains": ["example.com"],
  "logType": "nginx"
}
```

支持通配符：
```json
{
  "name": "主站",
  "logPath": "/var/log/nginx/access-*.log",
  "domains": ["example.com"],
  "logType": "nginx"
}
```

支持 `.gz`：
```json
{
  "name": "主站",
  "logPath": "/var/log/nginx/access-*.log.gz",
  "domains": ["example.com"],
  "logType": "nginx"
}
```

注意：
- 容器部署时，`logPath` 必须是容器内路径。
- 如果配置了 `sources`，`logPath` 会被忽略。
- 站点 ID 由 `websites[].name` 生成，改名会产生新站点并重新解析。

## Docker 挂载日志
宿主机日志目录：
```text
/var/log/nginx/access.log
```

Docker 挂载：
```yaml
volumes:
  - /var/log/nginx:/share/logs/nginx:ro
```

NginxPulse 配置：
```json
{
  "name": "主站",
  "logPath": "/share/logs/nginx/access.log",
  "domains": ["example.com"],
  "logType": "nginx"
}
```

## sources
当日志不方便挂载到本机或容器时，可在站点配置中使用 `sources` 替代 `logPath`。

通用字段：
- `id` (string, 必填): 来源唯一 ID，建议全站唯一。
- `type` (string, 必填): `local` | `sftp` | `http` | `s3` | `agent`。
- `mode` (string): `poll` | `stream` | `hybrid`，默认 `poll`。
- `pollInterval` (string): 轮询间隔，当前版本为预留字段。
- `compression` (string): `auto` | `gz` | `none`，默认 `auto`。
- `parse` (object): 当前 source 的解析覆盖，支持 `logType`、`logFormat`、`logRegex`、`timeLayout`。

## local source
适合在同一个站点中配置多个本地文件或目录规则。

字段要点：`path` 或 `pattern` 二选一。
```json
{
  "id": "local-main",
  "type": "local",
  "path": "/var/log/nginx/access.log",
  "pattern": "",
  "compression": "auto"
}
```

通配符示例：
```json
{
  "id": "local-rotated",
  "type": "local",
  "pattern": "/var/log/nginx/access-*.log.gz",
  "compression": "auto"
}
```

## sftp source
适合从远端服务器拉取日志。

字段要点：
- `host`、`user` 必填。
- `auth` 支持 `keyFile`、`passphrase` 或 `password`。
- `path` 或 `pattern` 二选一。
- `auth.keyFile` 必须是运行 NginxPulse 的机器或容器内可访问的绝对路径。

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
适合你能在内网通过 HTTP 暴露日志文件的场景。务必加认证或 IP 白名单，避免日志泄露。

单文件示例：
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

索引列表示例：
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

Nginx 暴露日志目录示例：
```nginx
location /logs/ {
  alias /var/log/nginx/;
  autoindex on;
  # 建议加 basic auth / IP 白名单
}
```

## s3 source
适合 AWS S3 或兼容 S3 的对象存储。

字段要点：
- `bucket` 必填。
- `endpoint` 为空表示使用 AWS 默认 endpoint。
- `accessKey`/`secretKey` 可选，取决于运行环境凭证。

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
用于接入 NginxPulse Agent 主动推送的日志。

说明：
- 这里的 Agent 是日志采集进程，不是 AI 大模型 Agent。
- `id` 需要和 Agent 配置中的 `sourceID` 一致。
- 该 source 用于给 Agent 推送的数据匹配解析覆盖规则，不参与服务端定期扫描。
- Agent 安装、部署和 `/api/ingest/logs` 推送配置见 [Agent 采集](Agent)。

```json
{
  "id": "agent-main",
  "type": "agent"
}
```

## 完整 sources 示例
可直接放到 `websites[]` 项中使用：
```json
{
  "name": "主站",
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

## 每个 source 单独覆盖解析规则
如果同一个站点接入不同格式的日志，可以在 source 上配置 `parse`：
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

## 相关文档
- [配置说明](Configuration)
- [支持的日志格式](Supported-Log-Formats)
- [Agent 采集](Agent)
- [日志解析机制](Log-Parsing)
