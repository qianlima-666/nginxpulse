# Agent 采集

Push Agent 用于日志不在 NginxPulse 主机上的场景，例如独立日志服务器、边缘节点、Kubernetes 节点或不方便挂载日志目录的机器。

这里的 Agent 指日志采集进程，不是 AI 大模型 Agent。它会读取本机日志文件新增内容，并主动推送到 NginxPulse 的 `POST /api/ingest/logs` 接口。

## Agent 是哪个包

Agent 源码入口：
```text
cmd/nginxpulse-agent
```

预构建二进制：
```text
prebuilt/nginxpulse-agent-linux-amd64
prebuilt/nginxpulse-agent-darwin-amd64
prebuilt/nginxpulse-agent-darwin-arm64
prebuilt/nginxpulse-agent-windows-amd64.exe
```

Docker 构建文件：
```text
Dockerfile.agent
```

本地构建二进制：
```bash
go build -trimpath -ldflags="-s -w" -o bin/nginxpulse-agent ./cmd/nginxpulse-agent
```

交叉编译 Windows amd64：
```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/nginxpulse-agent-windows-amd64.exe ./cmd/nginxpulse-agent
```

交叉编译 macOS：
```bash
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/nginxpulse-agent-darwin-arm64 ./cmd/nginxpulse-agent
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/nginxpulse-agent-darwin-amd64 ./cmd/nginxpulse-agent
```

本地构建镜像：
```bash
docker build -f Dockerfile.agent -t nginxpulse-agent:local .
```

## 平台支持

| 平台 | 支持状态 | 说明 |
|---|---|---|
| Linux | 推荐 | 适合服务器、systemd、Docker、Kubernetes。 |
| macOS | 可用 | 提供 arm64 和 amd64 预编译包；适合开发调试、边缘节点或轻量采集。 |
| Windows | 可用 | 提供 amd64 预编译包；长期后台运行建议配合 NSSM、WinSW 或任务计划程序。 |

Agent 代码不依赖 Linux-only 系统调用。macOS 上主要注意执行权限、Gatekeeper quarantine、日志文件读权限；Windows 上主要注意日志路径、文件权限和防火墙。

## 工作方式

Agent 会按 `routes[].paths` 读取日志文件新增行，并按批次推送到：
```text
POST /api/ingest/logs
```

请求会带上：
- `website_id`: 目标站点 ID。
- `source_id`: 来源 ID，用于匹配 `websites[].sources[].id` 上的解析覆盖配置。
- `lines`: 新增日志行。

首次读取大文件时，Agent 默认只读取文件尾部最近 `8MiB`，避免初次启动把历史日志全量灌入服务端和 PostgreSQL。如果你确实需要从文件头完整回放，设置：
```json
{
  "initialTailBytes": -1
}
```

## 主服务配置

运行 NginxPulse 的机器需要满足：
- Agent 能访问主服务 HTTP 地址，例如 `http://10.0.0.5:8089`。
- 如果启用了访问密钥，Agent 配置中的 `accessKey` 需要匹配 `system.accessKeys`。
- 目标站点已经存在，并能查到 `websiteID`。

查询站点 ID：
```bash
curl -H "X-NginxPulse-Key: your-key" http://10.0.0.5:8089/api/websites
```

建议在站点中配置 `type=agent` 的 source，并让 `id` 与 Agent 的 `sourceID` 保持一致：
```json
{
  "name": "主站",
  "domains": ["example.com", "www.example.com"],
  "sources": [
    {
      "id": "agent-main",
      "type": "agent",
      "parse": {
        "logFormat": "$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\" $host"
      }
    }
  ]
}
```

`type=agent` 的 source 只用于标识 `sourceID` 和配置解析覆盖规则，不会被服务端定期扫描。

## Agent 配置

在日志所在机器上创建配置文件，例如 `/etc/nginxpulse/agent.json`：
```json
{
  "server": "http://10.0.0.5:8089",
  "accessKey": "your-key",
  "routes": [
    {
      "websiteID": "abcd",
      "sourceID": "agent-main",
      "paths": ["/var/log/nginx/main-access.log"]
    },
    {
      "websiteID": "ef01",
      "sourceID": "agent-blog",
      "paths": ["/var/log/nginx/blog-access.log"]
    }
  ],
  "pollInterval": "1s",
  "batchSize": 200,
  "flushInterval": "2s"
}
```

Windows 路径需要在 JSON 中转义反斜杠，例如：
```json
{
  "server": "http://10.0.0.5:8089",
  "accessKey": "your-key",
  "routes": [
    {
      "websiteID": "abcd",
      "sourceID": "agent-main",
      "paths": ["C:\\nginx\\logs\\access.log"]
    }
  ]
}
```

常用字段：
- `server`: NginxPulse 主服务地址。
- `accessKey`: 访问密钥；主服务未启用密钥时可留空。
- `routes`: 多站点或多来源采集规则。
- `routes[].websiteID`: 目标站点 ID。
- `routes[].sourceID`: 来源 ID，建议与 `websites[].sources[].id` 一致。
- `routes[].paths`: 需要采集的本机日志路径列表。
- `pollInterval`: 读取文件新增内容的间隔，默认 `1s`。
- `batchSize`: 单次推送最大行数，默认 `200`。
- `flushInterval`: 定时推送间隔，默认 `2s`。
- `initialTailBytes`: 首次读取大文件时读取尾部字节数；`0` 表示默认 `8MiB`，负数表示从文件头开始。
- `initialMaxLines`: 首次读取最多行数；`0` 表示不额外限制。
- `maxPendingLines`: 内存中待发送行数上限，默认 `5000`。
- `maxLineBytes`: 单行最大字节数，默认 `256KiB`。
- `requestTimeout`: HTTP 推送超时，默认 `90s`。
- `retryBackoffMin`: 推送失败后的最小退避，默认 `1s`。
- `retryBackoffMax`: 推送失败后的最大退避，默认 `30s`。
- `exitOnMaxBackoff`: 达到最大退避后再次失败是否退出进程，默认 `false`。

