import {
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Area,
  AreaChart,
} from 'recharts'
import type { CheckResult } from '../types'

interface Props {
  data: CheckResult[]
  height?: number
}

export default function LatencyChart({ data, height = 300 }: Props) {
  const chartData = [...data]
    .reverse()
    .map((r) => ({
      time: new Date(r.checked_at).toLocaleTimeString(),
      latency: r.latency_ms,
      success: r.success,
    }))

  if (!chartData.length) {
    return (
      <div className="flex items-center justify-center h-60 bg-gray-50 rounded-lg text-gray-400">
        No data available
      </div>
    )
  }

  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={chartData}>
        <defs>
          <linearGradient id="latencyGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#6366f1" stopOpacity={0.3} />
            <stop offset="95%" stopColor="#6366f1" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
        <XAxis
          dataKey="time"
          tick={{ fontSize: 12, fill: '#9ca3af' }}
          interval="preserveStartEnd"
        />
        <YAxis
          tick={{ fontSize: 12, fill: '#9ca3af' }}
          label={{
            value: 'ms',
            angle: -90,
            position: 'insideLeft',
            style: { fill: '#9ca3af' },
          }}
        />
        <Tooltip />
        <Area
          type="monotone"
          dataKey="latency"
          stroke="#6366f1"
          fill="url(#latencyGradient)"
          strokeWidth={2}
          dot={false}
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}
