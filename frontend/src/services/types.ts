export type Provider = {
  name: string
  endpoint: string
  apiKeyEnv: string
  supportedClient: string
  clients: string[]
  models: string[]
  keepProxyConfig: boolean
  useProxy: boolean
  enabled: boolean
  description: string
  maskedApiKey?: string
}

export type Tool = {
  name: string
  command: string
  description: string
  enabled: boolean
  installed: boolean
  version: string
}

export type MCPClient = {
  name: string
  userPath: string
  projectPath?: string
  container: string
  format: string
  supportsUser: boolean
  supportsProject: boolean
}

export type MCPServer = {
  name: string
  client?: string
  scope?: string
  command?: string
  args?: string[]
  url?: string
  type?: string
}

export type Entity = {
  kind: 'prompt' | 'skill' | 'agent' | 'plugin'
  name: string
  description?: string
  content?: string
  path?: string
  apps?: string[]
  tags?: string[]
  updatedAt?: string
}

export type DoctorCheck = {
  name: string
  issues: number
  messages: { level: 'header' | 'info' | 'pass' | 'warn' | 'fail'; text: string; hint?: string }[]
}

export type ConfigFile = {
  app: string
  scope: string
  path: string
  format: string
  description?: string
  exists: boolean
}

export type LaunchPlan = {
  tool: Tool
  provider: Provider
  model: string
  command: string
  args: string[]
  environment: Record<string, string>
}

export type MetadataItem = {
  kind: string
  name: string
  description: string
  install_key: string
  repo_owner: string
  repo_name: string
  repo_branch: string
  target_apps: string
  installed_apps?: string[]
  installed: boolean
}

export type MetadataSearchResponse = {
  items: MetadataItem[]
  total: number
  limit: number
  offset: number
}

export type MetadataDetail = {
  item: MetadataItem
  content: string
  manifest_path: string
}

export type MetadataRefreshSummary = {
  sources_scanned: number
  items_added: number
  items_updated: number
  items_stale: number
  failed_sources: string[]
}
