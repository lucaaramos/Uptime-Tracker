import { useState, useEffect, useCallback } from 'react'
import { getPublicStatus } from '../services/api'
import type { PublicStatusResponse, DailyUptime } from '../types'

function formatDuration(seconds?: number): string {
  if (seconds === undefined || seconds < 0) return '--'
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  return `${h}h ${m}m`
}

function barColor(day: DailyUptime): string {
  if (day.total_checks === 0) return 'bg-gray-200'
  if (day.uptime_percent >= 99) return 'bg-green-500'
  if (day.uptime_percent >= 95) return 'bg-yellow-400'
  return 'bg-red-500'
}

function overallBanner(status: PublicStatusResponse['overall_status']) {
  switch (status) {
    case 'operational':
      return {
        bg: 'bg-green-50 border-green-200 text-green-800',
        dot: 'bg-green-500',
        label: 'All Systems Operational',
      }
    case 'degraded':
      return {
        bg: 'bg-yellow-50 border-yellow-200 text-yellow-800',
        dot: 'bg-yellow-500',
        label: 'Partial System Outage',
      }
    case 'major_outage':
      return {
        bg: 'bg-red-50 border-red-200 text-red-800',
        dot: 'bg-red-500',
        label: 'Major System Outage',
      }
    default:
      return {
        bg: 'bg-gray-50 border-gray-200 text-gray-700',
        dot: 'bg-gray-400',
        label: 'No Public Services Configured',
      }
  }
}

export default function StatusPage() {
  const [data, setData] = useState<PublicStatusResponse | null>(null)
  const [days, setDays] = useState(30)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchStatus = useCallback(async () => {
    try {
      setError(null)
      const res = await getPublicStatus(days)
      setData(res)
    } catch {
      setError('Unable to load system status.')
    } finally {
      setLoading(false)
    }
  }, [days])

  useEffect(() => {
    fetchStatus()
    const timer = window.setInterval(fetchStatus, 60_000)
    return () => window.clearInterval(timer)
  }, [fetchStatus])

  const banner = data ? overallBanner(data.overall_status) : null

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
      <div className="flex flex-wrap items-center justify-between gap-4 mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">System Status</h1>
          {data && (
            <p className="text-xs text-gray-500 mt-1">
              Last updated {new Date(data.updated_at).toLocaleString()}
            </p>
          )}
        </div>
        <div className="flex items-center space-x-2">
          {[7, 30, 90].map((d) => (
            <button
              key={d}
              type="button"
              onClick={() => setDays(d)}
              className={`px-3 py-1 text-xs font-medium rounded-md ${
                days === d
                  ? 'bg-indigo-100 text-indigo-700'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
              }`}
            >
              {d}d
            </button>
          ))}
          <button
            type="button"
            onClick={fetchStatus}
            className="px-3 py-1 text-xs font-medium rounded-md bg-white border border-gray-300 text-gray-700 hover:bg-gray-50"
          >
            Refresh
          </button>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 text-red-600 border border-red-200 px-4 py-3 rounded-lg mb-6 text-sm">
          {error}
        </div>
      )}

      {loading && !data ? (
        <div className="text-center py-12 text-gray-400">Loading status...</div>
      ) : data && banner ? (
        <div className="space-y-8">
          <div className={`border rounded-lg px-5 py-4 flex items-center space-x-3 ${banner.bg}`}>
            <span className={`h-3 w-3 rounded-full ${banner.dot}`} />
            <span className="font-semibold">{banner.label}</span>
          </div>

          <section className="bg-white rounded-lg shadow-sm border divide-y divide-gray-100">
            {data.monitors.length === 0 ? (
              <div className="p-8 text-center text-sm text-gray-400">
                No public monitors have been published yet.
              </div>
            ) : (
              data.monitors.map((m) => (
                <div key={m.id} className="p-5">
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center space-x-3">
                      <span className="font-semibold text-gray-900">{m.name}</span>
                      <span
                        className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                          m.status === 'up'
                            ? 'bg-green-100 text-green-700'
                            : m.status === 'down'
                              ? 'bg-red-100 text-red-700'
                              : 'bg-gray-100 text-gray-600'
                        }`}
                      >
                        {m.status === 'up'
                          ? 'Operational'
                          : m.status === 'down'
                            ? 'Down'
                            : 'Pending'}
                      </span>
                    </div>
                    <div className="text-xs text-gray-500 space-x-3">
                      <span>{m.uptime_percent.toFixed(2)}% uptime</span>
                      {m.avg_latency_ms > 0 && (
                        <span>{Math.round(m.avg_latency_ms)}ms avg</span>
                      )}
                    </div>
                  </div>

                  <div className="flex items-center gap-[2px] h-8">
                    {m.daily.map((day) => (
                      <div
                        key={day.date}
                        title={
                          day.total_checks > 0
                            ? `${day.date}: ${day.uptime_percent.toFixed(2)}% (${day.success_checks}/${day.total_checks})`
                            : `${day.date}: No data`
                        }
                        className={`flex-1 h-full rounded-sm ${barColor(day)} hover:opacity-80 transition-opacity`}
                      />
                    ))}
                  </div>
                  <div className="flex justify-between text-[11px] text-gray-400 mt-1.5">
                    <span>{days} days ago</span>
                    <span>Today</span>
                  </div>
                </div>
              ))
            )}
          </section>

          <section className="bg-white rounded-lg shadow-sm border p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">
              Recent Incidents
            </h2>
            {data.recent_incidents.length === 0 ? (
              <p className="text-sm text-gray-400">No recent incidents.</p>
            ) : (
              <ul className="divide-y divide-gray-100">
                {data.recent_incidents.map((inc) => (
                  <li key={inc.id} className="py-3 flex flex-wrap items-center justify-between gap-2 text-sm">
                    <div>
                      <span
                        className={`inline-block mr-2 px-2 py-0.5 rounded text-xs font-medium ${
                          inc.status === 'open'
                            ? 'bg-red-100 text-red-700'
                            : 'bg-green-100 text-green-700'
                        }`}
                      >
                        {inc.status === 'open' ? 'OPEN' : 'RESOLVED'}
                      </span>
                      <span className="font-medium text-gray-900">
                        {inc.monitor_name}
                      </span>
                      <span className="text-gray-500 ml-2">{inc.cause}</span>
                    </div>
                    <div className="text-xs text-gray-400">
                      {new Date(inc.started_at).toLocaleString()}
                      {inc.status === 'resolved' && inc.duration_s !== undefined && (
                        <span> · {formatDuration(inc.duration_s)}</span>
                      )}
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </div>
      ) : null}
    </div>
  )
}
