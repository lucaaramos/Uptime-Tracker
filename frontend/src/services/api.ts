import axios from 'axios'
import type {
  Monitor,
  MonitorRequest,
  MonitorResponse,
  CheckResult,
  Stats,
  LoginRequest,
  LoginResponse,
  APIKeyResponse,
  APIKeyInfo,
  Incident,
} from '../types'

const api = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  const apiKey = localStorage.getItem('apiKey')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  } else if (apiKey) {
    config.headers['X-API-Key'] = apiKey
  }
  return config
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401 && window.location.pathname !== '/login') {
      localStorage.removeItem('token')
      localStorage.removeItem('apiKey')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export async function register(req: LoginRequest): Promise<LoginResponse> {
  const { data } = await api.post('/auth/register', req)
  return data
}

export async function login(req: LoginRequest): Promise<LoginResponse> {
  const { data } = await api.post('/auth/login', req)
  return data
}

export async function generateApiKey(name?: string): Promise<APIKeyResponse> {
  const { data } = await api.post('/auth/apikey', name ? { name } : {})
  return data
}

export async function listApiKeys(): Promise<APIKeyInfo[]> {
  const { data } = await api.get('/auth/apikeys')
  return data
}

export async function revokeApiKey(id: string): Promise<void> {
  await api.delete(`/auth/apikeys/${id}`)
}

export async function getMonitors(): Promise<MonitorResponse[]> {
  const { data } = await api.get('/monitors')
  return data
}

export async function getMonitor(id: string): Promise<MonitorResponse> {
  const { data } = await api.get(`/monitors/${id}`)
  return data
}

export async function createMonitor(req: MonitorRequest): Promise<Monitor> {
  const { data } = await api.post('/monitors', req)
  return data
}

export async function updateMonitor(id: string, req: MonitorRequest): Promise<Monitor> {
  const { data } = await api.put(`/monitors/${id}`, req)
  return data
}

export async function deleteMonitor(id: string): Promise<void> {
  await api.delete(`/monitors/${id}`)
}

export async function pauseMonitor(id: string): Promise<void> {
  await api.post(`/monitors/${id}/pause`)
}

export async function resumeMonitor(id: string): Promise<void> {
  await api.post(`/monitors/${id}/resume`)
}

export async function getMonitorHistory(
  id: string,
  limit = 50,
  offset = 0
): Promise<CheckResult[]> {
  const { data } = await api.get(`/monitors/${id}/history`, {
    params: { limit, offset },
  })
  return data
}

export async function getMonitorStats(
  id: string,
  period = '24h'
): Promise<Stats> {
  const { data } = await api.get(`/monitors/${id}/stats`, {
    params: { period },
  })
  return data
}

export async function getIncidents(
  status?: 'open' | 'resolved',
  limit = 50,
  offset = 0
): Promise<Incident[]> {
  const { data } = await api.get('/incidents', {
    params: { ...(status ? { status } : {}), limit, offset },
  })
  return data
}

export async function getMonitorIncidents(
  id: string,
  limit = 20,
  offset = 0
): Promise<Incident[]> {
  const { data } = await api.get(`/monitors/${id}/incidents`, {
    params: { limit, offset },
  })
  return data
}
