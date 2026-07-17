import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { login, generateApiKey } from '../services/api'

export default function LoginPage() {
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    if (!username || !password) {
      setError('Username and password required')
      return
    }

    try {
      setLoading(true)
      const res = await login({ username, password })
      localStorage.setItem('token', res.token)
      navigate('/')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  const handleApiKey = async () => {
    try {
      setLoading(true)
      const res = await generateApiKey()
      localStorage.setItem('apiKey', res.api_key)
      navigate('/')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to generate key')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-[80vh] flex items-center justify-center">
      <div className="bg-white p-8 rounded-lg shadow-sm border w-full max-w-md">
        <h1 className="text-2xl font-bold text-gray-900 mb-6 text-center">
          UptimeTracker
        </h1>

        {error && (
          <div className="bg-red-50 text-red-600 px-4 py-2 rounded-md text-sm mb-4">
            {error}
          </div>
        )}

        <form onSubmit={handleLogin} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Username
            </label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Password
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none"
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full py-2 px-4 text-sm font-medium text-white bg-indigo-600 rounded-md hover:bg-indigo-700 disabled:opacity-50"
          >
            {loading ? 'Logging in...' : 'Login'}
          </button>
        </form>

        <div className="mt-4 text-center">
          <span className="text-sm text-gray-500">or</span>
        </div>

        <button
          onClick={handleApiKey}
          disabled={loading}
          className="mt-4 w-full py-2 px-4 text-sm font-medium text-indigo-600 bg-indigo-50 border border-indigo-200 rounded-md hover:bg-indigo-100 disabled:opacity-50"
        >
          Generate API Key
        </button>
      </div>
    </div>
  )
}
