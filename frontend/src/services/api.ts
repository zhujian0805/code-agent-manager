import { mockConfigFiles, mockDoctorChecks, mockEntities, mockMCPClients, mockMCPServers, mockProviders, mockTools } from './mockData'
import type { ConfigFile, DoctorCheck, Entity, LaunchPlan, MCPClient, MCPServer, Provider, Tool } from './types'

type SidecarConfig = {
  baseUrl: string
  token?: string
}

declare global {
  interface Window { __CAM_SIDECAR__?: SidecarConfig; __TAURI_INTERNALS__?: unknown }
}

let tauriSidecar: Promise<SidecarConfig | undefined> | undefined

const configuredSidecar = (): SidecarConfig | undefined => {
  if (typeof window !== 'undefined' && window.__CAM_SIDECAR__?.baseUrl) {
    return window.__CAM_SIDECAR__
  }
  const baseUrl = import.meta.env.VITE_CAM_API_BASE_URL
  const token = import.meta.env.VITE_CAM_API_TOKEN
  return baseUrl ? { baseUrl, token } : undefined
}

const sidecar = async (): Promise<SidecarConfig | undefined> => {
  const configured = configuredSidecar()
  if (configured) return configured
  if (typeof window === 'undefined' || !('__TAURI_INTERNALS__' in window)) return undefined
  tauriSidecar ??= import('@tauri-apps/api/core')
    .then(({ invoke }) => invoke<SidecarConfig>('sidecar_config'))
    .then((config) => {
      window.__CAM_SIDECAR__ = config
      return config
    })
    .catch(() => undefined)
  return tauriSidecar
}

const normalizeBaseUrl = (baseUrl: string) => baseUrl.replace(/\/$/, '')

async function request<T>(path: string, init: RequestInit = {}): Promise<T | undefined> {
  const config = await sidecar()
  if (!config) return undefined
  const headers = new Headers(init.headers)
  if (init.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')
  if (config.token) headers.set('Authorization', `Bearer ${config.token}`)
  const response = await fetch(`${normalizeBaseUrl(config.baseUrl)}${path}`, { ...init, headers })
  if (!response.ok) throw new Error(await response.text())
  return response.json() as Promise<T>
}

export const api = {
  async listTools(): Promise<Tool[]> {
    return (await request<Tool[]>('/api/tools')) ?? mockTools
  },
  async listProviders(): Promise<Provider[]> {
    return (await request<Provider[]>('/api/providers')) ?? mockProviders
  },
  async addProvider(input: Partial<Provider> & { name: string }): Promise<Provider> {
    return (await request<Provider>('/api/providers', { method: 'POST', body: JSON.stringify(input) })) ?? { ...mockProviders[0], ...input, clients: input.clients ?? [], models: input.models ?? [], enabled: input.enabled ?? true, endpoint: input.endpoint ?? '', apiKeyEnv: input.apiKeyEnv ?? '', supportedClient: input.supportedClient ?? '', keepProxyConfig: input.keepProxyConfig ?? false, useProxy: input.useProxy ?? false, description: input.description ?? '' }
  },
  async toggleProvider(name: string, enabled: boolean): Promise<Provider> {
    return (await request<Provider>(`/api/providers/${encodeURIComponent(name)}/${enabled ? 'enable' : 'disable'}`, { method: 'POST' })) ?? { ...mockProviders[0], name, enabled }
  },
  async removeProvider(name: string): Promise<void> {
    await (request<unknown>(`/api/providers/${encodeURIComponent(name)}`, { method: 'DELETE' }) ?? Promise.resolve())
  },
  async listMCPClients(): Promise<MCPClient[]> {
    return (await request<MCPClient[]>('/api/mcp/clients')) ?? mockMCPClients
  },
  async listMCPServers(client = 'claude', scope = 'user'): Promise<MCPServer[]> {
    return (await request<MCPServer[]>(`/api/mcp/servers?client=${encodeURIComponent(client)}&scope=${encodeURIComponent(scope)}`)) ?? mockMCPServers
  },
  async listEntities(kind: Entity['kind']): Promise<Entity[]> {
    return (await request<Entity[]>(`/api/entities?kind=${encodeURIComponent(kind)}`)) ?? mockEntities.filter((entity) => entity.kind === kind)
  },
  async searchEntities(kind: Entity['kind'], query: string): Promise<Entity[]> {
    return (await request<Entity[]>(`/api/entities?kind=${encodeURIComponent(kind)}&query=${encodeURIComponent(query)}`)) ?? mockEntities.filter((entity) => entity.kind === kind && `${entity.name} ${entity.description}`.toLowerCase().includes(query.toLowerCase()))
  },
  async listConfigFiles(): Promise<ConfigFile[]> {
    return (await request<ConfigFile[]>('/api/config/files')) ?? mockConfigFiles
  },
  async runDoctor(): Promise<DoctorCheck[]> {
    return (await request<DoctorCheck[]>('/api/doctor/checks')) ?? mockDoctorChecks
  },
  async dryRun(tool: string, provider: string, model: string): Promise<LaunchPlan> {
    return (await request<LaunchPlan>('/api/launch/dry-run', { method: 'POST', body: JSON.stringify({ tool, provider, model, args: [] }) })) ?? { tool: mockTools[0], provider: mockProviders[0], model, command: tool, args: ['--model', model], environment: { CAM_PROVIDER: provider } }
  },
}
