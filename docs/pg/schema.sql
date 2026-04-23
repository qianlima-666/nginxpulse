-- PostgreSQL schema template for nginxpulse
-- Replace {{website_id}} with the actual website id (e.g. site1).
-- This keeps epoch seconds in BIGINT and uses DATE for daily buckets.
-- For local day bucketing, set the connection timezone, e.g.:
--   SET TIME ZONE 'Asia/Shanghai';

-- Dimension tables
CREATE TABLE IF NOT EXISTS "{{website_id}}_dim_ip" (
  id BIGSERIAL PRIMARY KEY,
  ip TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS "{{website_id}}_dim_url" (
  id BIGSERIAL PRIMARY KEY,
  url TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS "{{website_id}}_dim_referer" (
  id BIGSERIAL PRIMARY KEY,
  referer TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS "{{website_id}}_dim_ua" (
  id BIGSERIAL PRIMARY KEY,
  browser TEXT NOT NULL,
  os TEXT NOT NULL,
  device TEXT NOT NULL,
  UNIQUE (browser, os, device)
);

-- IP geo cache (global)
CREATE TABLE IF NOT EXISTS "ip_geo_cache" (
  ip TEXT PRIMARY KEY,
  domestic TEXT NOT NULL,
  global TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT 'unknown',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ip_geo_cache_created_at ON "ip_geo_cache"(created_at);

-- IP geo pending queue (global)
CREATE TABLE IF NOT EXISTS "ip_geo_pending" (
  ip TEXT PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ip_geo_pending_updated_at ON "ip_geo_pending"(updated_at);

CREATE TABLE IF NOT EXISTS "{{website_id}}_dim_location" (
  id BIGSERIAL PRIMARY KEY,
  domestic TEXT NOT NULL,
  global TEXT NOT NULL,
  UNIQUE (domestic, global)
);

-- Log table (partitioned)
CREATE TABLE IF NOT EXISTS "{{website_id}}_nginx_logs" (
  id BIGSERIAL NOT NULL,
  ip_id BIGINT NOT NULL,
  pageview_flag SMALLINT NOT NULL DEFAULT 0,
  timestamp BIGINT NOT NULL,
  method TEXT NOT NULL,
  url_id BIGINT NOT NULL,
  status_code INT NOT NULL,
  bytes_sent BIGINT NOT NULL,
  request_length BIGINT NOT NULL DEFAULT 0,
  request_time_ms BIGINT NOT NULL DEFAULT 0,
  upstream_response_time_ms BIGINT NOT NULL DEFAULT 0,
  upstream_addr TEXT NOT NULL DEFAULT '',
  host TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  fingerprint TEXT,
  referer_id BIGINT NOT NULL,
  ua_id BIGINT NOT NULL,
  location_id BIGINT NOT NULL,
  PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

-- Optional: create a monthly partition (example)
-- CREATE TABLE IF NOT EXISTS "{{website_id}}_nginx_logs_2025_01"
--   PARTITION OF "{{website_id}}_nginx_logs"
--   FOR VALUES FROM (
--     EXTRACT(EPOCH FROM TIMESTAMPTZ '2025-01-01 00:00:00+08')::BIGINT
--   ) TO (
--     EXTRACT(EPOCH FROM TIMESTAMPTZ '2025-02-01 00:00:00+08')::BIGINT
--   );

-- Default partition (accepts all rows if no explicit partitions exist)
CREATE TABLE IF NOT EXISTS "{{website_id}}_nginx_logs_default"
  PARTITION OF "{{website_id}}_nginx_logs"
  DEFAULT;

-- Aggregates
CREATE TABLE IF NOT EXISTS "{{website_id}}_agg_hourly" (
  bucket BIGINT PRIMARY KEY,
  pv BIGINT NOT NULL DEFAULT 0,
  traffic BIGINT NOT NULL DEFAULT 0,
  s2xx BIGINT NOT NULL DEFAULT 0,
  s3xx BIGINT NOT NULL DEFAULT 0,
  s4xx BIGINT NOT NULL DEFAULT 0,
  s5xx BIGINT NOT NULL DEFAULT 0,
  other BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS "{{website_id}}_agg_hourly_ip" (
  bucket BIGINT NOT NULL,
  ip_id BIGINT NOT NULL,
  PRIMARY KEY (bucket, ip_id)
);

CREATE TABLE IF NOT EXISTS "{{website_id}}_agg_daily" (
  day DATE PRIMARY KEY,
  pv BIGINT NOT NULL DEFAULT 0,
  traffic BIGINT NOT NULL DEFAULT 0,
  s2xx BIGINT NOT NULL DEFAULT 0,
  s3xx BIGINT NOT NULL DEFAULT 0,
  s4xx BIGINT NOT NULL DEFAULT 0,
  s5xx BIGINT NOT NULL DEFAULT 0,
  other BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS "{{website_id}}_agg_daily_ip" (
  day DATE NOT NULL,
  ip_id BIGINT NOT NULL,
  PRIMARY KEY (day, ip_id)
);

-- First seen
CREATE TABLE IF NOT EXISTS "{{website_id}}_first_seen" (
  ip_id BIGINT PRIMARY KEY,
  first_ts BIGINT NOT NULL
);

-- Sessions
CREATE TABLE IF NOT EXISTS "{{website_id}}_sessions" (
  id BIGSERIAL PRIMARY KEY,
  ip_id BIGINT NOT NULL,
  ua_id BIGINT NOT NULL,
  location_id BIGINT NOT NULL,
  start_ts BIGINT NOT NULL,
  end_ts BIGINT NOT NULL,
  entry_url_id BIGINT NOT NULL,
  exit_url_id BIGINT NOT NULL,
  page_count INT NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS "{{website_id}}_session_state" (
  ip_id BIGINT NOT NULL,
  ua_id BIGINT NOT NULL,
  session_id BIGINT NOT NULL,
  last_ts BIGINT NOT NULL,
  PRIMARY KEY (ip_id, ua_id)
);

CREATE TABLE IF NOT EXISTS "{{website_id}}_agg_session_daily" (
  day DATE PRIMARY KEY,
  sessions BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS "{{website_id}}_agg_entry_daily" (
  day DATE NOT NULL,
  entry_url_id BIGINT NOT NULL,
  count BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (day, entry_url_id)
);

-- Indexes (create on the partitioned parent; partitions inherit)
CREATE INDEX IF NOT EXISTS "idx_{{website_id}}_timestamp"
  ON "{{website_id}}_nginx_logs"(timestamp);

CREATE INDEX IF NOT EXISTS "idx_{{website_id}}_pv_ts_ip"
  ON "{{website_id}}_nginx_logs"(timestamp, ip_id)
  WHERE pageview_flag = 1;

CREATE INDEX IF NOT EXISTS "idx_{{website_id}}_session_key"
  ON "{{website_id}}_nginx_logs"(ip_id, ua_id, timestamp)
  WHERE pageview_flag = 1;

CREATE INDEX IF NOT EXISTS "idx_{{website_id}}_ip_location"
  ON "{{website_id}}_nginx_logs"(ip_id, location_id);

CREATE INDEX IF NOT EXISTS "idx_{{website_id}}_location_ip"
  ON "{{website_id}}_nginx_logs"(location_id, ip_id);

CREATE INDEX IF NOT EXISTS "idx_{{website_id}}_sessions_start"
  ON "{{website_id}}_sessions"(start_ts);

CREATE INDEX IF NOT EXISTS "idx_{{website_id}}_sessions_start_entry"
  ON "{{website_id}}_sessions"(start_ts, entry_url_id);

CREATE INDEX IF NOT EXISTS "idx_{{website_id}}_sessions_key"
  ON "{{website_id}}_sessions"(ip_id, ua_id, end_ts);

CREATE INDEX IF NOT EXISTS "idx_{{website_id}}_sessions_ip_loc"
  ON "{{website_id}}_sessions"(ip_id, location_id);
