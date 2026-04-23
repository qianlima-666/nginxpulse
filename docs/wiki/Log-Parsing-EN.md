# Log Parsing

## Flow
1. Initial scan: parse recent window after startup.
2. Incremental scan: periodic scan by `system.taskInterval`.
3. Backfill: fill older logs in background.
4. IP geo backfill: resolve IP locations asynchronously.

## Incremental scan & state
- State file: `var/nginxpulse_data/nginx_scan_state.json`
- If current size < last size, the file is treated as rotated and re-parsed.
- Site ID is derived from `websites[].name`. Renaming creates a new site.

## Batch size
- `system.parseBatchSize` controls batch size (default 100).
- Can be overridden by `LOG_PARSE_BATCH_SIZE`.

## Progress & ETA
Endpoint: `GET /api/status`
- `log_parsing_progress`
- `log_parsing_estimated_remaining_seconds`
- `ip_geo_progress`
- `ip_geo_estimated_remaining_seconds`

Poll this endpoint to update progress in UI.

## 10G+ log optimization
- Parsing writes core fields first; IP geo is queued.
- IP geo is resolved in batches after parsing.
- For speed: increase `parseBatchSize`, use faster disk, or split logs by day.

## IIS default rule (W3C Extended)
NginxPulse now supports `logType=iis` (alias: `iis-w3c`). The built-in parser follows the common IIS W3C default field order:

`date time s-ip cs-method cs-uri-stem cs-uri-query s-port cs-username c-ip cs(User-Agent) cs(Referer) sc-status sc-substatus sc-win32-status time-taken`

Notes:
- Metadata lines starting with `#` (such as `#Software`, `#Version`, `#Fields`) are skipped automatically.
- URL is built from `cs-uri-stem`; when `cs-uri-query` is not `-`, it is appended as `path?query`.
- IIS W3C timestamps are typically UTC, and the default time layout is `2006-01-02 15:04:05`.

Config example:
```json
{
  "name": "iis-site",
  "logPath": "/var/log/iis/u_ex*.log",
  "logType": "iis"
}
```

Sample line:
```text
2026-02-08 10:05:34 10.0.0.10 GET /index.html a=1&b=2 443 - 203.0.113.8 Mozilla/5.0+(Windows+NT+10.0;+Win64;+x64) https://example.com/ 200 0 0 36
```

## Retention
- `system.logRetentionDays` controls cleanup.
- Cleanup runs at 02:00 (system timezone).

## Mounting Multiple Log Files
`WEBSITES` is a **JSON array**, each item describes one site. `logPath` must be a **container-accessible path**.

Example:
```yaml
environment:
  WEBSITES: '[{"name":"Site 1","logPath":"/share/logs/nginx/access-site1.log","domains":["www.kaisir.cn","kaisir.cn"]}, {"name":"Site 2","logPath":"/share/logs/nginx/access-site2.log","domains":["home.kaisir.cn"]}]'
volumes:
  - ./nginx_data/logs/site1/access.log:/share/logs/nginx/access-site1.log:ro
  - ./nginx_data/logs/site2/access.log:/share/logs/nginx/access-site2.log:ro
```

If you have many sites, consider **mounting the entire log directory** and specify exact files in `WEBSITES`:
```yaml
environment:
  WEBSITES: '[{"name":"Site 1","logPath":"/share/logs/nginx/access-site1.log","domains":["www.kaisir.cn","kaisir.cn"]}, {"name":"Site 2","logPath":"/share/logs/nginx/access-site2.log","domains":["home.kaisir.cn"]}]'
volumes:
  - ./nginx_data/logs:/share/logs/nginx/
```

> Tip: If logs are rotated daily, use `*` to replace the date, e.g. `{"logPath":"/share/logs/nginx/site1.top-*.log"}`.

#### Compressed logs (.gz)
`.gz` logs are supported. `logPath` can point to a single `.gz` file or a glob:
```json
{"logPath": "/share/logs/nginx/access-*.log.gz"}
```
There is a gzip sample in `var/log/gz-log-read-test/`.

## Remote Log Sources (sources)
When logs are not convenient to mount locally, you can use `sources` instead of `logPath`. Once `sources` is set, `logPath` is ignored.

`sources` is a **JSON array**. Each item defines a log source. This design allows:
1) Multiple sources per site (multiple machines/directories/buckets).
2) Different parsing/auth/polling strategies per source.
3) Easy extension for rotation/archival without changing old sources.