兼容旧版单站点配置：
```json
{
  "server": "http://10.0.0.5:8089",
  "accessKey": "your-key",
  "websiteID": "abcd",
  "sourceID": "agent-main",
  "paths": ["/var/log/nginx/access.log"]
}
```

如果同时配置了 `routes` 和顶层 `websiteID/sourceID/paths`，Agent 会使用 `routes`，并忽略顶层字段。

## 运行方式

直接运行：
```bash
nginxpulse-agent -config /etc/nginxpulse/agent.json
```

Windows 直接运行：
```powershell
.\nginxpulse-agent-windows-amd64.exe -config .\agent.json
```

macOS 直接运行：
```bash
# Apple Silicon 使用 darwin-arm64，Intel Mac 使用 darwin-amd64
sudo install -m 755 prebuilt/nginxpulse-agent-darwin-arm64 /usr/local/bin/nginxpulse-agent
sudo xattr -d com.apple.quarantine /usr/local/bin/nginxpulse-agent 2>/dev/null || true
sudo /usr/local/bin/nginxpulse-agent -config /etc/nginxpulse/agent.json
```

macOS launchd 示例：
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.nginxpulse.agent</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/nginxpulse-agent</string>
    <string>-config</string>
    <string>/etc/nginxpulse/agent.json</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>/var/log/nginxpulse/agent.log</string>
  <key>StandardErrorPath</key>
  <string>/var/log/nginxpulse/agent.err.log</string>
</dict>
</plist>
```

保存为 `/Library/LaunchDaemons/com.nginxpulse.agent.plist` 后启动：
```bash
sudo mkdir -p /etc/nginxpulse /var/log/nginxpulse
sudo cp agent.json /etc/nginxpulse/agent.json
sudo chown root:wheel /Library/LaunchDaemons/com.nginxpulse.agent.plist
sudo chmod 644 /Library/LaunchDaemons/com.nginxpulse.agent.plist
sudo launchctl bootstrap system /Library/LaunchDaemons/com.nginxpulse.agent.plist
sudo launchctl enable system/com.nginxpulse.agent
sudo launchctl kickstart -k system/com.nginxpulse.agent
sudo launchctl print system/com.nginxpulse.agent
```

systemd 示例：
```ini
[Unit]
Description=NginxPulse Agent
After=network-online.target

[Service]
ExecStart=/usr/local/bin/nginxpulse-agent -config /etc/nginxpulse/agent.json
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
```

Docker 示例：
```bash
docker run -d --name nginxpulse-agent \
  -v /etc/nginxpulse/agent.json:/etc/nginxpulse/agent.json:ro \
  -v /var/log/nginx:/var/log/nginx:ro \
  nginxpulse-agent:local \
  -config /etc/nginxpulse/agent.json
```

## 环境变量覆盖

配置文件为主，以下环境变量可覆盖可选项，便于 Kubernetes 或 DaemonSet 管理：

| 环境变量 | 覆盖字段 |
|---|---|
| `NGINXPULSE_AGENT_POLL_INTERVAL` | `pollInterval` |
| `NGINXPULSE_AGENT_FLUSH_INTERVAL` | `flushInterval` |
| `NGINXPULSE_AGENT_INITIAL_TAIL_BYTES` | `initialTailBytes` |
| `NGINXPULSE_AGENT_INITIAL_MAX_LINES` | `initialMaxLines` |
| `NGINXPULSE_AGENT_BATCH_SIZE` | `batchSize` |
| `NGINXPULSE_AGENT_REQUEST_TIMEOUT` | `requestTimeout` |
| `NGINXPULSE_AGENT_MAX_PENDING_LINES` | `maxPendingLines` |
| `NGINXPULSE_AGENT_MAX_LINE_BYTES` | `maxLineBytes` |
| `NGINXPULSE_AGENT_RETRY_BACKOFF_MIN` | `retryBackoffMin` |
| `NGINXPULSE_AGENT_RETRY_BACKOFF_MAX` | `retryBackoffMax` |
| `NGINXPULSE_AGENT_EXIT_ON_MAX_BACKOFF` | `exitOnMaxBackoff` |

这些环境变量不会覆盖 `server`、`websiteID`、`sourceID` 和 `paths`，避免误配置导致 Agent 推送到错误目标。

## 验证与排障

先确认主服务可访问：
```bash
curl http://10.0.0.5:8089/health
```

再确认密钥和站点列表：
```bash
curl -H "X-NginxPulse-Key: your-key" http://10.0.0.5:8089/api/websites
```

Agent 正常启动时会出现类似日志：
```text
nginxpulse-agent: config loaded
read new lines
push succeeded
```

常见问题：
- `401 Unauthorized`: `accessKey` 不匹配，或主服务启用了密钥但 Agent 未配置。
- `400 站点不存在`: `websiteID` 不存在，先通过 `/api/websites` 查询真实 ID。
- `push succeeded` 但页面没有数据: 检查 `sourceID` 是否匹配、解析格式是否正确、站点域名是否覆盖日志里的 host。
- 没有 `read new lines`: 检查日志路径是否存在、容器挂载路径是否正确、Agent 运行用户是否有读权限。
- 重启后没有读取历史日志: 默认只读取尾部最近 `8MiB`；需要全量回放时设置 `initialTailBytes: -1`。

## 相关文档

- [配置说明](Configuration)
- [日志来源配置](Log-Sources)
- [支持的日志格式](Supported-Log-Formats)
- [日志解析机制](Log-Parsing)
