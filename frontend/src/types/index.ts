export interface Monitor {
  id: string
  name: string
  url: string
  method: string
  interval_s: number
  timeout_ms: number
  status: 'active' | 'paused' | 'error'
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
  api_key: string
}

export interface WSCheckResult {
  monitor_id: string
  success: boolean
  status_code: number
  latency_ms: number
  error_msg?: string
  checked_at: number
}
