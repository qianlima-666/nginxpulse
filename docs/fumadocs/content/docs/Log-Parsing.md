---
title: "日志解析机制"
---

# 日志解析机制

## 整体流程
1. 初始扫描：启动时先解析“最近窗口”日志。
2. 增量扫描：定时任务按 `system.taskInterval` 继续扫描新增内容。
3. 历史回填：在后台逐步补齐历史日志（不阻塞实时解析）。
4. IP 归属地回填：解析日志后异步解析 IP 归属地并回填。

## 增量解析与状态文件
- 状态文件: `var/nginxpulse_data/nginx_scan_state.json`
- 若文件大小小于上次记录大小，视为轮转，从头解析。
- 站点 ID 由 `websites[].name` 生成，改名会产生新站点并重新解析。

## 批次与性能
- `system.parseBatchSize` 控制批次大小，默认 100。
- 也可通过环境变量 `LOG_PARSE_BATCH_SIZE` 覆盖。

## 解析进度与预计剩余
接口: `GET /api/status`
- `log_parsing_progress`: 解析进度（0~1）
- `log_parsing_estimated_remaining_seconds`: 预计剩余秒数
- `ip_geo_progress`: IP 归属地解析进度（0~1）
- `ip_geo_estimated_remaining_seconds`: IP 归属地预计剩余秒数

前端可按固定间隔轮询该接口以刷新进度。

## 10G+ 大日志优化思路
- 解析日志时只写入基础字段，IP 归属地放入待解析队列。
- 归属地解析在后台批量回填，不阻塞主解析。
- 如需更快：调大 `parseBatchSize`、提高机器 IO 或将日志按天切分。

### 当前架构 vs 理想流式架构
如果目标是提升大日志解析速度，常见会想到“改成流式解析”。  
但这里需要先区分两种不同层次的“流式”：

- 当前实现：单线程逐行读取、逐行解析，攒满一批后同步批量写库。
- 理想流式架构：把“读取 / 解析 / 清洗 / 入库”拆成多阶段流水线，每一段可独立并发。

