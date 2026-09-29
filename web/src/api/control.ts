/**
 * BFF REST API 客户端
 *
 * 端点契约（与 internal/ui/handlers.go 完全一致）：
 * - GET    /api/control/system/health          → SystemHealth
 * - GET    /api/control/nodes                  → Node[]
 * - POST   /api/control/nodes                  → Node
 * - PUT    /api/control/nodes/{id}             → Node
 * - DELETE /api/control/nodes/{id}             → void
 * - GET    /api/control/scenarios              → Scenario[]
 * - POST   /api/control/scenarios/{id}/start   → Scenario
 * - POST   /api/control/scenarios/{id}/stop    → Scenario
 * - GET    /api/control/captures               → Capture[]
 * - GET    /api/control/stats                  → Stats
 * - GET    /api/control/config                 → Config (raw object)
 */

const BASE = '/api/control'

async function request<T>(
  path: string,
  init?: RequestInit
): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {})
    }
  })
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new Error(`API ${res.status} ${path}: ${text || res.statusText}`)
  }
  if (res.status === 204) {
    return undefined as T
  }
  return (await res.json()) as T
}

export interface SystemHealth {
  status: string
  uptime: number
  version: string
  nodesOnline: number
  nodesTotal: number
}

export interface Node {
  id: string
  name: string
  role: string
  listenPort: number
  upstreamURL: string
  capabilities?: string[]
  status?: 'online' | 'offline' | 'unknown'
  lastSeen?: string
}

export interface Scenario {
  id: string
  name: string
  tags?: string[]
  nodes: string[]
  subscriptionIds: string[]
  status: 'idle' | 'running' | 'stopped'
  createdAt?: string
}

export interface Capture {
  id: string
  timestamp: string
  direction: 'inbound' | 'outbound'
  method: string
  path: string
  status: number
  duration: number
  scenarioId?: string
  nodeId?: string
}

export interface Stats {
  nodes: number
  online: number
  scenarios: number
  running: number
  captures: number
}

export const controlApi = {
  // System
  getSystemHealth: () => request<SystemHealth>('/system/health'),

  // Nodes
  getNodes: () => request<Node[]>('/nodes'),
  upsertNode: (node: Partial<Node>) =>
    request<Node>(node.id ? `/nodes/${node.id}` : '/nodes', {
      method: node.id ? 'PUT' : 'POST',
      body: JSON.stringify(node)
    }),
  deleteNode: (id: string) =>
    request<void>(`/nodes/${id}`, { method: 'DELETE' }),

  // Scenarios
  getScenarios: () => request<Scenario[]>('/scenarios'),
  startScenario: (id: string) =>
    request<Scenario>(`/scenarios/${id}/start`, { method: 'POST' }),
  stopScenario: (id: string) =>
    request<Scenario>(`/scenarios/${id}/stop`, { method: 'POST' }),

  // Captures
  getCaptures: (limit = 50) =>
    request<Capture[]>(`/captures?limit=${limit}`),

  // Stats
  getStats: () => request<Stats>('/stats'),

  // Config
  getConfig: () => request<Record<string, unknown>>('/config')
}