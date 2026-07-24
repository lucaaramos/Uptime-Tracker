import { useState } from 'react'
import type { MonitorRequest } from '../types'

interface Props {
  initial?: MonitorRequest
  onSubmit: (req: MonitorRequest) => Promise<void>
  onCancel: () => void
}

const DEFAULT_FORM: MonitorRequest = {
  name: '',
  url: '',
  method: 'GET',
  interval_s: 60,
  timeout_ms: 5000,
  alert_email: '',
  alert_webhook: '',
}

export default function MonitorForm({ initial, onSubmit, onCancel }: Props) {
  const [form, setForm] = useState<MonitorRequest>(initial ?? DEFAULT_FORM)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value } = e.target
    setForm((prev) => ({
      ...prev,
      [name]: name === 'interval_s' || name === 'timeout_ms' ? Number(value) : value,
    }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    if (!form.name || !form.url) {
      setError('Name and URL are required')
      return
    }

    const cleanedHeaders: Record<string, string> = {}
    if (form.headers) {
      for (const [k, v] of Object.entries(form.headers)) {
        if (k.trim() && v.trim()) cleanedHeaders[k.trim()] = v.trim()
      }
    }

    try {
      setSubmitting(true)
      await onSubmit({ ...form, headers: Object.keys(cleanedHeaders).length > 0 ? cleanedHeaders : undefined })
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to save monitor')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {error && (
        <div className="bg-red-50 text-red-600 px-4 py-2 rounded-md text-sm">
          {error}
        </div>
      )}

      <div>
        <label className="block text-sm font-medium text-gray-700">Name</label>
        <input
          type="text"
          name="name"
          value={form.name}
          onChange={handleChange}
          required
          className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none"
          placeholder="My API"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700">URL</label>
        <input
          type="url"
          name="url"
          value={form.url}
          onChange={handleChange}
          required
          className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none"
          placeholder="https://api.example.com/health"
        />
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div>
          <label className="block text-sm font-medium text-gray-700">Method</label>
          <select
            name="method"
            value={form.method}
            onChange={handleChange}
            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none"
          >
            <option value="GET">GET</option>
            <option value="POST">POST</option>
            <option value="HEAD">HEAD</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700">
            Interval (seconds)
          </label>
          <input
            type="number"
            name="interval_s"
            value={form.interval_s}
            onChange={handleChange}
            min={10}
            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700">
            Timeout (ms)
          </label>
          <input
            type="number"
            name="timeout_ms"
            value={form.timeout_ms}
            onChange={handleChange}
            min={500}
            max={30000}
            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none"
          />
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Custom Headers (optional)
        </label>
        {Object.entries(form.headers || {}).map(([k, v], i) => (
          <div key={i} className="flex space-x-2 mb-2">
            <input
              type="text"
              value={k}
              onChange={(e) => {
                const newHeaders = { ...form.headers }
                delete newHeaders[k]
                newHeaders[e.target.value] = v
                setForm({ ...form, headers: newHeaders })
              }}
              className="flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none"
              placeholder="Header name"
            />
            <input
              type="text"
              value={v}
              onChange={(e) => {
                const newHeaders = { ...form.headers, [k]: e.target.value }
                setForm({ ...form, headers: newHeaders })
              }}
              className="flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none"
              placeholder="Value"
            />
            <button
              type="button"
              onClick={() => {
                const newHeaders = { ...form.headers }
                delete newHeaders[k]
                setForm({ ...form, headers: newHeaders })
              }}
              className="px-2 py-1 text-sm text-red-600 hover:text-red-800"
            >
              ×
            </button>
          </div>
        ))}
        <button
          type="button"
          onClick={() =>
            setForm({
              ...form,
              headers: { ...form.headers, 'Authorization': '' },
            })
          }
          className="text-sm text-indigo-600 hover:text-indigo-800"
        >
          + Add Header
        </button>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700">
          Alert Email (optional)
        </label>
        <input
          type="email"
          name="alert_email"
          value={form.alert_email || ''}
          onChange={handleChange}
          className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none"
          placeholder="admin@example.com"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700">
          Alert Webhook (optional)
        </label>
        <input
          type="url"
          name="alert_webhook"
          value={form.alert_webhook || ''}
          onChange={handleChange}
          className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none"
          placeholder="https://hooks.slack.com/..."
        />
      </div>

      <div className="flex justify-end space-x-3 pt-4">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={submitting}
          className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-md hover:bg-indigo-700 disabled:opacity-50"
        >
          {submitting ? 'Saving...' : initial ? 'Update' : 'Create'}
        </button>
      </div>
    </form>
  )
}
