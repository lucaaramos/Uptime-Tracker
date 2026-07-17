import { useState, useEffect, useCallback } from 'react'
import { useParams, Link } from 'react-router-dom'
import LatencyChart from '../components/LatencyChart'
import MonitorForm from '../components/MonitorForm'
import * as api from '../services/api'
import { useWebSocket } from '../hooks/useWebSocket'
import type { MonitorResponse, CheckResult, Stats, MonitorRequest, WSCheckResult } from '../types'

export default function MonitorDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [monitor, setMonitor] = useState<MonitorResponse | null>(null)
  const [history, setHistory] = useState<CheckResult[]>([])
  const [stats, setStats] = useState<Stats | null>(null)
  const [loading, setLoading] = useState(true)
  const [period, setPeriod] = useState('24h')
  const [editing, setEditing] = useState(false)

  const fetchData = useCallback(async () => {
    if (!id) return
    try {
      const [mon, hist, st] = await Promise.all([
        api.getMonitor(id),
        api.getMonitorHistory(id, 100),
        api.getMonitorStats(id, period),
      ])
      setMonitor(mon)
      setHistory(hist)
      setStats(st)
    } catch {
      // handled
    } finally {
      setLoading(false)
    }
  }, [id, period])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  useWebSocket(
    useCallback(
      (result: WSCheckResult) => {
        if (result.monitor_id !== id) return
        setMonitor((prev) => {
          if (!prev) return prev
          return {
            ...prev,
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
        })
        setHistory((prev) => {
          const newResult: CheckResult = {
            id: '',
            monitor_id: result.monitor_id,
            status_code: result.status_code,
            latency_ms: result.latency_ms,
            success: result.success,
            error_msg: result.error_msg,
            checked_at: new Date(result.checked_at).toISOString(),
          }
          return [newResult, ...prev].slice(0, 100)
        })
      },
      [id]
    )
  )

  const handleUpdate = async (req: MonitorRequest) => {
    if (!id) return
    await api.updateMonitor(id, req)
    setEditing(false)
    await fetchData()
  }

  if (loading) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-8 text-center text-gray-400">
        Loading...
      </div>
    )
  }

  if (!monitor) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-8 text-center text-gray-400">
        Monitor not found.
      </div>
    )
  }

  const { monitor: m, last_result: last, uptime_24h: uptime } = monitor

  const periodOptions = [
    { label: '24h', value: '24h' },
    { label: '7d', value: '7d' },
    { label: '30d', value: '30d' },
  ]

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="mb-6">
        <Link to="/" className="text-sm text-indigo-600 hover:text-indigo-800">
          &larr; Back to Dashboard
        </Link>
      </div>

      <div className="bg-white rounded-lg shadow-sm border p-6 mb-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">{m.name}</h1>
            <p className="text-sm text-gray-500 mt-1">{m.url}</p>
          </div>
          <div className="flex items-center space-x-3">
            {last && (
              <span
                className={`px-3 py-1 rounded-full text-sm font-medium ${
                  last.success
                    ? 'bg-green-100 text-green-800'
                    : 'bg-red-100 text-red-800'
                }`}
              >
                {last.success ? 'UP' : 'DOWN'} ({last.status_code})
              </span>
            )}
            <button
              onClick={() => setEditing(!editing)}
              className="px-3 py-1.5 text-sm font-medium text-indigo-600 bg-indigo-50 rounded-md hover:bg-indigo-100"
            >
              Edit
            </button>
          </div>
        </div>

        <div className="grid grid-cols-4 gap-4 text-sm">
          <div>
            <span className="text-gray-400">Method</span>
            <p className="font-medium">{m.method}</p>
          </div>
          <div>
            <span className="text-gray-400">Interval</span>
            <p className="font-medium">{m.interval_s}s</p>
          </div>
          <div>
            <span className="text-gray-400">Timeout</span>
            <p className="font-medium">{m.timeout_ms}ms</p>
          </div>
          <div>
            <span className="text-gray-400">Status</span>
            <p className="font-medium capitalize">{m.status}</p>
          </div>
        </div>
      </div>

      {editing && (
        <div className="bg-white rounded-lg shadow-sm border p-6 mb-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">
            Edit Monitor
          </h2>
          <MonitorForm
            initial={{
              name: m.name,
              url: m.url,
              method: m.method,
              interval_s: m.interval_s,
              timeout_ms: m.timeout_ms,
              headers: m.headers,
              alert_email: m.alert_email,
              alert_webhook: m.alert_webhook,
            }}
            onSubmit={handleUpdate}
            onCancel={() => setEditing(false)}
          />
        </div>
      )}

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-white rounded-lg shadow-sm border p-4">
          <div className="text-xs text-gray-400">Latency</div>
          <div className="text-lg font-bold text-gray-900">
            {last ? `${last.latency_ms}ms` : '--'}
          </div>
        </div>
        <div className="bg-white rounded-lg shadow-sm border p-4">
          <div className="text-xs text-gray-400">Uptime (24h)</div>
          <div className="text-lg font-bold text-gray-900">
            {uptime !== undefined ? `${uptime.toFixed(1)}%` : '--'}
          </div>
        </div>
        <div className="bg-white rounded-lg shadow-sm border p-4">
          <div className="text-xs text-gray-400">Avg Latency</div>
          <div className="text-lg font-bold text-gray-900">
            {stats ? `${stats.avg_latency_ms.toFixed(0)}ms` : '--'}
          </div>
        </div>
        <div className="bg-white rounded-lg shadow-sm border p-4">
          <div className="text-xs text-gray-400">P99 Latency</div>
          <div className="text-lg font-bold text-gray-900">
            {stats ? `${stats.p99_latency_ms.toFixed(0)}ms` : '--'}
          </div>
        </div>
      </div>

      <div className="bg-white rounded-lg shadow-sm border p-6 mb-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-gray-900">Latency</h2>
          <div className="flex space-x-2">
            {periodOptions.map((opt) => (
              <button
                key={opt.value}
                onClick={() => setPeriod(opt.value)}
                className={`px-3 py-1 text-xs font-medium rounded-md ${
                  period === opt.value
                    ? 'bg-indigo-100 text-indigo-700'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>
        <LatencyChart data={history} />
      </div>

      {stats && (
        <div className="bg-white rounded-lg shadow-sm border p-6 mb-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Statistics</h2>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            <div>
              <div className="text-xs text-gray-400">Total Checks</div>
              <div className="text-sm font-medium">{stats.total_checks}</div>
            </div>
            <div>
              <div className="text-xs text-gray-400">Successful</div>
              <div className="text-sm font-medium text-green-600">
                {stats.success_checks}
              </div>
            </div>
            <div>
              <div className="text-xs text-gray-400">Failed</div>
              <div className="text-sm font-medium text-red-600">
                {stats.failure_checks}
              </div>
            </div>
          </div>
        </div>
      )}

      <div className="bg-white rounded-lg shadow-sm border p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">
          History ({history.length})
        </h2>
        {history.length === 0 ? (
          <div className="text-center py-8 text-gray-400">
            No checks recorded yet.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-gray-400 border-b">
                  <th className="pb-2 font-medium">Time</th>
                  <th className="pb-2 font-medium">Status</th>
                  <th className="pb-2 font-medium">Latency</th>
                  <th className="pb-2 font-medium">Error</th>
                </tr>
              </thead>
              <tbody>
                {history.map((r, i) => (
                  <tr key={i} className="border-b border-gray-50">
                    <td className="py-2 text-gray-600">
                      {new Date(r.checked_at).toLocaleString()}
                    </td>
                    <td className="py-2">
                      <span
                        className={`px-2 py-0.5 rounded text-xs font-medium ${
                          r.success
                            ? 'bg-green-100 text-green-700'
                            : 'bg-red-100 text-red-700'
                        }`}
                      >
                        {r.status_code}
                      </span>
                    </td>
                    <td className="py-2 text-gray-600">{r.latency_ms}ms</td>
                    <td className="py-2 text-red-500 max-w-xs truncate">
                      {r.error_msg || '-'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