#### 当前实现简图
![image-20260331113258312](https://resource.kaisir.cn/uploads/MarkDownImg/20260331/TwbCON.png)

#### 理想流式架构简图
![image-20260331113302109](https://resource.kaisir.cn/uploads/MarkDownImg/20260331/RyUnIu.png)

#### 当前实现的优点
- 实现简单，代码路径短，排查问题更直接。
- 内存占用更可控，不会因为 worker 积压而快速膨胀。
- 顺序清晰，状态文件、解析进度和回填逻辑都更容易保持一致。
- 对单机、单站点、中低速增长日志场景足够稳定。

#### 当前实现的缺点
- 读取、解析、入库基本在同一条串行链路里，吞吐上限较低。
- 单站点超大历史日志场景下，更容易受限于单线程解析速度。
- 即使机器 CPU 和磁盘还有余量，也不容易充分利用。
- 回填和实时解析虽然做了节流隔离，但整体补历史数据会偏慢。

#### 理想流式架构的优点
- 更容易把 CPU、磁盘和数据库吞吐压榨出来，整体上限更高。
- 更适合多来源实时接入或大体量历史日志持续导入。
- 可以按阶段分别扩容，例如单独增加 parser worker 或 writer worker。
- 当某一段成为瓶颈时，更容易做定向优化。

#### 理想流式架构的缺点
- 实现复杂度明显更高，维护成本也更高。
- 需要额外处理背压、队列堆积、重复写入、顺序一致性和错误重试。
- 状态跟踪、进度展示、停机恢复都会比当前方案复杂很多。
- 如果限流设计不好，容易把 PostgreSQL、磁盘 IO 或内存瞬间打满。

#### 这两个方案怎么选
- 如果目标是“稳、简单、容易维护”，当前实现更合适。
- 如果目标是“单机尽量快地吃完超大历史日志，或者长期处理多来源高吞吐输入”，理想流式架构更有潜力。
- 对当前项目而言，优先保留现在的实现，再通过 `taskInterval`、`parseBatchSize`、日志切分和机器 IO 做调优，通常是性价比更高的路线。

### 单站点 300GB+ 历史日志导入建议
如果当前只有 **1 个站点**，但需要补齐几百 GB 的历史日志，通常 **不只是调大 `parseBatchSize`**。

需要先理解当前链路：
- 启动时只会优先解析“最近窗口”的日志，历史日志会在后台逐步回填。
- 历史回填不是无限速跑满，而是按定时任务节奏推进。
- 默认配置下，后台回填每轮有固定预算，因此 300GB 级别数据补齐可能需要较长时间。

对这类场景，建议按下面顺序调优：

1. 先调 `system.taskInterval`
   - 这会直接影响后台历史回填推进频率。
   - 默认一般是 `1m`；导入期可临时调到 `5s` 或 `10s`，让回填更积极。
   - 历史数据补齐后，再调回 `30s` 或 `1m`，避免长期占用过多资源。

2. 再调 `system.parseBatchSize`
   - 默认是 `100`。
   - 建议逐步提高，而不是一次拉太高：
   - 可先试 `500`
   - 再试 `1000`
   - 机器和 PostgreSQL 都稳定时，再考虑 `2000`
   - 批次过大可能带来更重的单事务、更多内存占用，以及失败后更高的重试成本。

3. 尽量按天切分日志
   - 相比一个超大单文件，按天或按小时切分更利于回填、重试和问题排查。
   - 例如：
   - `"/share/logs/nginx/access-*.log"`
   - `"/share/logs/nginx/access-*.log.gz"`

4. 关注磁盘和 PostgreSQL 性能
   - 单站点场景下，瓶颈通常很快会落到磁盘 IO 和数据库写入。
   - 如果 PostgreSQL 与 NginxPulse 部署在慢盘、网络盘或共享盘上，提升 `parseBatchSize` 的收益会很有限。

推荐做法：
- 导入历史日志期间，临时使用如下配置：

```json
{
  "system": {
    "taskInterval": "5s",
    "parseBatchSize": 1000
  }
}
```

- 等大体量历史日志补齐后，再恢复为更保守的配置，例如：

```json
{
  "system": {
    "taskInterval": "30s",
    "parseBatchSize": 500
  }
}
```

如果你观察到以下现象，说明参数可能已经调得过猛，需要适当回退：
- PostgreSQL CPU 或 IO 持续很高
- 容器/进程内存明显上涨
- 日志中频繁出现数据库写入失败、死锁重试或超时
- 前台实时新增日志变慢

## IIS 默认规则（W3C Extended）
NginxPulse 现已支持 `logType=iis`（别名：`iis-w3c`），默认按 IIS W3C 扩展日志的常见默认字段顺序解析：

`date time s-ip cs-method cs-uri-stem cs-uri-query s-port cs-username c-ip cs(User-Agent) cs(Referer) sc-status sc-substatus sc-win32-status time-taken`

注意点：
- 日志中以 `#` 开头的元数据行（如 `#Software`、`#Version`、`#Fields`）会自动跳过。
- URL 会优先取 `cs-uri-stem`，当 `cs-uri-query` 不是 `-` 时会自动拼接为 `path?query`。
- IIS W3C 默认时间按 UTC 记录，默认时间格式为 `2006-01-02 15:04:05`。

配置示例：
```json
{
  "name": "iis-site",
  "logPath": "/var/log/iis/u_ex*.log",
  "logType": "iis"
}
```

示例日志行：
```text
2026-02-08 10:05:34 10.0.0.10 GET /index.html a=1&b=2 443 - 203.0.113.8 Mozilla/5.0+(Windows+NT+10.0;+Win64;+x64) https://example.com/ 200 0 0 36
```

## 日志清理
- `system.logRetentionDays` 控制保留天数。
- 清理任务在系统时间凌晨 2 点触发（按系统时区）。
- 该清理仅针对“已解析入库”的访问数据；不会删除你原始的 Nginx 日志文件。
- 系统运行日志（`var/nginxpulse_data/nginxpulse.log`）走文件轮转策略，与 `logRetentionDays` 无关。
- 修改 `logRetentionDays` 后需重启服务进程/容器才会生效；如需立即按新值处理历史数据，请重启后执行“重新解析”。

## 多个日志文件如何挂载？
`WEBSITES` 是一个 **JSON 数组**，每个元素描述一个网站。`logPath` 需要填写**容器内可访问的路径**，你可以按需指定。

参考示例：
```yaml
environment:
  WEBSITES: '[{"name":"网站1","logPath":"/share/logs/nginx/access-site1.log","domains":["www.kaisir.cn","kaisir.cn"]}, {"name":"网站2","logPath":"/share/logs/nginx/access-site2.log","domains":["home.kaisir.cn"]}]'
volumes:
  - ./nginx_data/logs/site1/access.log:/share/logs/nginx/access-site1.log:ro
  - ./nginx_data/logs/site2/access.log:/share/logs/nginx/access-site2.log:ro
```

如果站点很多，一个个挂载较繁琐，可以**直接挂载整个日志目录**，再在 `WEBSITES` 里指定具体文件：
```yaml
environment:
  WEBSITES: '[{"name":"网站1","logPath":"/share/logs/nginx/access-site1.log","domains":["www.kaisir.cn","kaisir.cn"]}, {"name":"网站2","logPath":"/share/logs/nginx/access-site2.log","domains":["home.kaisir.cn"]}]'
volumes:
  - ./nginx_data/logs:/share/logs/nginx/
```

> 注意：如果 Nginx 日志按天切割，可用 `*` 替代日期，例如：`{"logPath":"/share/logs/nginx/site1.top-*.log"}`。

#### 压缩日志（.gz）
支持直接解析 `.gz` 压缩日志，`logPath` 可指向单个 `.gz` 文件或使用通配符：
```json
{"logPath": "/share/logs/nginx/access-*.log.gz"}
```
项目内提供 gzip 参考样例：`var/log/gz-log-read-test/`。

## 远端日志支持（sources）
当日志不方便挂载到本机或容器时，可在站点配置中使用 `sources` 替代 `logPath`。一旦配置 `sources`，`logPath` 会被忽略。

`sources` 接受 **JSON 数组**，每一项表示一个日志来源配置。这样设计是为了：
1) 同一站点可接入多个来源（多台机器/多目录/多桶并行）。
2) 不同来源可使用不同解析/鉴权/轮询策略，方便扩展与灰度切换。
3) 轮转/归档场景下按来源拆分，后续新增来源无需改动旧配置。

