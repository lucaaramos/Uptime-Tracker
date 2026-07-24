import { useState, useCallback } from 'react'
import MonitorCard from '../components/MonitorCard'
import MonitorForm from '../components/MonitorForm'
import { useMonitors } from '../hooks/useMonitors'
import { useWebSocket } from '../hooks/useWebSocket'
import * as api from '../services/api'
import type { MonitorRequest } from '../types'

export default function DashboardPage() {
  const { monitors, loading, error, fetchMonitors, handleLiveResult } =
    useMonitors()
  const [showForm, setShowForm] = useState(false)

  useWebSocket(handleLiveResult)

  const handlePause = useCallback(async (id: string) => {
    try {
      await api.pauseMonitor(id)
      await fetchMonitors()
    } catch {
      // ignore
    }
  }, [fetchMonitors])

  const handleResume = useCallback(async (id: string) => {
    try {
      await api.resumeMonitor(id)
      await fetchMonitors()
    } catch {
      // ignore
    }
  }, [fetchMonitors])

  const handleDelete = useCallback(async (id: string) => {
    if (!confirm('Delete this monitor?')) return
    try {
      await api.deleteMonitor(id)
      await fetchMonitors()
    } catch {
      // ignore
    }
  }, [fetchMonitors])

  const handleCreate = useCallback(async (req: MonitorRequest) => {
    await api.createMonitor(req)
    setShowForm(false)
    await fetchMonitors()
  }, [fetchMonitors])

  const upMonitors = monitors.filter((m) => m.last_result?.success)
  const downMonitors = monitors.filter(
    (m) => m.last_result && !m.last_result.success
  )
  const pausedMonitors = monitors.filter((m) => m.monitor.status === 'paused')
  const unknownMonitors = monitors.filter((m) => !m.last_result && m.monitor.status !== 'paused')

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Monitors</h1>
          <p className="text-sm text-gray-500 mt-1">
            {monitors.length} monitor{monitors.length !== 1 ? 's' : ''} configured
          </p>
        </div>
        <button
          onClick={() => setShowForm(true)}
          className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-md hover:bg-indigo-700"
        >
          + New Monitor
        </button>
      </div>

      {error && (
        <div className="bg-red-50 text-red-600 px-4 py-3 rounded-md mb-6">
          {error}
        </div>
      )}

      {showForm && (
        <div className="bg-white rounded-lg shadow-sm border p-6 mb-8">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">
            Create Monitor
          </h2>
          <MonitorForm
            onSubmit={handleCreate}
            onCancel={() => setShowForm(false)}
          />
        </div>
      )}

      {loading && monitors.length === 0 ? (
        <div className="text-center py-12 text-gray-400">Loading...</div>
      ) : monitors.length === 0 ? (
        <div className="text-center py-12 text-gray-400">
          No monitors configured yet. Click "+ New Monitor" to get started.
        </div>
      ) : (
        <div className="space-y-8">
          {downMonitors.length > 0 && (
            <section>
              <h2 className="text-lg font-semibold text-red-600 mb-4">
                Down ({downMonitors.length})
              </h2>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {downMonitors.map((m) => (
                  <MonitorCard
                    key={m.monitor.id}
                    monitor={m}
                    onPause={handlePause}
                    onResume={handleResume}
                    onDelete={handleDelete}
                  />
                ))}
              </div>
            </section>
          )}

          {upMonitors.length > 0 && (
            <section>
              <h2 className="text-lg font-semibold text-green-600 mb-4">
                Operational ({upMonitors.length})
              </h2>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {upMonitors.map((m) => (
                  <MonitorCard
                    key={m.monitor.id}
                    monitor={m}
                    onPause={handlePause}
                    onResume={handleResume}
                    onDelete={handleDelete}
                  />
                ))}
              </div>
            </section>
          )}

          {unknownMonitors.length > 0 && (
            <section>
              <h2 className="text-lg font-semibold text-gray-500 mb-4">
                Pending ({unknownMonitors.length})
              </h2>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {unknownMonitors.map((m) => (
                  <MonitorCard
                    key={m.monitor.id}
                    monitor={m}
                    onPause={handlePause}
                    onResume={handleResume}
                    onDelete={handleDelete}
                  />
                ))}
              </div>
            </section>
          )}

          {pausedMonitors.length > 0 && (
            <section>
              <h2 className="text-lg font-semibold text-yellow-600 mb-4">
                Paused ({pausedMonitors.length})
              </h2>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {pausedMonitors.map((m) => (
                  <MonitorCard
                    key={m.monitor.id}
                    monitor={m}
                    onPause={handlePause}
                    onResume={handleResume}
                    onDelete={handleDelete}
                  />
                ))}
              </div>
            </section>
          )}
        </div>
      )}
    </div>
  )
}
