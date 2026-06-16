import { mockConfigFiles, mockDoctorChecks, mockEntities, mockMCPClients, mockMCPServers, mockProviders, mockTools } from './mockData'
import type { ConfigFile, DoctorCheck, Entity, LaunchPlan, MCPClient, MCPServer, Provider, Tool } from './types'

type WailsAPI = {
  Providers?: {
    List?: () => Promise<Provider[]>
    Add?: (input: Partial<Provider> & { name: string }) => Promise<Provider>
    Enable?: (name: string) => Promise<Provider>
    Disable?: (name: string) => Promise<Provider>
    Remove?: (name: string) => Promise<unknown>
  }
  Tools?: { List?: () => Promise<Tool[]> }
  MCP?: { ListClients?: () => Promise<MCPClient[]>; ListInstalled?: (client: string, scope: string) => Promise<MCPServer[]> }
  Entities?: { List?: (kind: string) => Promise<Entity[]>; Search?: (kind: string, query: string) => Promise<Entity[]> }
  Config?: { ListFiles?: () => Promise<ConfigFile[]> }
  Doctor?: { RunChecks?: () => Promise<DoctorCheck[]> }
  Launch?: { DryRun?: (tool: string, provider: string, model: string, args: string[]) => Promise<LaunchPlan> }
}

declare global {
  interface Window { go?: { desktop?: WailsAPI } }
}

const wails = () => window.go?.desktop

export const api = {
  async listTools(): Promise<Tool[]> {
    return wails()?.Tools?.List?.() ?? mockTools
  },
  async listProviders(): Promise<Provider[]> {
    return wails()?.Providers?.List?.() ?? mockProviders
  },
  async addProvider(input: Partial<Provider> & { name: string }): Promise<Provider> {
    return wails()?.Providers?.Add?.(input) ?? { ...mockProviders[0], ...input, clients: input.clients ?? [], models: input.models ?? [], enabled: input.enabled ?? true, endpoint: input.endpoint ?? '', apiKeyEnv: input.apiKeyEnv ?? '', supportedClient: input.supportedClient ?? '', keepProxyConfig: input.keepProxyConfig ?? false, useProxy: input.useProxy ?? false, description: input.description ?? '' }
  },
  async toggleProvider(name: string, enabled: boolean): Promise<Provider> {
    const binding = enabled ? wails()?.Providers?.Enable : wails()?.Providers?.Disable
    return binding?.(name) ?? { ...mockProviders[0], name, enabled }
  },
  async removeProvider(name: string): Promise<void> {
    await (wails()?.Providers?.Remove?.(name) ?? Promise.resolve())
  },
  async listMCPClients(): Promise<MCPClient[]> {
    return wails()?.MCP?.ListClients?.() ?? mockMCPClients
  },
  async listMCPServers(client = 'claude', scope = 'user'): Promise<MCPServer[]> {
    return wails()?.MCP?.ListInstalled?.(client, scope) ?? mockMCPServers
  },
  async listEntities(kind: Entity['kind']): Promise<Entity[]> {
    return wails()?.Entities?.List?.(kind) ?? mockEntities.filter((entity) => entity.kind === kind)
  },
  async searchEntities(kind: Entity['kind'], query: string): Promise<Entity[]> {
    return wails()?.Entities?.Search?.(kind, query) ?? mockEntities.filter((entity) => entity.kind === kind && `${entity.name} ${entity.description}`.toLowerCase().includes(query.toLowerCase()))
  },
  async listConfigFiles(): Promise<ConfigFile[]> {
    return wails()?.Config?.ListFiles?.() ?? mockConfigFiles
  },
  async runDoctor(): Promise<DoctorCheck[]> {
    return wails()?.Doctor?.RunChecks?.() ?? mockDoctorChecks
  },
  async dryRun(tool: string, provider: string, model: string): Promise<LaunchPlan> {
    return wails()?.Launch?.DryRun?.(tool, provider, model, []) ?? { tool: mockTools[0], provider: mockProviders[0], model, command: tool, args: ['--model', model], environment: { CAM_PROVIDER: provider } }
  },
}