通用字段：
- `id`：来源唯一标识（建议全站唯一）。
- `type`：`local` / `sftp` / `http` / `s3` / `agent`。
- `mode`：
  - `poll`：按间隔拉取（默认）。
  - `stream`：仅流式输入（当前仅 Push Agent 生效）。
  - `hybrid`：流式 + 轮询兜底（当前仅 Push Agent 会流式，其它来源仍按 `poll`）。
- `pollInterval`：轮询间隔（如 `5s`）。
- `pattern`：轮转匹配（SFTP/Local/S3 使用 glob；HTTP 依赖 index JSON）。
- `compression`：`auto` / `gz` / `none`。
- `parse`：覆盖解析格式（见下文“解析覆盖”）。
> `stream` 模式目前主要用于 Push Agent，其它来源会按 `poll` 处理。

### 方案一：HTTP 服务暴露日志
适合你能在日志服务器上提供 HTTP 访问（内网或加鉴权）的场景。

方式 A：Nginx/Apache 直接暴露日志文件（务必限制访问，避免日志泄露）
```nginx
location /logs/ {
  alias /var/log/nginx/;
  autoindex on;
  # 建议加 basic auth / IP 白名单
}
```

然后在 `sources` 配置：
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

`rangePolicy` 说明：
- `auto`：优先 Range，不支持则自动回退为整包下载（会跳过已读字节）。
- `range`：强制 Range，不支持则报错。
- `full`：始终整包下载。

方式 B：自建 JSON 索引 API  
适合轮转日志（按天/按小时）或 `.gz` 归档：
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

更详细的索引 API 约定（建议）：
1) 索引接口返回一个 JSON，包含日志对象数组。
2) 每条对象至少提供 `path`（可访问 URL）。
3) 建议提供 `size` / `mtime` / `etag`，用于变更检测与避免重复解析。
4) `mtime` 支持 RFC3339 / RFC3339Nano / `2006-01-02 15:04:05` / Unix 秒时间戳。

推荐返回示例：
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

如果你的字段名不同，可以在 `jsonMap` 中映射：
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

注意事项：
- `path` 必须是可直接访问的日志 URL。
- `.gz` 文件建议提供稳定的 `etag` / `size` / `mtime`，否则可能重复解析。
- 如果 HTTP 服务不支持 Range，建议将 `rangePolicy` 设为 `auto` 或 `full`。

### 方案二：SFTP 直连拉取
适合你能开放 SSH/SFTP 端口的场景，无需额外 HTTP 服务。
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
> `auth` 支持 `keyFile`、`passphrase`（私钥口令）和 `password`。

