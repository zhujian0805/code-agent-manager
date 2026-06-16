import { mockConfigFiles, mockDoctorChecks, mockEntities, mockMCPClients, mockMCPServers, mockProviders, mockTools } from './mockData'
import type { ConfigFile, DoctorCheck, Entity, LaunchPlan, MCPClient, MCPServer, Provider, Tool } from './types'

type WailsAPI = {
  App?: { Version?: () => Promise<string>; Platform?: () => Promise<string> }
  AppService?: { Version?: () => Promise<string>; Platform?: () => Promise<string> }
  Providers?: {
    List?: () => Promise<Provider[]>
    Add?: (input: Partial<Provider> & { name: string }) => Promise<Provider>
    Enable?: (name: string) => Promise<Provider>
    Disable?: (name: string) => Promise<Provider>
    Remove?: (name: string) => Promise<unknown>
  }
  ProviderService?: WailsAPI['Providers']
  Tools?: { List?: () => Promise<Tool[]> }
  ToolService?: WailsAPI['Tools']
  MCP?: { ListClients?: () => Promise<MCPClient[]>; ListInstalled?: (client: string, scope: string) => Promise<MCPServer[]> }
  MCPService?: WailsAPI['MCP']
  Entities?: { List?: (kind: string) => Promise<Entity[]>; Search?: (kind: string, query: string) => Promise<Entity[]> }
  EntityService?: WailsAPI['Entities']
  Config?: { ListFiles?: () => Promise<ConfigFile[]> }
  ConfigService?: WailsAPI['Config']
  Doctor?: { RunChecks?: () => Promise<DoctorCheck[]> }
  DoctorService?: WailsAPI['Doctor']
  Launch?: { DryRun?: (tool: string, provider: string, model: string, args: string[]) => Promise<LaunchPlan> }
  LaunchService?: WailsAPI['Launch']
}

declare global {
  interface Window { go?: { desktop?: WailsAPI } }
}

const wails = () => window.go?.desktop
const service = <T>(shortName: keyof WailsAPI, structName: keyof WailsAPI): T | undefined => {
  const bindings = wails()
  return (bindings?.[shortName] ?? bindings?.[structName]) as T | undefined
}

export const api = {
  async listTools(): Promise<Tool[]> {
    return service<WailsAPI['Tools']>('Tools', 'ToolService')?.List?.() ?? mockTools
  },
  async listProviders(): Promise<Provider[]> {
    return service<WailsAPI['Providers']>('Providers', 'ProviderService')?.List?.() ?? mockProviders
  },
  async addProvider(input: Partial<Provider> & { name: string }): Promise<Provider> {
    return service<WailsAPI['Providers']>('Providers', 'ProviderService')?.Add?.(input) ?? { ...mockProviders[0], ...input, clients: input.clients ?? [], models: input.models ?? [], enabled: input.enabled ?? true, endpoint: input.endpoint ?? '', apiKeyEnv: input.apiKeyEnv ?? '', supportedClient: input.supportedClient ?? '', keepProxyConfig: input.keepProxyConfig ?? false, useProxy: input.useProxy ?? false, description: input.description ?? '' }
  },
  async toggleProvider(name: string, enabled: boolean): Promise<Provider> {
    const providers = service<WailsAPI['Providers']>('Providers', 'ProviderService')
    const binding = enabled ? providers?.Enable : providers?.Disable
    return binding?.(name) ?? { ...mockProviders[0], name, enabled }
  },
  async removeProvider(name: string): Promise<void> {
    await (service<WailsAPI['Providers']>('Providers', 'ProviderService')?.Remove?.(name) ?? Promise.resolve())
  },
  async listMCPClients(): Promise<MCPClient[]> {
    return service<WailsAPI['MCP']>('MCP', 'MCPService')?.ListClients?.() ?? mockMCPClients
  },
  async listMCPServers(client = 'claude', scope = 'user'): Promise<MCPServer[]> {
    return service<WailsAPI['MCP']>('MCP', 'MCPService')?.ListInstalled?.(client, scope) ?? mockMCPServers
  },
  async listEntities(kind: Entity['kind']): Promise<Entity[]> {
    return service<WailsAPI['Entities']>('Entities', 'EntityService')?.List?.(kind) ?? mockEntities.filter((entity) => entity.kind === kind)
  },
  async searchEntities(kind: Entity['kind'], query: string): Promise<Entity[]> {
    return service<WailsAPI['Entities']>('Entities', 'EntityService')?.Search?.(kind, query) ?? mockEntities.filter((entity) => entity.kind === kind && `${entity.name} ${entity.description}`.toLowerCase().includes(query.toLowerCase()))
  },
  async listConfigFiles(): Promise<ConfigFile[]> {
    return service<WailsAPI['Config']>('Config', 'ConfigService')?.ListFiles?.() ?? mockConfigFiles
  },
  async runDoctor(): Promise<DoctorCheck[]> {
    return service<WailsAPI['Doctor']>('Doctor', 'DoctorService')?.RunChecks?.() ?? mockDoctorChecks
  },
  async dryRun(tool: string, provider: string, model: string): Promise<LaunchPlan> {
    return service<WailsAPI['Launch']>('Launch', 'LaunchService')?.DryRun?.(tool, provider, model, []) ?? { tool: mockTools[0], provider: mockProviders[0], model, command: tool, args: ['--model', model], environment: { CAM_PROVIDER: provider } }
  },
}
