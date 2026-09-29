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

/**
 * GA/T 1400.4 资源对象控制面端点契约（BFF 透传到 /VIID/<Collection>）。
 *
 * - GET    /api/control/resources                       → { kinds: ResourceKindMeta[] }
 * - GET    /api/control/resources/:kind/list            → 信封 <Kind>List.<Kind>Object[]
 * - GET    /api/control/resources/:kind/list/:id        → 信封 <Kind>: {<obj>}
 * - POST   /api/control/resources/:kind/list            → 信封 {ResponseStatus, ItemCount}
 * - PUT    /api/control/resources/:kind/list/:id        → 信封 {ResponseStatus}
 * - DELETE /api/control/resources/:kind/list/:id        → 信封 {ResponseStatus}
 * - GET    /api/control/resources/:kind/list/:id/info   → Info 子资源
 *
 * BFF 不解析协议端响应，资源对象视为不透明 payload。
 */
export interface ResourceKindMeta {
  kind: string
  collection: string
  idField: string
  description: string
  count: number
}

/** 资源对象条目；具体字段按 Kind 区分，且可能含第三方扩展字段。 */
export interface ResourceObject {
  [key: string]: unknown
}

export interface ResourceListEnvelope {
  ResponseStatus?: Record<string, unknown>
  [listKey: string]: unknown
}

export interface ResourceSingleEnvelope {
  ResponseStatus?: Record<string, unknown>
  [kindKey: string]: unknown
}

export interface ResourceCreateResponse {
  ResponseStatus?: Record<string, unknown>
  ItemCount?: number
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
  getConfig: () => request<Record<string, unknown>>('/config'),

  // Resources（GA/T 1400.4 资源对象控制面）
  listResourceKinds: () =>
    request<{ kinds: ResourceKindMeta[] }>('/resources'),
  listResources: (kind: string) =>
    request<ResourceListEnvelope>(`/resources/${encodeURIComponent(kind)}/list`),
  getResource: (kind: string, id: string) =>
    request<ResourceSingleEnvelope>(
      `/resources/${encodeURIComponent(kind)}/list/${encodeURIComponent(id)}`,
    ),
  createResource: (kind: string, body: unknown) =>
    request<ResourceCreateResponse>(
      `/resources/${encodeURIComponent(kind)}/list`,
      { method: 'POST', body: JSON.stringify(body) },
    ),
  updateResource: (kind: string, id: string, body: unknown) =>
    request<ResourceCreateResponse>(
      `/resources/${encodeURIComponent(kind)}/list/${encodeURIComponent(id)}`,
      { method: 'PUT', body: JSON.stringify(body) },
    ),
  deleteResource: (kind: string, id: string) =>
    request<ResourceCreateResponse>(
      `/resources/${encodeURIComponent(kind)}/list/${encodeURIComponent(id)}`,
      { method: 'DELETE' },
    ),
  getResourceInfo: (kind: string, id: string) =>
    request<ResourceSingleEnvelope>(
      `/resources/${encodeURIComponent(kind)}/list/${encodeURIComponent(id)}/info`,
    ),
}