#### SFTP 密钥登录实操（本机 -> 远端）
1) 在本机生成专用密钥（推荐 `ed25519`）：
```bash
ssh-keygen -t ed25519 -a 100 -f ~/.ssh/nginxpulse_sftp -C "nginxpulse-sftp"
```

2) 将公钥写入远端用户（需要先能用密码或已有方式登录）：
```bash
ssh-copy-id -i ~/.ssh/nginxpulse_sftp.pub <user>@<host>
```
若没有 `ssh-copy-id`，可手动执行：
```bash
cat ~/.ssh/nginxpulse_sftp.pub | ssh <user>@<host> \
'mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys'
```

3) 在远端确认权限（以当前登录用户为例）：
```bash
chmod 700 ~/.ssh
chmod 600 ~/.ssh/authorized_keys
```

4) 在本机验证 SSH 密钥登录（强制只走公钥认证）：
```bash
ssh -i ~/.ssh/nginxpulse_sftp -o PreferredAuthentications=publickey <user>@<host>
```

5) 在本机验证 SFTP 密钥登录：
```bash
sftp -i ~/.ssh/nginxpulse_sftp <user>@<host>
```

6) 验证通过后，再填入 `sources`：
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
> `keyFile` 路径必须是运行 NginxPulse 的机器（或容器）内可访问的绝对路径。

7) 若仍失败，建议先用调试日志定位：
```bash
ssh -vvv -i ~/.ssh/nginxpulse_sftp -o PreferredAuthentications=publickey <user>@<host>
```
Alpine 常见日志查看：
```bash
grep sshd /var/log/messages | tail -n 80
```

### 方案三：对象存储（S3/OSS）
适合日志统一归档到 OSS/S3（支持阿里云/腾讯云/AWS 兼容端点）。
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

### 解析覆盖（sources[].parse）
当同一站点不同来源日志格式不一致时，可在 `sources[].parse` 内覆盖：
```json
{
  "parse": {
    "logType": "nginx",
    "logRegex": "^(?P<ip>\\S+) - (?P<user>\\S+) \\[(?P<time>[^\\]]+)\\] \"(?P<request>[^\"]+)\" (?P<status>\\d+) (?P<bytes>\\d+) \"(?P<referer>[^\"]*)\" \"(?P<ua>[^\"]*)\"$",
    "timeLayout": "02/Jan/2006:15:04:05 -0700"
  }
}
```

### Push Agent（实时推送）
适合内网、边缘节点、Kubernetes 节点或不方便把日志目录挂载到 NginxPulse 主服务的场景。Agent 是一个独立采集进程，它在日志服务器上读取本地日志新增行，然后推送到 NginxPulse 主服务的 `POST /api/ingest/logs`。

> 术语说明：这里的 Agent 指日志采集代理进程，不是 AI 大模型 Agent（LLM Agent）。

#### Agent 是哪个包
仓库里有三个可用入口：

- 源码入口：`cmd/nginxpulse-agent`
- 预编译二进制：
  - `prebuilt/nginxpulse-agent-linux-amd64`
  - `prebuilt/nginxpulse-agent-darwin-arm64`
- 容器构建文件：`Dockerfile.agent`

构建二进制：
```bash
go build -trimpath -ldflags="-s -w" -o bin/nginxpulse-agent ./cmd/nginxpulse-agent
```

构建容器镜像：
```bash
docker build -f Dockerfile.agent -t nginxpulse-agent:local .
```

#### 工作方式
Agent 不直接连接 PostgreSQL，也不读取 NginxPulse 的数据目录。它只做三件事：

1. 按 `routes[].paths` 读取日志服务器上的本地文件。
2. 维护进程内 offset，持续读取新增行。
3. 批量 POST 到主服务 `/api/ingest/logs`，由主服务按站点解析、去重、入库。

首次读取大文件时，Agent 默认只从文件尾部读取最近 `8MiB`，避免第一次部署就把历史日志全量灌入主服务。需要全量回放时，将 `initialTailBytes` 设为 `-1`。

#### 主服务配置（运行 NginxPulse 的机器）
1) 确保主服务 HTTP 地址能被日志服务器访问，例如 `http://10.0.0.5:8089`。
2) 建议启用访问密钥：设置环境变量 `ACCESS_KEYS`，或配置 `system.accessKeys`。
3) 获取 `websiteID`：
```bash
curl -H "X-NginxPulse-Key: your-key" http://10.0.0.5:8089/api/websites
```
返回中的 `id` 就是 Agent 配置里的 `websiteID`。

