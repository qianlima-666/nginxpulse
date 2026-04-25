export interface WebsiteInfo {
  id: string;
  name: string;
}

export interface WebsitesResponse {
  websites: WebsiteInfo[];
}

export interface AppStatusResponse {
  log_parsing: boolean;
  log_parsing_website_id?: string;
  log_parsing_stage?: string;
  log_parsing_progress?: number;
  log_parsing_estimated_total_seconds?: number;
  log_parsing_estimated_remaining_seconds?: number;
  ip_geo_parsing?: boolean;
  ip_geo_pending?: boolean;
  ip_geo_progress?: number;
  ip_geo_estimated_remaining_seconds?: number;
  demo_mode?: boolean;
  access_key_expire_days?: number;
  mobile_pwa_enabled?: boolean;
  language?: string;
  version?: string;
  git_commit?: string;
  migration_required?: boolean;
  setup_required?: boolean;
  config_readonly?: boolean;
}

export interface VersionInfoResponse {
  version?: string;
  git_commit?: string;
  latest_version?: string;
  latest_release_url?: string;
  update_available?: boolean;
}

export interface SourceConfig {
  [key: string]: any;
}

export interface WhitelistConfig {
  enabled?: boolean;
  ips?: string[];
  cities?: string[];
  nonMainland?: boolean;
}

export interface WebsiteConfig {
  name: string;
  logPath?: string;
  domains?: string[];
  logType?: string;
  logFormat?: string;
  logRegex?: string;
  timeLayout?: string;
  sources?: SourceConfig[];
  whitelist?: WhitelistConfig;
  autoDiscoverHosts?: boolean;
}

export interface SystemConfig {
  logDestination?: string;
  taskInterval?: string;
  backfillMaxDurationPerRun?: string;
  backfillMaxBytesPerRun?: number;
  httpSourceTimeout?: string;
  logRetentionDays?: number;
  parseBatchSize?: number;
  ipGeoCacheLimit?: number;
  alertPush?: Record<string, any>;
  demoMode?: boolean;
  accessKeys?: string[];
  accessKeyExpireDays?: number;
  language?: string;
  webBasePath?: string;
  mobilePwaEnabled?: boolean;
  serverStatus?: ServerStatusConfig;
}

export interface ServerStatusConfig {
  enabled?: boolean;
  mockEnabled?: boolean;
  metricsUrl?: string;
  disksUrl?: string;
  timeout?: string;
  refreshInterval?: string;
}

export interface ServerConfig {
  Port?: string;
}

export interface DatabaseConfig {
  driver?: string;
  dsn?: string;
  maxOpenConns?: number;
  maxIdleConns?: number;
  connMaxLifetime?: string;
}

export interface PVFilterConfig {
  statusCodeInclude?: number[];
  excludePatterns?: string[];
  excludeIPs?: string[];
}

export interface ConfigPayload {
  system: SystemConfig;
  server: ServerConfig;
  database: DatabaseConfig;
  websites: WebsiteConfig[];
  pvFilter: PVFilterConfig;
}

export interface FieldError {
  field: string;
  message: string;
}

export interface ConfigValidationResult {
  errors: FieldError[];
  warnings: FieldError[];
}

export interface ConfigResponse {
  config: ConfigPayload;
  readonly: boolean;
  setup_required: boolean;
  default_log_path?: string;
}

export interface ConfigSaveResponse {
  success: boolean;
  restart_required?: boolean;
}

export interface ServerDiskStatus {
  name?: string;
  path?: string;
  smartctl_path?: string;
  type?: string;
  model?: string;
  serial?: string;
  firmware_version?: string;
  smartctl_exit_status?: number;
  size_bytes?: number;
  smart_available?: boolean;
  smart_enabled?: boolean;
  health_passed?: boolean;
  temperature_celsius?: number;
  percentage_used?: number;
  percentage_remaining?: number;
  media_errors?: number;
  error_log_entries?: number;
  unsafe_shutdowns?: number;
  power_on_hours?: number;
  power_cycles?: number;
  data_units_read_bytes?: number;
  data_units_written_bytes?: number;
  [key: string]: unknown;
}

export interface ServerStatusResponse {
  enabled: boolean;
  status: 'ok' | 'warning' | 'partial' | 'error' | 'disabled' | string;
  updated_at?: string;
  metrics?: Record<string, number | string | null>;
  missing_metrics?: string[];
  disk_count?: number;
  disks?: ServerDiskStatus[];
  errors?: string[];
  refresh_interval_seconds?: number;
}

export interface TimeSeriesStats {
  labels: string[];
  visitors: number[];
  pageviews: number[];
}

