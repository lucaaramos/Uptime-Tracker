export interface Monitor {
  id: string
  name: string
  url: string
  method: string
  interval_s: number
  timeout_ms: number
  status: 'active' | 'paused' | 'error'
  public?: boolean
  headers?: Record<string, string>
  alert_email?: string
  alert_webhook?: string
  created_at: string
  updated_at: string
}

export interface CheckResult {
  id: string
  monitor_id: string
  status_code: number
  latency_ms: number
  success: boolean
  error_msg?: string
  checked_at: string
}

export interface MonitorResponse {
  monitor: Monitor
  last_result?: CheckResult
  uptime_24h?: number
}

export interface Stats {
  uptime_percent: number
  avg_latency_ms: number
  p95_latency_ms: number
  p99_latency_ms: number
  total_checks: number
  success_checks: number
  failure_checks: number
}

export interface MonitorRequest {
  name: string
  url: string
  method?: string
  interval_s?: number
  timeout_ms?: number
  public?: boolean
  headers?: Record<string, string>
  alert_email?: string
  alert_webhook?: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  expires_at: number
}

export interface APIKeyResponse {
  id: string
  name: string
  prefix: string
  api_key: string
  created_at: string
}

export interface APIKeyInfo {
  id: string
  user_id: string
  name: string
  prefix: string
  created_at: string
  last_used_at?: string
}

export interface WSCheckResult {
  monitor_id: string
  success: boolean
  status_code: number
  latency_ms: number
  error_msg?: string
  checked_at: number
}

export interface Incident {
  id: string
  monitor_id: string
  monitor_name: string
  url: string
  status: 'open' | 'resolved'
  cause: string
  status_code?: number
  fail_count: number
  started_at: string
  resolved_at?: string
  duration_s?: number
}

export interface DailyUptime {
  date: string
  total_checks: number
  success_checks: number
  uptime_percent: number
}

export interface PublicMonitorStatus {
  id: string
  name: string
  status: 'up' | 'down' | 'pending'
  uptime_percent: number
  avg_latency_ms: number
  last_checked_at?: string
  daily: DailyUptime[]
}

export interface PublicIncident {
  id: string
  monitor_id: string
  monitor_name: string
  status: 'open' | 'resolved'
  cause: string
  started_at: string
  resolved_at?: string
  duration_s?: number
}

export interface PublicStatusResponse {
  overall_status: 'operational' | 'degraded' | 'major_outage' | 'no_monitors'
  updated_at: string
  days: number
  monitors: PublicMonitorStatus[]
  recent_incidents: PublicIncident[]
}
