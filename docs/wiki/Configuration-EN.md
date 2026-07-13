# Configuration

This page focuses on getting NginxPulse configured quickly. Full field details, log sources, and log format examples are split into dedicated pages to keep this entry point lightweight.

## Config Location
- Default config: `configs/nginxpulse_config.json`
- Local development: `scripts/dev_local.sh` uses `configs/nginxpulse_config.dev.json`
- Environment injection: `CONFIG_JSON` or `WEBSITES`

## Minimal Config
If the log file and PostgreSQL are reachable, start with this:

```json
{
  "websites": [
    {
      "name": "Main Site",
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

## Fields You Usually Need To Change
- `websites[].name`: site name. It is used to derive the site ID; renaming creates a new site.
- `websites[].logPath`: log path. For remote, multi-source, or Agent collection, see [Log Sources](Log-Sources-EN).
- `websites[].domains`: domain list. Recommended for same-site referer detection.
- `websites[].logType`: log type. Supported values and samples are in [Supported Log Formats](Supported-Log-Formats-EN).
- `websites[].autoDiscoverHosts`: use this entry as a discovery template and generate real sites from parsed `host` values.
- `websites[].customLabel`: custom site tag shown in the site selector when `autoDiscoverHosts` is enabled; falls back to the default label when empty.
- `websites[].tags`: extra tag array returned together with the site info.
- `database.dsn`: PostgreSQL connection string.
- `server.Port`: Web/API listen port. `:8089` is the common default.

## Common Scenarios

### Local or Container Nginx Logs
```json
{
  "name": "Main Site",
  "logPath": "/var/log/nginx/access.log",
  "domains": ["example.com", "www.example.com"],
  "logType": "nginx"
}
```

For rotated daily logs, use a glob:
```json
{
  "name": "Main Site",
  "logPath": "/var/log/nginx/access-*.log",
  "domains": ["example.com"],
  "logType": "nginx"
}
```

### Docker-Mounted Logs
The config must use the path inside the container, not the host path:
```yaml
volumes:
  - /var/log/nginx:/share/logs/nginx:ro
```

```json
{
  "name": "Main Site",
  "logPath": "/share/logs/nginx/access.log",
  "domains": ["example.com"],
  "logType": "nginx"
}
```

### Multiple Sites
```json
{
  "websites": [
    {
      "name": "Main Site",
      "logPath": "/share/logs/nginx/main-access.log",
      "domains": ["example.com", "www.example.com"],
      "logType": "nginx"
    },
    {
      "name": "Blog",
      "logPath": "/share/logs/nginx/blog-access.log",
      "domains": ["blog.example.com"],
      "logType": "nginx"
    }
  ]
}
```

### Host-Based Site Auto Discovery
In plain terms: before this feature, you had to tell NginxPulse which sites exist. With this feature, NginxPulse can read the domain from logs first, then create sites automatically.

Previously, if one Nginx / Nginx Proxy Manager instance hosted many domains, you usually had to configure each site manually:
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

If you later add `c.com`, you also need to add another site config.

With `autoDiscoverHosts`, you only configure one discovery template. NginxPulse periodically scans matching logs, extracts domains from the `host` field, and registers real sites by domain. For example, if the logs contain `a.com`, `b.com`, and `c.com`, the site selector will show those 3 sites; each site only counts log entries for its own domain.

The requirement is that the log format must include a domain field, such as `$host` or `$http_host`. If the logs only contain IP, time, and URL, NginxPulse cannot know which site a request belongs to.

nginx-proxy-manager can use the built-in log type directly:
```json
{
  "name": "NPM Auto Discover",
  "logPath": "/share/logs/npm/proxy-host-*_access.log",
  "logType": "nginx-proxy-manager",
  "autoDiscoverHosts": true
}
```

Plain Nginx or custom logs also work as long as `logFormat` / `logRegex` can parse a `host` value. For example:
```json
{
  "name": "Host Auto Discover",
  "logPath": "/share/logs/nginx/*.log",
  "logFormat": "$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\" $host",
  "autoDiscoverHosts": true
}
```

If you use `logRegex`, include a named group like `(?P<host>...)`.

### Remote Logs or Multi-Source Collection
When logs are not on the NginxPulse host/container, use `websites[].sources`:
```json
{
  "name": "Main Site",
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

`autoDiscoverHosts` also works with polling `sources` such as `local`, `sftp`, `http`, and `s3`. The same requirement still applies: the log format must expose a parsable `host` field, and `mode` must not be `stream`. For example:
```json
{
  "name": "SFTP Host Auto Discover",
  "sources": [
    {
      "id": "sftp-main",
      "type": "sftp",
      "host": "10.0.0.10",
      "port": 22,
      "user": "nginx",
      "auth": { "keyFile": "/home/nginxpulse/.ssh/id_ed25519" },
      "path": "/var/log/nginx/access.log"
    }
  ],
  "logFormat": "$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\" $host",
  "autoDiscoverHosts": true
}
```

More `local`, `sftp`, `http`, `s3`, and `agent` examples are in [Log Sources](Log-Sources-EN).

### Custom Log Format
If no built-in `logType` matches your logs, prefer `logFormat`:
```json
{
  "name": "Custom Site",
  "logPath": "/var/log/custom/access.log",
  "logFormat": "$remote_addr [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\""
}
```

Custom fields and full examples are in [Supported Log Formats](Supported-Log-Formats-EN).

### Reverse Proxy Subpath
To serve NginxPulse under `/nginxpulse/`:
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

Note: `webBasePath` only supports one path segment. Restart the service after changing it.

## Detailed Docs
- [Config Reference](Config-Reference-EN)
- [Log Sources](Log-Sources-EN)
- [Supported Log Formats](Supported-Log-Formats-EN)
- [Agent Collection](Agent-EN)
- [Log Parsing](Log-Parsing-EN)
- [IP Geo](IP-Geo-EN)