export interface DeveloperMetric {
  current: number;
  previous: number;
  delta: number;
  changeRate?: number | null;
  shareCurrent?: number | null;
  sharePrevious?: number | null;
}

export interface DeveloperDailySummary {
  totalRequests: number;
  avgRequestSizeBytes: DeveloperMetric;
  status5xx: DeveloperMetric;
  status4xx: DeveloperMetric;
  avgRequestTimeMs: DeveloperMetric;
  avgUpstreamTimeMs: DeveloperMetric;
  slowRequests: DeveloperMetric;
  slowRequestRate: DeveloperMetric;
}

export interface DeveloperDailyTrend {
  labels: string[];
  status4xx: number[];
  status5xx: number[];
  avgRequestTimeMs: number[];
  avgUpstreamTimeMs: number[];
  slowRequestRate: number[];
}

export interface DeveloperDailyURLIssue {
  url: string;
  requests: number;
  errors5xx: number;
  errors5xxDelta: number;
  slowRequests: number;
  avgRequestTimeMs: number;
  avgRequestTimeDeltaMs: number;
  maxRequestTimeMs: number;
}

export interface DeveloperDailyStats {
  currentDate: string;
  previousDate: string;
  slowThresholdMs: number;
  summary: DeveloperDailySummary;
  trend: DeveloperDailyTrend;
  topIssues: DeveloperDailyURLIssue[];
}

export interface SimpleSeriesStats {
  key: string[];
  uv: number[];
  uv_percent?: number[];
  pv?: number[];
  pv_percent?: number[];
}

export interface RefererIPGroupStats {
  key: string[];
  uv: number[];
  share: number[];
  domestic: string[];
  global: string[];
  total_uv: number;
}

export interface RefererIPBatchStats {
  all: RefererIPGroupStats;
  search: RefererIPGroupStats;
  direct: RefererIPGroupStats;
  external: RefererIPGroupStats;
}

export interface RealtimeSeriesItem {
  name: string;
  count: number;
  percent: number;
}

export interface RealtimeStats {
  activeCount: number;
  activeSeries: number[];
  deviceBreakdown: RealtimeSeriesItem[];
  referers: RealtimeSeriesItem[];
  pages: RealtimeSeriesItem[];
  entryPages: RealtimeSeriesItem[];
  browsers: RealtimeSeriesItem[];
  locations: RealtimeSeriesItem[];
}

export interface LogsExportStartResponse {
  job_id: string;
  status: string;
  fileName?: string;
}

export interface LogsExportJob {
  id: string;
  status: string;
  processed?: number;
  total?: number;
  fileName?: string;
  error?: string;
  created_at?: string;
  updated_at?: string;
  website_id?: string;
}

export interface LogsExportStatusResponse {
  id: string;
  status: string;
  processed?: number;
  total?: number;
  fileName?: string;
  error?: string;
  created_at?: string;
  updated_at?: string;
  website_id?: string;
}

export interface LogsExportListResponse {
  jobs: LogsExportJob[];
  total?: number;
  has_more?: boolean;
}

export interface IPGeoAPIFailure {
  id: number;
  ip: string;
  source: string;
  reason: string;
  error?: string;
  status_code?: number;
  created_at?: string;
}

export interface IPGeoAPIFailureListResponse {
  failures: IPGeoAPIFailure[];
  has_more?: boolean;
}

export interface IPGeoOverride {
  ip: string;
  domestic: string;
  global: string;
  note?: string;
  created_at?: string;
  updated_at?: string;
}

export interface IPGeoOverrideResponse {
  ip: string;
  domestic: string;
  global: string;
  source: string;
  note?: string;
  overridden: boolean;
  override?: IPGeoOverride | null;
}

export interface IPGeoOverrideMutationResponse {
  success: boolean;
  ip: string;
  domestic: string;
  global: string;
  source: string;
  note?: string;
  overridden: boolean;
  updated_websites?: number;
  affected_logs?: number;
  affected_sessions?: number;
}

export interface SystemNotification {
  id: number;
  level: string;
  category: string;
  title: string;
  message: string;
  fingerprint?: string;
  occurrences?: number;
  created_at?: string;
  last_occurred_at?: string;
  read_at?: string | null;
  metadata?: Record<string, any>;
}

export interface SystemNotificationListResponse {
  notifications: SystemNotification[];
  has_more?: boolean;
  unread_count?: number;
}

export interface AlertPushChannelResult {
  enabled?: boolean;
  success: boolean;
  error?: string;
}

export interface AlertPushTestResponse {
  success: boolean;
  tested: number;
  succeeded: number;
  results: Record<string, AlertPushChannelResult>;
}

export type ApiResponse<T> = T;
