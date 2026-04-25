---
title: "Agent Collection"
---

# Agent Collection

Push Agent is for cases where logs are not on the NginxPulse host, such as a separate log server, edge node, Kubernetes node, or any machine where mounting log directories is inconvenient.

Agent here means the log collector process, not an AI LLM agent. It reads appended lines from local log files and pushes them to NginxPulse through `POST /api/ingest/logs`.

## Which Package Is The Agent

Agent source entry:
```text
cmd/nginxpulse-agent
```

Prebuilt binaries:
```text
prebuilt/nginxpulse-agent-linux-amd64
prebuilt/nginxpulse-agent-darwin-amd64
prebuilt/nginxpulse-agent-darwin-arm64
prebuilt/nginxpulse-agent-windows-amd64.exe
```

Docker build file:
```text
Dockerfile.agent
```

Build a local binary:
```bash
go build -trimpath -ldflags="-s -w" -o bin/nginxpulse-agent ./cmd/nginxpulse-agent
```

Cross-compile Windows amd64:
```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/nginxpulse-agent-windows-amd64.exe ./cmd/nginxpulse-agent
```

Cross-compile macOS:
```bash
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/nginxpulse-agent-darwin-arm64 ./cmd/nginxpulse-agent
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/nginxpulse-agent-darwin-amd64 ./cmd/nginxpulse-agent
```

Build a local image:
```bash
docker build -f Dockerfile.agent -t nginxpulse-agent:local .
```

## Platform Support

| Platform | Status | Notes |
|---|---|---|
| Linux | Recommended | Best for servers, systemd, Docker, and Kubernetes. |
| macOS | Supported | arm64 and amd64 prebuilt binaries are provided; useful for development, edge nodes, or lightweight collection. |
| Windows | Supported | amd64 prebuilt binary is provided; for long-running background use, pair it with NSSM, WinSW, or Task Scheduler. |

Agent does not rely on Linux-only syscalls. On macOS, pay attention to executable permission, Gatekeeper quarantine, and log file read permissions. On Windows, pay attention to log paths, file permissions, and firewall rules.

## How It Works

Agent reads newly appended lines from `routes[].paths` and pushes them in batches to:
```text
POST /api/ingest/logs
```

The payload includes:
- `website_id`: target website ID.
- `source_id`: source ID used to match parse overrides on `websites[].sources[].id`.
- `lines`: appended log lines.

On the first read of a large file, Agent reads only the latest `8MiB` by default. This prevents the first startup from flooding the server and PostgreSQL with historical logs. If you need a full replay from the beginning of the file, set:
```json
{
  "initialTailBytes": -1
}
```

## Server Configuration

The machine running NginxPulse must meet these requirements:
- Agent can reach the server HTTP address, for example `http://10.0.0.5:8089`.
- If access keys are enabled, Agent `accessKey` must match `system.accessKeys`.
- The target website already exists and its `websiteID` is known.

Query website IDs:
```bash
curl -H "X-NginxPulse-Key: your-key" http://10.0.0.5:8089/api/websites
```

It is recommended to configure a `type=agent` source on the website, with `id` matching the Agent `sourceID`:
```json
{
  "name": "Main Site",
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

A `type=agent` source only identifies `sourceID` and configures parse overrides. It is not periodically scanned by the server.

## Agent Configuration

Create a config file on the log machine, for example `/etc/nginxpulse/agent.json`:
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

Windows paths must escape backslashes in JSON, for example:
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

Common fields:
- `server`: NginxPulse server URL.
- `accessKey`: access key; leave empty if the server does not enforce keys.
- `routes`: multi-site or multi-source collection rules.
- `routes[].websiteID`: target website ID.
- `routes[].sourceID`: source ID, recommended to match `websites[].sources[].id`.
- `routes[].paths`: local log paths to collect.
- `pollInterval`: interval for reading appended lines, default `1s`.
- `batchSize`: max lines per push, default `200`.
- `flushInterval`: periodic flush interval, default `2s`.
- `initialTailBytes`: tail bytes to read on first read of a large file; `0` means default `8MiB`, negative means read from the beginning.
- `initialMaxLines`: max lines on first read; `0` means no extra limit.
- `maxPendingLines`: max buffered lines in memory, default `5000`.
- `maxLineBytes`: max bytes per line, default `256KiB`.
- `requestTimeout`: HTTP push timeout, default `90s`.
- `retryBackoffMin`: minimum retry backoff after push failure, default `1s`.
- `retryBackoffMax`: maximum retry backoff after push failure, default `30s`.
- `exitOnMaxBackoff`: whether to exit after failing again at max backoff, default `false`.

Legacy single-site config is still supported:
```json
{
  "server": "http://10.0.0.5:8089",
  "accessKey": "your-key",
  "websiteID": "abcd",
  "sourceID": "agent-main",
  "paths": ["/var/log/nginx/access.log"]
}
```

If both `routes` and top-level `websiteID/sourceID/paths` are configured, Agent uses `routes` and ignores the top-level fields.

## Running Agent

Run directly:
```bash
nginxpulse-agent -config /etc/nginxpulse/agent.json
```

Run directly on Windows:
```powershell
.\nginxpulse-agent-windows-amd64.exe -config .\agent.json
```

Run directly on macOS:
```bash
# Use darwin-arm64 for Apple Silicon, darwin-amd64 for Intel Mac.
sudo install -m 755 prebuilt/nginxpulse-agent-darwin-arm64 /usr/local/bin/nginxpulse-agent
sudo xattr -d com.apple.quarantine /usr/local/bin/nginxpulse-agent 2>/dev/null || true
sudo /usr/local/bin/nginxpulse-agent -config /etc/nginxpulse/agent.json
```

macOS launchd example:
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

Save it as `/Library/LaunchDaemons/com.nginxpulse.agent.plist`, then start it:
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

systemd example:
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

Docker example:
```bash
docker run -d --name nginxpulse-agent \
  -v /etc/nginxpulse/agent.json:/etc/nginxpulse/agent.json:ro \
  -v /var/log/nginx:/var/log/nginx:ro \
  nginxpulse-agent:local \
  -config /etc/nginxpulse/agent.json
```

## Environment Overrides

The config file is the source of truth. These environment variables can override optional fields, which is useful for Kubernetes or DaemonSet deployments:

| Environment variable | Field |
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

These environment variables do not override `server`, `websiteID`, `sourceID`, or `paths`, to avoid accidentally pushing logs to the wrong target.

## Verification And Troubleshooting

First confirm that the server is reachable:
```bash
curl http://10.0.0.5:8089/health
```

Then confirm the access key and website list:
```bash
curl -H "X-NginxPulse-Key: your-key" http://10.0.0.5:8089/api/websites
```

Normal Agent startup logs include messages like:
```text
nginxpulse-agent: config loaded
read new lines
push succeeded
```

Common issues:
- `401 Unauthorized`: `accessKey` does not match, or the server requires a key but Agent does not configure one.
- `400 站点不存在`: `websiteID` does not exist. Query the real ID through `/api/websites`.
- `push succeeded` but no data appears: check `sourceID`, parse format, and whether website domains match the host in log lines.
- No `read new lines`: check the path, container mount, and read permissions of the Agent process.
- Restart does not read historical logs: Agent reads only the latest `8MiB` by default; set `initialTailBytes: -1` for full replay.

## Related Docs

- [Configuration](Configuration-EN)
- [Log Sources](Log-Sources-EN)
- [Supported Log Formats](Supported-Log-Formats-EN)
- [Log Parsing](Log-Parsing-EN)
