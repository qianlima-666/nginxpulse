# 日志解析机制

本文只解释 NginxPulse 如何解析、增量扫描、回填和清理日志。日志从哪里来见[日志来源配置](Log-Sources)，日志格式示例见[支持的日志格式](Supported-Log-Formats)，Push Agent 见[Agent 采集](Agent)。

## 整体流程
1. 初始扫描：启动时先解析“最近窗口”日志。
2. 增量扫描：定时任务按 `system.taskInterval` 继续扫描新增内容。
3. 历史回填：在后台逐步补齐历史日志，不阻塞实时解析。
4. IP 归属地回填：日志入库后异步解析 IP 归属地并回填。

## 增量解析与状态文件
- 状态文件：`var/nginxpulse_data/nginx_scan_state.json`
- 如果当前文件大小小于上次记录大小，视为日志轮转，会从头解析。
- 站点 ID 由 `websites[].name` 生成，改名会产生新站点并重新解析。
- `.gz` 日志按文件整体解析，基于文件元信息判断是否变更。

## 批次与性能
- `system.parseBatchSize` 控制单批入库条数，默认 100。
- 可通过环境变量 `LOG_PARSE_BATCH_SIZE` 覆盖。
- 调大批次可提升历史导入吞吐，但会增加单事务压力和内存占用。

## 解析进度与预计剩余
接口：`GET /api/status`

- `log_parsing_progress`: 日志解析进度，范围 0~1。
- `log_parsing_estimated_remaining_seconds`: 日志解析预计剩余秒数。
- `ip_geo_progress`: IP 归属地解析进度，范围 0~1。
- `ip_geo_estimated_remaining_seconds`: IP 归属地预计剩余秒数。

前端可按固定间隔轮询该接口刷新进度。

## 日志保留与清理
- `system.logRetentionDays` 控制保留天数，默认 30。
- 清理任务在系统时间凌晨 2 点触发。
- 清理仅针对“已解析入库”的访问数据，不删除原始日志文件。
- 系统运行日志 `var/nginxpulse_data/nginxpulse.log` 走文件轮转策略，与 `logRetentionDays` 无关。
- 修改 `logRetentionDays` 后需重启服务进程/容器，解析器才会按新值过滤入库日志。
- 如需立即按新值重建历史入库数据，请重启后执行“重新解析”。

## 10G+ 大日志优化思路
- 解析日志时只写入基础字段，IP 归属地放入待解析队列。
- IP 归属地在后台批量回填，不阻塞主解析。
- 如需更快：调大 `parseBatchSize`、缩短导入期 `taskInterval`、提高磁盘/数据库 IO，或将日志按天切分。

## 当前架构 vs 理想流式架构
如果目标是提升大日志解析速度，常见会想到“改成流式解析”。这里需要先区分两种层次：

- 当前实现：单线程逐行读取、逐行解析，攒满一批后同步批量写库。
- 理想流式架构：把“读取 / 解析 / 清洗 / 入库”拆成多阶段流水线，每一段可独立并发。

### 当前实现简图
![当前实现简图](https://resource.kaisir.cn/uploads/MarkDownImg/20260331/TwbCON.png)

### 理想流式架构简图
![理想流式架构简图](https://resource.kaisir.cn/uploads/MarkDownImg/20260331/RyUnIu.png)

### 当前实现的优点
- 实现简单，代码路径短，排查问题直接。
- 内存占用更可控，不会因为 worker 积压而快速膨胀。
- 状态文件、解析进度和回填逻辑更容易保持一致。
- 对单机、单站点、中低速增长日志场景足够稳定。

### 当前实现的缺点
- 读取、解析、入库基本在同一条串行链路里，吞吐上限较低。
- 单站点超大历史日志场景下，更容易受限于单线程解析速度。
- 即使机器 CPU 和磁盘还有余量，也不容易充分利用。

### 理想流式架构的优点
- 更容易压榨 CPU、磁盘和数据库吞吐，整体上限更高。
- 更适合多来源实时接入或大体量历史日志持续导入。
- 可以按阶段分别扩容，例如单独增加 parser worker 或 writer worker。

### 理想流式架构的缺点
- 实现复杂度和维护成本明显更高。
- 需要额外处理背压、队列堆积、重复写入、顺序一致性和错误重试。
- 状态跟踪、进度展示、停机恢复都会比当前方案复杂。
- 如果限流设计不好，容易把 PostgreSQL、磁盘 IO 或内存瞬间打满。

## 单站点 300GB+ 历史日志导入建议
如果当前只有 1 个站点，但需要补齐几百 GB 的历史日志，通常不只是调大 `parseBatchSize`。

建议按下面顺序调优：

1. 先调 `system.taskInterval`
   - 默认一般是 `1m`。
   - 导入期可临时调到 `5s` 或 `10s`，让回填更积极。
   - 历史数据补齐后，再调回 `30s` 或 `1m`。

2. 再调 `system.parseBatchSize`
   - 默认是 `100`。
   - 建议逐步提高：先试 `500`，再试 `1000`。
   - 机器和 PostgreSQL 都稳定时，再考虑 `2000`。
   - 批次过大可能带来更重的单事务、更多内存占用，以及失败后更高的重试成本。

3. 尽量按天切分日志
   - 相比一个超大单文件，按天或按小时切分更利于回填、重试和问题排查。
   - 示例：`"/share/logs/nginx/access-*.log"` 或 `"/share/logs/nginx/access-*.log.gz"`

4. 关注磁盘和 PostgreSQL 性能
   - 单站点场景下，瓶颈通常很快会落到磁盘 IO 和数据库写入。
   - 如果 PostgreSQL 与 NginxPulse 部署在慢盘、网络盘或共享盘上，提升 `parseBatchSize` 的收益会很有限。

导入期示例：
```json
{
  "system": {
    "taskInterval": "5s",
    "parseBatchSize": 1000
  }
}
```

历史补齐后建议恢复为更保守的配置：
```json
{
  "system": {
    "taskInterval": "30s",
    "parseBatchSize": 500
  }
}
```

如果出现以下现象，说明参数可能已经调得过猛，需要回退：
- PostgreSQL CPU 或 IO 持续很高。
- 容器/进程内存明显上涨。
- 日志中频繁出现数据库写入失败、死锁重试或超时。
- 前台实时新增日志变慢。

## 相关文档
- [配置说明](Configuration)
- [日志来源配置](Log-Sources)
- [支持的日志格式](Supported-Log-Formats)
- [Agent 采集](Agent)
- [IP 归属地解析](IP-Geo)
