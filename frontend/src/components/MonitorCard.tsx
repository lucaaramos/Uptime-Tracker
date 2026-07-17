import { Link } from 'react-router-dom'
import type { MonitorResponse } from '../types'

interface Props {
  monitor: MonitorResponse
  onPause: (id: string) => void
  onResume: (id: string) => void
  onDelete: (id: string) => void
}

export default function MonitorCard({ monitor, onPause, onResume, onDelete }: Props) {
  const { monitor: m, last_result: last, uptime_24h: uptime } = monitor

  const statusColor =
    m.status === 'paused'
      ? 'bg-yellow-100 text-yellow-800'
      : last?.success
        ? 'bg-green-100 text-green-800'
        : last && !last.success
          ? 'bg-red-100 text-red-800'
          : 'bg-gray-100 text-gray-800'

  const statusDot =
    m.status === 'paused'
      ? 'bg-yellow-400'
      : last?.success
        ? 'bg-green-400'
        : last && !last.success
          ? 'bg-red-400'
          : 'bg-gray-400'

  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6 hover:shadow-md transition-shadow">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center space-x-3">
          <span className={`w-3 h-3 rounded-full ${statusDot}`} />
          <Link
            to={`/monitors/${m.id}`}
            className="text-lg font-semibold text-gray-900 hover:text-indigo-600"
          >
            {m.name}
          </Link>
        </div>
        <span
          className={`px-2 py-1 rounded-full text-xs font-medium ${statusColor}`}
        >
          {m.status === 'paused' ? 'Paused' : last?.success ? 'UP' : 'DOWN'}
        </span>
      </div>

      <div className="text-sm text-gray-500 mb-4 truncate">{m.url}</div>

      <div className="grid grid-cols-3 gap-4 mb-4">
        <div>
          <div className="text-xs text-gray-400">Latency</div>
          <div className="text-sm font-medium text-gray-900">
            {last ? `${last.latency_ms}ms` : '--'}
          </div>
        </div>
        <div>
          <div className="text-xs text-gray-400">Status</div>
          <div className="text-sm font-medium text-gray-900">
            {last ? last.status_code : '--'}
          </div>
        </div>
        <div>
          <div className="text-xs text-gray-400">Uptime 24h</div>
          <div className="text-sm font-medium text-gray-900">
            {uptime !== undefined ? `${uptime.toFixed(1)}%` : '--'}
          </div>
        </div>
      </div>

      <div className="flex justify-end space-x-2">
        <Link
          to={`/monitors/${m.id}`}
          className="px-3 py-1.5 text-xs font-medium text-indigo-600 bg-indigo-50 rounded-md hover:bg-indigo-100"
        >
          Details
        </Link>
        {m.status === 'paused' ? (
          <button
            onClick={() => onResume(m.id)}
            className="px-3 py-1.5 text-xs font-medium text-green-600 bg-green-50 rounded-md hover:bg-green-100"
          >
            Resume
          </button>
        ) : (
          <button
            onClick={() => onPause(m.id)}
            className="px-3 py-1.5 text-xs font-medium text-yellow-600 bg-yellow-50 rounded-md hover:bg-yellow-100"
          >
            Pause
          </button>
        )}
        <button
          onClick={() => onDelete(m.id)}
          className="px-3 py-1.5 text-xs font-medium text-red-600 bg-red-50 rounded-md hover:bg-red-100"
        >
          Delete
        </button>
      </div>
    </div>
  )
}
