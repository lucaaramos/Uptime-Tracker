import { useState } from 'react'
import { generateApiKey } from '../services/api'

export default function SettingsPage() {
  const [apiKey, setApiKey] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [copied, setCopied] = useState(false)

  const handleGenerateKey = async () => {
    try {
      setLoading(true)
      const res = await generateApiKey()
      setApiKey(res.api_key)
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }

  const handleCopy = () => {
    if (apiKey) {
      navigator.clipboard.writeText(apiKey)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  const handleClearKey = () => {
    localStorage.removeItem('apiKey')
    setApiKey(null)
  }

  const storedApiKey = localStorage.getItem('apiKey')

  return (
    <div className="max-w-2xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <h1 className="text-2xl font-bold text-gray-900 mb-8">Settings</h1>

      <div className="space-y-6">
        <div className="bg-white rounded-lg shadow-sm border p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">
            API Keys
          </h2>

          {apiKey && (
            <div className="mb-4">
              <div className="text-xs text-gray-400 mb-1">
                Your new API key (copy it now, it won't be shown again):
              </div>
              <div className="flex items-center space-x-2">
                <code className="flex-1 bg-gray-50 border rounded px-3 py-2 text-sm font-mono break-all">
                  {apiKey}
                </code>
                <button
                  onClick={handleCopy}
                  className="px-3 py-2 text-sm font-medium text-indigo-600 bg-indigo-50 rounded-md hover:bg-indigo-100"
                >
                  {copied ? 'Copied!' : 'Copy'}
                </button>
              </div>
            </div>
          )}

          {storedApiKey && !apiKey && (
            <div className="mb-4">
              <div className="text-xs text-gray-400 mb-1">
                Active API key (hidden for security):
              </div>
              <div className="flex items-center space-x-2">
                <code className="flex-1 bg-gray-50 border rounded px-3 py-2 text-sm font-mono text-gray-400">
                  {storedApiKey.slice(0, 12)}...
                </code>
                <button
                  onClick={handleClearKey}
                  className="px-3 py-2 text-sm font-medium text-red-600 bg-red-50 rounded-md hover:bg-red-100"
                >
                  Remove
                </button>
              </div>
            </div>
          )}

          <button
            onClick={handleGenerateKey}
            disabled={loading}
            className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-md hover:bg-indigo-700 disabled:opacity-50"
          >
            {loading ? 'Generating...' : 'Generate New API Key'}
          </button>
        </div>

        <div className="bg-white rounded-lg shadow-sm border p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">
            About
          </h2>
          <p className="text-sm text-gray-600">
            UptimeTracker monitors your APIs and services, providing real-time
            status, latency charts, and alerting via email or webhooks.
          </p>
        </div>
      </div>
    </div>
  )
}
