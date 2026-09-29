/**
 * WebSocket 客户端，带自动重连（指数退避）
 */

export type WsEventKind = 'capture_added' | 'node_status' | 'scenario_state'

export interface WsEvent {
  kind: WsEventKind
  payload: unknown
  ts: string
}

type Listener = (event: WsEvent) => void

export class ControlWs {
  private socket: WebSocket | null = null
  private url: string
  private retries = 0
  private maxRetries = 3
  private listeners: Set<Listener> = new Set()
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private explicitlyClosed = false
  public status: 'connecting' | 'open' | 'closed' = 'closed'

  constructor(path = '/ws/events') {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    this.url = `${proto}//${window.location.host}${path}`
  }

  start(): void {
    this.explicitlyClosed = false
    this.connect()
  }

  stop(): void {
    this.explicitlyClosed = true
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.socket) {
      this.socket.close()
      this.socket = null
    }
    this.status = 'closed'
  }

  on(listener: Listener): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private connect(): void {
    if (this.explicitlyClosed) return
    this.status = 'connecting'
    try {
      this.socket = new WebSocket(this.url)
    } catch (err) {
      console.warn('WS construct failed', err)
      this.scheduleReconnect()
      return
    }

    this.socket.onopen = () => {
      this.retries = 0
      this.status = 'open'
    }

    this.socket.onmessage = (msg) => {
      try {
        const event = JSON.parse(msg.data) as WsEvent
        this.listeners.forEach((l) => l(event))
      } catch (err) {
        console.warn('WS message parse failed', err)
      }
    }

    this.socket.onclose = () => {
      this.status = 'closed'
      this.socket = null
      this.scheduleReconnect()
    }

    this.socket.onerror = () => {
      this.socket?.close()
    }
  }

  private scheduleReconnect(): void {
    if (this.explicitlyClosed) return
    if (this.retries >= this.maxRetries) return
    const delay = Math.min(1000 * 2 ** this.retries, 8000)
    this.retries += 1
    this.reconnectTimer = setTimeout(() => this.connect(), delay)
  }
}

export const controlWs = new ControlWs()