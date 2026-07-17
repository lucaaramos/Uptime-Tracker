import { useEffect, useRef, useCallback } from 'react'
import type { WSCheckResult } from '../types'

type WSCallback = (result: WSCheckResult) => void

export function useWebSocket(onResult: WSCallback) {
  const wsRef = useRef<WebSocket | null>(null)
  const onResultRef = useRef(onResult)
  onResultRef.current = onResult

  const connect = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws`
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
      console.log('WebSocket disconnected, reconnecting in 3s...')
      setTimeout(connect, 3000)
    }

    ws.onerror = (err) => {
      console.error('WebSocket error:', err)
      ws.close()
    }

    wsRef.current = ws
  }, [])

  useEffect(() => {
    connect()
    return () => {
      wsRef.current?.close()
    }
  }, [connect])

  return wsRef
}