Common fields:
- `id`: unique source ID (recommend globally unique).
- `type`: `local` / `sftp` / `http` / `s3` / `agent`.
- `mode`:
  - `poll`: periodic pulling (default).
  - `stream`: streaming input only (currently Push Agent only).
  - `hybrid`: stream + polling fallback (only Push Agent streams; others still use `poll`).
- `pollInterval`: polling interval (e.g. `5s`).
- `pattern`: rotation glob (SFTP/Local/S3 use glob; HTTP uses index JSON).
- `compression`: `auto` / `gz` / `none`.
- `parse`: override parsing (see “Parsing Override”).
> `stream` mode is mainly for Push Agent; other sources still run as `poll`.

### Option 1: HTTP Exposed Logs
Best when you can provide HTTP access to log files (internal network or with auth).

Method A: Expose files via Nginx/Apache (lock it down to avoid leakage)
```nginx
location /logs/ {
  alias /var/log/nginx/;
  autoindex on;
  # Add basic auth / IP allowlist
}
```

Then configure `sources`:
```json
{
  "id": "http-main",
  "type": "http",
  "mode": "poll",
  "url": "https://logs.example.com/logs/access.log",
  "rangePolicy": "auto",
  "pollInterval": "10s"
}
```

`rangePolicy`:
- `auto`: prefer Range; fallback to full download (skips already-read bytes).
- `range`: force Range; error if not supported.
- `full`: always download full file.

