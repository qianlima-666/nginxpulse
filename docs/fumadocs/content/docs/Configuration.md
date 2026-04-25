---
title: "配置说明"
---

# 配置说明

这页只讲“怎么把 NginxPulse 配起来”。完整字段说明、日志来源和日志格式示例已经拆到独立页面，避免配置入口过重。

## 配置文件位置
- 默认配置：`configs/nginxpulse_config.json`
- 本地开发：`scripts/dev_local.sh` 使用 `configs/nginxpulse_config.dev.json`
- 环境变量注入：`CONFIG_JSON` 或 `WEBSITES`

## 最小可用配置
只要日志文件和 PostgreSQL 可访问，通常从下面这个配置开始即可：

```json
{
  "websites": [
    {
      "name": "主站",
      "logPath": "/var/log/nginx/access.log",
      "domains": ["example.com", "www.example.com"],
      "logType": "nginx"
    }
  ],
  "database": {
    "driver": "postgres",
    "dsn": "postgres://nginxpulse:nginxpulse@127.0.0.1:5432/nginxpulse?sslmode=disable"
  },
  "server": {
    "Port": ":8089"
  }
}
```

## 复制后通常只需要改这些
- `websites[].name`：站点名称，会生成站点 ID；改名会被视为新站点。
- `websites[].logPath`：日志路径。远端、多来源或 Agent 采集见[日志来源配置](Log-Sources)。
- `websites[].domains`：站点域名列表，建议填写，用于判断站内来源。
- `websites[].logType`：日志类型。可选值和示例见[支持的日志格式](Supported-Log-Formats)。
- `websites[].autoDiscoverHosts`：开启后该项作为发现模板，按日志里的 `host` 字段自动生成真实站点。
- `database.dsn`：PostgreSQL 连接地址。
- `server.Port`：Web/API 监听端口，默认常用 `:8089`。

## 常见场景

### 本机或容器内 Nginx 日志
```json
{
  "name": "主站",
  "logPath": "/var/log/nginx/access.log",
  "domains": ["example.com", "www.example.com"],
  "logType": "nginx"
}
```

如果日志按天切割，可以用通配符：
```json
{
  "name": "主站",
  "logPath": "/var/log/nginx/access-*.log",
  "domains": ["example.com"],
  "logType": "nginx"
}
```

### Docker 挂载日志
容器内配置必须填写“容器内路径”，不是宿主机路径：
```yaml
volumes:
  - /var/log/nginx:/share/logs/nginx:ro
```

```json
{
  "name": "主站",
  "logPath": "/share/logs/nginx/access.log",
  "domains": ["example.com"],
  "logType": "nginx"
}
```

### 多站点
```json
{
  "websites": [
    {
      "name": "主站",
      "logPath": "/share/logs/nginx/main-access.log",
      "domains": ["example.com", "www.example.com"],
      "logType": "nginx"
    },
    {
      "name": "博客",
      "logPath": "/share/logs/nginx/blog-access.log",
      "domains": ["blog.example.com"],
      "logType": "nginx"
    }
  ]
}
```

### 按 Host 自动识别站点
大白话说：以前你要手动告诉 NginxPulse“有哪些站点”；现在可以让它先看日志里的域名，再自动生成站点。

以前如果一台 Nginx / Nginx Proxy Manager 上有很多域名，你通常要一个个写站点配置：
```json
{
  "name": "a.com",
  "logPath": "/share/logs/nginx/access.log",
  "domains": ["a.com"],
  "logFormat": "... $host"
},
{
  "name": "b.com",
  "logPath": "/share/logs/nginx/access.log",
  "domains": ["b.com"],
  "logFormat": "... $host"
}
```

如果后面又新增 `c.com`，还要再手动加一段配置。

开启 `autoDiscoverHosts` 后，只需要配置一个“发现模板”。系统会定时扫描匹配到的日志，从 `host` 字段提取域名，并按域名生成真实站点配置。比如日志里出现了 `a.com`、`b.com`、`c.com`，页面的站点下拉框就会出现这 3 个站点；每个站点只统计自己域名的日志。

前提是日志格式必须包含域名字段，例如 `$host` 或 `$http_host`。如果日志里只有 IP、时间、URL，没有域名，系统就无法判断这条访问属于哪个站点。

nginx-proxy-manager 可直接使用内置日志类型：
```json
{
  "name": "NPM Auto Discover",
  "logPath": "/share/logs/npm/proxy-host-*_access.log",
  "logType": "nginx-proxy-manager",
  "autoDiscoverHosts": true
}
```

普通 Nginx 或自定义日志也可以使用，只要 `logFormat` / `logRegex` 能解析出 `host`。例如：
```json
{
  "name": "Host Auto Discover",
  "logPath": "/share/logs/nginx/*.log",
  "logFormat": "$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\" $host",
  "autoDiscoverHosts": true
}
```

如果使用 `logRegex`，需要包含命名分组 `(?P<host>...)`。

### 远端日志或多来源采集
当日志不在 NginxPulse 所在机器/容器内，使用 `websites[].sources`：
```json
{
  "name": "主站",
  "domains": ["example.com"],
  "sources": [
    {
      "id": "sftp-main",
      "type": "sftp",
      "host": "10.0.0.10",
      "port": 22,
      "user": "nginx",
      "auth": { "keyFile": "/home/nginxpulse/.ssh/id_ed25519" },
      "path": "/var/log/nginx/access.log",
      "compression": "auto"
    }
  ],
  "logType": "nginx"
}
```

更多 `local`、`sftp`、`http`、`s3`、`agent` 示例见[日志来源配置](Log-Sources)。

### 自定义日志格式
如果内置 `logType` 不匹配，优先用 `logFormat`：
```json
{
  "name": "自定义站点",
  "logPath": "/var/log/custom/access.log",
  "logFormat": "$remote_addr [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\""
}
```

自定义格式字段和完整示例见[支持的日志格式](Supported-Log-Formats)。

### 反向代理二级路径
如果需要通过 `/nginxpulse/` 访问：
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

注意：`webBasePath` 只支持单段路径，修改后需要重启服务。

## 详细文档
- [完整字段参考](Config-Reference)
- [日志来源配置](Log-Sources)
- [支持的日志格式](Supported-Log-Formats)
- [Agent 采集](Agent)
- [日志解析机制](Log-Parsing)
- [IP 归属地解析](IP-Geo)
