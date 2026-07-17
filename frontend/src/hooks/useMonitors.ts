import { useState, useEffect, useCallback } from 'react'
import type { MonitorResponse, WSCheckResult } from '../types'
import * as api from '../services/api'

export function useMonitors() {
  const [monitors, setMonitors] = useState<MonitorResponse[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchMonitors = useCallback(async () => {
    try {
      setLoading(true)
      const data = await api.getMonitors()
      setMonitors(data)
      setError(null)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to fetch monitors')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchMonitors()
  }, [fetchMonitors])

  const handleLiveResult = useCallback((result: WSCheckResult) => {
    setMonitors((prev) =>
      prev.map((m) => {
        if (m.monitor.id === result.monitor_id) {
          return {
            ...m,
            last_result: {
              id: '',
              monitor_id: result.monitor_id,
              status_code: result.status_code,
              latency_ms: result.latency_ms,
              success: result.success,
              error_msg: result.error_msg,
              checked_at: new Date(result.checked_at).toISOString(),
            },
          }
        }
        return m
      })
    )
  }, [])

  return { monitors, loading, error, fetchMonitors, handleLiveResult }
}