Method B: JSON index API  
Good for rotated logs (daily/hourly) or `.gz` archives:
```json
{
  "index": {
    "url": "https://logs.example.com/index.json",
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

Recommended index contract:
1) Return a JSON with an array of log objects.
2) Each item must include `path` (a fetchable URL).
3) Provide `size` / `mtime` / `etag` to detect changes and avoid duplicates.
4) `mtime` supports RFC3339 / RFC3339Nano / `2006-01-02 15:04:05` / Unix seconds.

Example response:
```json
{
  "items": [
    {
      "path": "https://logs.example.com/access-2024-11-03.log.gz",
      "size": 123456,
      "mtime": "2024-11-03T13:00:00Z",
      "etag": "abc123",
      "compressed": true
    },
    {
      "path": "https://logs.example.com/access.log",
      "size": 98765,
      "mtime": 1730638800,
      "etag": "def456",
      "compressed": false
    }
  ]
}
```

If your fields differ, map them in `jsonMap`:
```json
{
  "index": {
    "url": "https://logs.example.com/index.json",
    "jsonMap": {
      "items": "data",
      "path": "url",
      "size": "length",
      "mtime": "updated_at",
      "etag": "hash",
      "compressed": "gz"
    }
  }
}
```

Notes:
- `path` must be a directly accessible log URL.
- For `.gz`, provide stable `etag` / `size` / `mtime` to avoid duplicate parsing.
- If HTTP Range is not supported, use `auto` or `full`.

### Option 2: SFTP Pull
Ideal when SSH/SFTP access is available, no extra HTTP service needed.
```json
{
  "id": "sftp-main",
  "type": "sftp",
  "mode": "poll",
  "host": "1.2.3.4",
  "port": 22,
  "user": "nginx",
  "auth": { "keyFile": "/secrets/id_rsa", "passphrase": "", "password": "" },
  "path": "/var/log/nginx/access.log",
  "pattern": "/var/log/nginx/access-*.log.gz",
  "pollInterval": "5s"
}
```
> `auth` supports `keyFile`, `passphrase` (private key passphrase), and `password`.

#### SFTP key-based login walkthrough (local -> remote)
1) Generate a dedicated key pair on your local machine (recommended: `ed25519`):
```bash
ssh-keygen -t ed25519 -a 100 -f ~/.ssh/nginxpulse_sftp -C "nginxpulse-sftp"
```

2) Install the public key on the remote user:
```bash
ssh-copy-id -i ~/.ssh/nginxpulse_sftp.pub <user>@<host>
```
If `ssh-copy-id` is unavailable:
```bash
cat ~/.ssh/nginxpulse_sftp.pub | ssh <user>@<host> \
'mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys'
```

3) Ensure remote permissions are correct:
```bash
chmod 700 ~/.ssh
chmod 600 ~/.ssh/authorized_keys
```

4) Verify SSH key login from local (public key only):
```bash
ssh -i ~/.ssh/nginxpulse_sftp -o PreferredAuthentications=publickey <user>@<host>
```

5) Verify SFTP key login:
```bash
sftp -i ~/.ssh/nginxpulse_sftp <user>@<host>
```

6) After verification, configure `sources`:
```json
{
  "id": "sftp-main",
  "type": "sftp",
  "host": "<host>",
  "port": 22,
  "user": "<user>",
  "auth": {
    "keyFile": "/absolute/path/to/nginxpulse_sftp",
    "passphrase": ""
  },
  "path": "/var/log/nginx/access.log"
}
```
> `keyFile` must be an absolute path accessible on the machine (or container) running NginxPulse.

7) If login still fails, use verbose SSH output first:
```bash
ssh -vvv -i ~/.ssh/nginxpulse_sftp -o PreferredAuthentications=publickey <user>@<host>
```
On Alpine, a common SSH log check is:
```bash
grep sshd /var/log/messages | tail -n 80
```

### Option 3: Object Storage (S3/OSS)
Best when logs are archived to OSS/S3 (Aliyun/Tencent/AWS compatible endpoints).
```json
{
  "id": "s3-main",
  "type": "s3",
  "mode": "poll",
  "endpoint": "https://oss-cn-hangzhou.aliyuncs.com",
  "bucket": "nginx-logs",
  "prefix": "prod/access/",
  "pollInterval": "30s"
}
```

### Parsing Override (sources[].parse)
If formats differ across sources, override parsing per source:
```json
{
  "parse": {
    "logType": "nginx",
    "logRegex": "^(?P<ip>\\S+) - (?P<user>\\S+) \\[(?P<time>[^\\]]+)\\] \"(?P<request>[^\"]+)\" (?P<status>\\d+) (?P<bytes>\\d+) \"(?P<referer>[^\"]*)\" \"(?P<ua>[^\"]*)\"$",
    "timeLayout": "02/Jan/2006:15:04:05 -0700"
  }
}
```

### Push Agent (Realtime)
Use the Push Agent when logs are on another machine, edge node, Kubernetes node, or any environment where mounting the log directory into the NginxPulse server is inconvenient. The agent is a standalone collector process: it reads local text log files on the log server and pushes new lines to the NginxPulse server through `POST /api/ingest/logs`.

> Terminology: Agent here means the log collection process, not an AI/LLM agent.

#### Which package is the Agent?
The repository provides three usable entry points:

- Source entry: `cmd/nginxpulse-agent`
- Prebuilt binaries:
  - `prebuilt/nginxpulse-agent-linux-amd64`
  - `prebuilt/nginxpulse-agent-darwin-arm64`
- Container build file: `Dockerfile.agent`

Build a binary:
```bash
go build -trimpath -ldflags="-s -w" -o bin/nginxpulse-agent ./cmd/nginxpulse-agent
```

Build a container image:
```bash
docker build -f Dockerfile.agent -t nginxpulse-agent:local .
```

#### How it works
The agent does not connect to PostgreSQL and does not read the NginxPulse data directory. It only does three things:

1. Reads local files configured in `routes[].paths` on the log server.
2. Keeps in-process file offsets and continuously reads appended lines.
3. Batches lines to the server endpoint `/api/ingest/logs`; the server handles site matching, parsing, deduplication, and database writes.

On first read of a large file, the agent defaults to tailing only the latest `8MiB` to avoid flooding the server and PostgreSQL with historical logs during initial deployment. Set `initialTailBytes` to `-1` if you intentionally want a full replay from the beginning of the file.

#### Server setup (machine running NginxPulse)
1) Make sure the server HTTP address is reachable from the log server, for example `http://10.0.0.5:8089`.
2) Access keys are recommended: set `ACCESS_KEYS` or `system.accessKeys`.
3) Get `websiteID`:
```bash
curl -H "X-NginxPulse-Key: your-key" http://10.0.0.5:8089/api/websites
```
The `id` field in the response is the `websiteID` used by the agent config.

4) If the site default parser already matches your logs, you do not need to configure `sources`. If you want a parser dedicated to agent-pushed logs, add a `type=agent` source and make its `id` equal the agent `sourceID`:
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

The `type=agent` source only identifies the pushed `sourceID` and provides parse overrides. The NginxPulse server does not periodically scan this source.

#### Agent config (log server)
Create `/etc/nginxpulse/agent.json`:
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

Field notes:

