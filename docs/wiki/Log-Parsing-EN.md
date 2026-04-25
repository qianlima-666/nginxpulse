# Log Parsing

This page explains how NginxPulse parses, incrementally scans, backfills, and cleans logs. For source setup, see [Log Sources](Log-Sources-EN). For format examples, see [Supported Log Formats](Supported-Log-Formats-EN). For Push Agent, see [Agent Collection](Agent-EN).

## Flow
1. Initial scan: parse the recent window after startup.
2. Incremental scan: scan appended logs by `system.taskInterval`.
3. Historical backfill: fill older logs in the background without blocking realtime parsing.
4. IP geo backfill: resolve IP locations asynchronously after logs are inserted.

## Incremental Scan And State
- State file: `var/nginxpulse_data/nginx_scan_state.json`
- If current file size is smaller than the last recorded size, it is treated as log rotation and parsed from the beginning.
- Site ID is derived from `websites[].name`. Renaming creates a new site and reparses logs.
- `.gz` logs are parsed as whole files and change detection is based on file metadata.

## Batch Size And Performance
- `system.parseBatchSize` controls insert batch size, default 100.
- It can be overridden by `LOG_PARSE_BATCH_SIZE`.
- Larger batches can improve historical import throughput, but increase transaction pressure and memory usage.

## Progress And ETA
Endpoint: `GET /api/status`

- `log_parsing_progress`: log parsing progress, from 0 to 1.
- `log_parsing_estimated_remaining_seconds`: estimated remaining seconds for log parsing.
- `ip_geo_progress`: IP geo progress, from 0 to 1.
- `ip_geo_estimated_remaining_seconds`: estimated remaining seconds for IP geo backfill.

The frontend can poll this endpoint to refresh progress.

## Retention And Cleanup
- `system.logRetentionDays` controls retention, default 30.
- Cleanup runs at 02:00 in system timezone.
- Cleanup only removes parsed access data in the database. It does not delete raw log files.
- Application logs in `var/nginxpulse_data/nginxpulse.log` use log rotation and are not controlled by `logRetentionDays`.
- Restart the service/container after changing `logRetentionDays`.
- To rebuild historical parsed data with a new retention value, restart and run “Reparse”.

## 10G+ Log Optimization
- Parsing writes core fields first and queues IP geo work.
- IP geo is resolved in batches in the background.
- For better throughput: increase `parseBatchSize`, shorten import-time `taskInterval`, improve disk/database IO, or split logs by day.

## Current Architecture vs Ideal Streaming Architecture
When optimizing large log imports, “streaming parsing” can mean two different things:

- Current implementation: read line by line, parse line by line, then synchronously batch insert.
- Ideal streaming architecture: split read / parse / clean / write into independently concurrent pipeline stages.

### Current Implementation
![Current implementation](https://resource.kaisir.cn/uploads/MarkDownImg/20260331/TwbCON.png)

### Ideal Streaming Architecture
![Ideal streaming architecture](https://resource.kaisir.cn/uploads/MarkDownImg/20260331/RyUnIu.png)

### Strengths Of Current Implementation
- Simple implementation and shorter debugging path.
- More predictable memory usage without worker backlog explosions.
- Easier consistency for state files, progress, and backfill logic.
- Stable enough for single-node, single-site, low-to-medium growth logs.

### Weaknesses Of Current Implementation
- Read, parse, and write are mostly serial, so throughput is limited.
- Very large single-site historical imports are more likely CPU-bound by single-thread parsing.
- Available CPU and disk capacity are not fully utilized.

### Strengths Of Ideal Streaming Architecture
- Higher throughput ceiling across CPU, disk, and database.
- Better fit for high-volume multi-source realtime ingestion or massive historical imports.
- Easier to scale specific stages, such as parser workers or writer workers.

### Weaknesses Of Ideal Streaming Architecture
- Much higher implementation and maintenance complexity.
- Requires backpressure, queue management, duplicate handling, ordering guarantees, and retry design.
- State tracking, progress display, and crash recovery all become more complex.
- Poor throttling can overwhelm PostgreSQL, disk IO, or memory.

## Single-Site 300GB+ Historical Import
For one site with hundreds of GB of historical logs, increasing `parseBatchSize` alone is usually not enough.

Recommended tuning order:

1. Tune `system.taskInterval` first
   - Default is usually `1m`.
   - During import, temporarily use `5s` or `10s`.
   - After backfill finishes, restore it to `30s` or `1m`.

2. Tune `system.parseBatchSize`
   - Default is `100`.
   - Increase gradually: try `500`, then `1000`.
   - If both the machine and PostgreSQL remain stable, consider `2000`.
   - Too large a batch means heavier transactions, higher memory usage, and higher retry cost after failures.

3. Split logs by day
   - Daily or hourly files are easier to backfill, retry, and debug than one huge file.
   - Examples: `"/share/logs/nginx/access-*.log"` or `"/share/logs/nginx/access-*.log.gz"`

4. Watch disk and PostgreSQL performance
   - For single-site imports, bottlenecks often move to disk IO and database writes.
   - If PostgreSQL or NginxPulse runs on slow, network, or shared storage, increasing `parseBatchSize` may not help much.

Import-time example:
```json
{
  "system": {
    "taskInterval": "5s",
    "parseBatchSize": 1000
  }
}
```

After import, use a more conservative config:
```json
{
  "system": {
    "taskInterval": "30s",
    "parseBatchSize": 500
  }
}
```

If these symptoms appear, roll back the tuning:
- PostgreSQL CPU or IO remains high.
- Container/process memory rises noticeably.
- Logs frequently show database write failures, deadlock retries, or timeouts.
- Realtime new logs become slower in the frontend.

## Related Docs
- [Configuration](Configuration-EN)
- [Log Sources](Log-Sources-EN)
- [Supported Log Formats](Supported-Log-Formats-EN)
- [Agent Collection](Agent-EN)
- [IP Geo](IP-Geo-EN)