4) 如果日志格式使用站点默认解析规则，可以不写 `sources`。如果想给 Agent 单独指定解析格式，在站点配置中添加 `type=agent` 的 source，且 `id` 必须等于 Agent 的 `sourceID`：
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

`type=agent` 的 source 只用于识别 `sourceID` 和解析覆盖，不会被主服务定期扫描。

#### Agent 配置（日志服务器）
创建 `/etc/nginxpulse/agent.json`：
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

字段说明：

- `server`: NginxPulse 主服务地址，不要写 `/api/ingest/logs`，Agent 会自动拼接。
- `accessKey`: 对应主服务 `ACCESS_KEYS` / `system.accessKeys`；未启用访问密钥时可留空。
- `routes`: 多站点/多日志路径配置。每个 route 对应一个站点和一个 `sourceID`。
- `routes[].websiteID`: 主服务 `/api/websites` 返回的站点 ID。
- `routes[].sourceID`: 推送来源 ID。若主服务站点配置了 `type=agent` source，这里必须与该 source 的 `id` 一致。
- `routes[].paths`: 日志服务器上的本地文件路径。当前 Agent 读取普通文本日志，跳过 `.gz` 文件。
- `pollInterval`: 读取新增日志的间隔，默认 `1s`。
- `batchSize`: pending 行数达到该值时立即推送，默认 `200`。
- `flushInterval`: 即使未达到 `batchSize`，也会按该间隔推送已有 pending 行，默认 `2s`。
- `initialTailBytes`: 首次读取大文件时只读末尾 N 字节；`0` 使用默认 `8MiB`，`-1` 表示从文件头开始。
- `initialMaxLines`: 首次读取最多行数，`0` 表示不额外限制。
- `maxPendingLines`: 单 route 内存缓冲上限，默认 `5000`；网络异常时达到上限会暂停读取。
- `maxLineBytes`: 单行最大字节数，默认 `262144`（256KiB），超长行会跳过。
- `requestTimeout`: 推送请求超时，默认 `90s`。
- `retryBackoffMin` / `retryBackoffMax`: 推送失败后的指数退避范围，默认 `1s` / `30s`。
- `exitOnMaxBackoff`: 达到最大退避后再次失败是否退出进程，适合 Kubernetes/systemd 交给外部重启。

单站点旧写法仍可用：
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

#### 运行方式
直接运行：
```bash
./bin/nginxpulse-agent -config /etc/nginxpulse/agent.json
```

systemd 示例：
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

Docker 示例：
```bash
docker run -d --name nginxpulse-agent \
  -v /etc/nginxpulse/agent.json:/etc/nginxpulse/agent.json:ro \
  -v /var/log/nginx:/var/log/nginx:ro \
  nginxpulse-agent:local
```

#### 环境变量覆盖
配置文件为主，以下环境变量可覆盖部分运行参数，方便 Docker/Kubernetes 管理：

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

#### 验证与排障
1) 检查主服务连通性：
```bash
curl -i -H "X-NginxPulse-Key: your-key" http://10.0.0.5:8089/healthz
```

2) 检查站点 ID：
```bash
curl -H "X-NginxPulse-Key: your-key" http://10.0.0.5:8089/api/websites
```

3) 启动 Agent 后观察日志：
- `config loaded`: 配置加载成功，会打印 `endpoint`、`routes`、`batch_size`。
- `read new lines`: 已从本地日志读到新增行。
- `push succeeded`: 已推送到主服务。
- `日志推送失败，将按退避重试`: 网络、访问密钥、站点 ID 或主服务解析异常。

常见问题：
- `401 Unauthorized`: `accessKey` 与主服务 `ACCESS_KEYS` 不一致，或主服务启用了访问密钥但 Agent 没配置。
- `400 站点不存在`: `websiteID` 写错，重新请求 `/api/websites`。
- 有 `push succeeded` 但前端没数据：检查 `sourceID` 对应的解析格式；如果日志里带 `$host`，确认站点 `domains` 与 Host 匹配。
- 没有 `read new lines`: 检查容器挂载、文件权限、路径是否存在；Agent 只读普通文本日志，跳过 `.gz`。
- Agent 重启后没有从头读历史日志：这是默认行为。需要全量回放时设置 `"initialTailBytes": -1`。

## 常见注意点
- 若重启后重复解析，请确认没有残留进程占用同一端口。
- 日志路径支持通配符，注意匹配到的文件数量。
- gzip 日志会按文件全量解析（基于文件元信息判断是否变更）。