- `server`: NginxPulse server address. Do not include `/api/ingest/logs`; the agent appends it automatically.
- `accessKey`: matches server-side `ACCESS_KEYS` / `system.accessKeys`; leave empty only when access keys are disabled.
- `routes`: multi-site and multi-path routing. Each route maps local files to one site and one `sourceID`.
- `routes[].websiteID`: site ID returned by the server endpoint `/api/websites`.
- `routes[].sourceID`: pushed source ID. If the site has a `type=agent` source, this value must match that source `id`.
- `routes[].paths`: local text log files on the log server. The current agent reads plain text logs and skips `.gz` files.
- `pollInterval`: interval for reading appended lines, default `1s`.
- `batchSize`: push immediately when pending lines reach this count, default `200`.
- `flushInterval`: push pending lines at this interval even if `batchSize` has not been reached, default `2s`.
- `initialTailBytes`: on first read, start from the last N bytes; `0` uses the default `8MiB`, `-1` starts from the beginning.
- `initialMaxLines`: max lines during first read; `0` means no extra limit.
- `maxPendingLines`: in-memory pending line limit per route, default `5000`; when full, reading pauses until pushes succeed.
- `maxLineBytes`: max bytes for one log line, default `262144` (256KiB); longer lines are skipped.
- `requestTimeout`: push request timeout, default `90s`.
- `retryBackoffMin` / `retryBackoffMax`: exponential backoff range after push failures, default `1s` / `30s`.
- `exitOnMaxBackoff`: exit after another failure at max backoff, useful when systemd or Kubernetes should restart the process.

Legacy single-site config is still supported:
```json
{
  "server": "http://10.0.0.5:8089",
  "accessKey": "your-key",
  "websiteID": "abcd",
  "sourceID": "agent-main",
  "paths": ["/var/log/nginx/access.log"],
  "pollInterval": "1s"
}
```

#### Running the agent
Run directly:
```bash
./bin/nginxpulse-agent -config /etc/nginxpulse/agent.json
```

systemd example:
```ini
[Unit]
Description=NginxPulse Agent
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/local/bin/nginxpulse-agent -config /etc/nginxpulse/agent.json
Restart=always
RestartSec=5s
User=nginx
Group=nginx

[Install]
WantedBy=multi-user.target
```

Docker example:
```bash
docker run -d --name nginxpulse-agent \
  -v /etc/nginxpulse/agent.json:/etc/nginxpulse/agent.json:ro \
  -v /var/log/nginx:/var/log/nginx:ro \
  nginxpulse-agent:local
```

#### Environment overrides
The config file is the primary source. These environment variables can override selected runtime options, which is useful for Docker/Kubernetes:

- `NGINXPULSE_AGENT_POLL_INTERVAL`
- `NGINXPULSE_AGENT_FLUSH_INTERVAL`
- `NGINXPULSE_AGENT_INITIAL_TAIL_BYTES`
- `NGINXPULSE_AGENT_INITIAL_MAX_LINES`
- `NGINXPULSE_AGENT_BATCH_SIZE`
- `NGINXPULSE_AGENT_REQUEST_TIMEOUT`
- `NGINXPULSE_AGENT_MAX_PENDING_LINES`
- `NGINXPULSE_AGENT_MAX_LINE_BYTES`
- `NGINXPULSE_AGENT_RETRY_BACKOFF_MIN`
- `NGINXPULSE_AGENT_RETRY_BACKOFF_MAX`
- `NGINXPULSE_AGENT_EXIT_ON_MAX_BACKOFF`

#### Verification and troubleshooting
1) Check server connectivity:
```bash
curl -i -H "X-NginxPulse-Key: your-key" http://10.0.0.5:8089/healthz
```

2) Check site IDs:
```bash
curl -H "X-NginxPulse-Key: your-key" http://10.0.0.5:8089/api/websites
```

3) After starting the agent, watch its logs:
- `config loaded`: config loaded successfully; the log includes `endpoint`, `routes`, and `batch_size`.
- `read new lines`: new local log lines were read.
- `push succeeded`: lines were pushed to the server.
- `日志推送失败，将按退避重试`: network, access key, site ID, or server parse error.

Common issues:
- `401 Unauthorized`: `accessKey` does not match server `ACCESS_KEYS`, or the server has access keys enabled but the agent config is missing one.
- `400 site not found`: wrong `websiteID`; fetch `/api/websites` again.
- `push succeeded` but no frontend data: check the parse config for the matching `sourceID`; if the log contains `$host`, confirm site `domains` match the Host value.
- No `read new lines`: check container mounts, file permissions, and paths. The agent reads plain text logs and skips `.gz`.
- Agent restart does not read all history: this is the default tailing behavior. Set `"initialTailBytes": -1` when you need full replay.

## Notes
- If reparse happens on restart, make sure no stale process is running.
- Globs may match more files than expected.
- Gzip logs are parsed as full files based on metadata.
