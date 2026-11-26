import { useEffect, useRef, useCallback } from 'react'
import type { WSCheckResult } from '../types'

type WSCallback = (result: WSCheckResult) => void

export function useWebSocket(onResult: WSCallback) {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimerRef = useRef<number | null>(null)
  const unmountedRef = useRef(false)
  const onResultRef = useRef(onResult)
  onResultRef.current = onResult

  const connect = useCallback(() => {
    if (unmountedRef.current) return

    const token = localStorage.getItem('token')
    const apiKey = localStorage.getItem('apiKey')
    if (!token && !apiKey) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const params = new URLSearchParams()
    if (token) {
      params.set('token', token)
    } else if (apiKey) {
      params.set('api_key', apiKey)
    }

    const wsUrl = `${protocol}//${window.location.host}/ws?${params.toString()}`
    const ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      console.log('WebSocket connected')
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as WSCheckResult
        onResultRef.current(data)
      } catch {
        console.warn('Failed to parse WS message')
      }
    }

    ws.onclose = () => {
      if (unmountedRef.current) return
      console.log('WebSocket disconnected, reconnecting in 3s...')
      reconnectTimerRef.current = window.setTimeout(connect, 3000)
    }

    ws.onerror = (err) => {
      console.error('WebSocket error:', err)
      ws.close()
    }

    wsRef.current = ws
  }, [])

  useEffect(() => {
    unmountedRef.current = false
    connect()
    return () => {
      unmountedRef.current = true
      if (reconnectTimerRef.current !== null) {
        window.clearTimeout(reconnectTimerRef.current)
      }
      wsRef.current?.close()
    }
  }, [connect])

  return wsRef
}
