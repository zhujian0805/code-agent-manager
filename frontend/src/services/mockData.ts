import type { ConfigFile, DoctorCheck, Entity, MCPClient, MCPServer, Provider, Tool } from './types'

export const mockTools: Tool[] = [
  { name: 'claude-code', command: 'claude', description: 'Claude Code CLI', enabled: true, installed: true, version: 'mock' },
  { name: 'codex', command: 'codex', description: 'OpenAI Codex CLI', enabled: true, installed: false, version: 'unknown' },
]

export const mockProviders: Provider[] = [
  { name: 'local', endpoint: 'http://localhost:4000/v1', apiKeyEnv: 'LOCAL_API_KEY', supportedClient: 'claude,codex', clients: ['claude', 'codex'], models: ['gpt-4.1', 'claude-opus-4.6'], keepProxyConfig: false, useProxy: false, enabled: true, description: 'Local gateway', maskedApiKey: 'loca************_key' },
]

export const mockMCPClients: MCPClient[] = [
  { name: 'claude', userPath: '~/.claude.json', projectPath: '.claude/settings.json', container: 'mcpServers', format: 'json', supportsUser: true, supportsProject: true },
  { name: 'gemini', userPath: '~/.gemini/settings.json', projectPath: '.gemini/settings.json', container: 'mcpServers', format: 'json', supportsUser: true, supportsProject: true },
]

export const mockMCPServers: MCPServer[] = [
  { name: 'github', client: 'claude', scope: 'user', command: 'npx', args: ['-y', '@modelcontextprotocol/server-github'], type: 'stdio' },
]

export const mockEntities: Entity[] = [
  { kind: 'skill', name: 'golang-testing', description: 'Go testing guidance', apps: ['claude'], tags: ['go', 'testing'], updatedAt: '2026-06-16T00:00:00Z' },
  { kind: 'agent', name: 'code-reviewer', description: 'Review code changes', apps: ['claude'], tags: ['review'], updatedAt: '2026-06-16T00:00:00Z' },
]

export const mockConfigFiles: ConfigFile[] = [
  { app: 'cam', scope: 'user', path: '~/.config/code-agent-manager/config.yaml', format: 'yaml', exists: true },
  { app: 'claude', scope: 'project', path: '.claude/settings.json', format: 'json', exists: true },
]

export const mockDoctorChecks: DoctorCheck[] = [
  { name: 'Installation Check', issues: 0, messages: [{ level: 'pass', text: 'Code Assistant Manager installed' }] },
  { name: 'Configuration Check', issues: 0, messages: [{ level: 'pass', text: 'providers.json is valid' }] },
]
