import { useState, useEffect, useCallback } from 'react'
import { generateApiKey, listApiKeys, revokeApiKey } from '../services/api'
import type { APIKeyInfo } from '../types'

export default function SettingsPage() {
  const [newKey, setNewKey] = useState<string | null>(null)
  const [keyName, setKeyName] = useState('')
  const [keys, setKeys] = useState<APIKeyInfo[]>([])
  const [loading, setLoading] = useState(false)
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchKeys = useCallback(async () => {
    try {
      const data = await listApiKeys()
      setKeys(data)
    } catch {
      // ignore if unauthenticated
    }
  }, [])

  useEffect(() => {
    fetchKeys()
  }, [fetchKeys])

  const handleGenerateKey = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    try {
      setLoading(true)
      const res = await generateApiKey(keyName.trim() || undefined)
      setNewKey(res.api_key)
      setKeyName('')
      await fetchKeys()
    } catch {
      setError('Failed to generate API key')
    } finally {
      setLoading(false)
    }
  }

  const handleRevoke = async (id: string) => {
    try {
      await revokeApiKey(id)
      setKeys((prev) => prev.filter((k) => k.id !== id))
    } catch {
      setError('Failed to revoke API key')
    }
  }

  const handleCopy = () => {
    if (newKey) {
      navigator.clipboard.writeText(newKey)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  return (
    <div className="max-w-2xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <h1 className="text-2xl font-bold text-gray-900 mb-8">Settings</h1>

      <div className="space-y-6">
        <div className="bg-white rounded-lg shadow-sm border p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">
            API Keys
          </h2>

          {error && (
            <div className="bg-red-50 text-red-600 px-3 py-2 rounded-md text-sm mb-4">
              {error}
            </div>
          )}

          {newKey && (
            <div className="mb-4 p-3 bg-indigo-50 border border-indigo-200 rounded-md">
              <div className="text-xs text-indigo-700 mb-1">
                Your new API key (copy it now, it will not be shown again):
              </div>
              <div className="flex items-center space-x-2">
                <code className="flex-1 bg-white border rounded px-3 py-2 text-sm font-mono break-all">
                  {newKey}
                </code>
                <button
                  type="button"
                  onClick={handleCopy}
                  className="px-3 py-2 text-sm font-medium text-indigo-600 bg-white border border-indigo-200 rounded-md hover:bg-indigo-100"
                >
                  {copied ? 'Copied!' : 'Copy'}
                </button>
              </div>
            </div>
          )}

          <form onSubmit={handleGenerateKey} className="flex space-x-2 mb-6">
            <input
              type="text"
              placeholder="Key name (e.g. CI/CD)"
              value={keyName}
              onChange={(e) => setKeyName(e.target.value)}
              className="flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none"
            />
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-md hover:bg-indigo-700 disabled:opacity-50"
            >
              {loading ? 'Generating...' : 'Generate Key'}
            </button>
          </form>

          {keys.length > 0 ? (
            <ul className="divide-y divide-gray-200 border-t">
              {keys.map((k) => (
                <li
                  key={k.id}
                  className="py-3 flex items-center justify-between text-sm"
                >
                  <div>
                    <div className="font-medium text-gray-900">{k.name}</div>
                    <div className="text-xs text-gray-500 font-mono">
                      {k.prefix}... · Created{' '}
                      {new Date(k.created_at).toLocaleDateString()}
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={() => handleRevoke(k.id)}
                    className="px-3 py-1 text-xs font-medium text-red-600 bg-red-50 rounded-md hover:bg-red-100"
                  >
                    Revoke
                  </button>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-sm text-gray-500">No API keys created yet.</p>
          )}
        </div>

        <div className="bg-white rounded-lg shadow-sm border p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">About</h2>
          <p className="text-sm text-gray-600">
            UptimeTracker monitors your APIs and services, providing real-time
            status, latency charts, and alerting via email or webhooks.
          </p>
        </div>
      </div>
    </div>
  )
}
