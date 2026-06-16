import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from './api'

const resetWails = () => {
  delete window.go
}

describe('api mock fallback', () => {
  afterEach(resetWails)

  it('lists tools and providers', async () => {
    await expect(api.listTools()).resolves.toEqual(expect.arrayContaining([expect.objectContaining({ command: 'claude' })]))
    await expect(api.listProviders()).resolves.toEqual(expect.arrayContaining([expect.objectContaining({ name: 'local' })]))
  })

  it('creates dry-run plans', async () => {
    const plan = await api.dryRun('claude', 'local', 'model')
    expect(plan.command).toBe('claude')
    expect(plan.provider.name).toBe('local')
    expect(plan.model).toBe('model')
  })
})

describe('api Wails bindings', () => {
  afterEach(resetWails)

  it('uses short service aliases when Wails bindings exist', async () => {
    const tools = [{ name: 'claude-code', command: 'claude', description: 'Claude Code', enabled: true, installed: true }]
    const providers = [{ name: 'anthropic', endpoint: 'https://api.anthropic.com', apiKeyEnv: 'ANTHROPIC_API_KEY', supportedClient: 'claude', clients: ['claude'], models: ['claude-opus-4-8'], keepProxyConfig: false, useProxy: false, enabled: true }]
    const listTools = vi.fn().mockResolvedValue(tools)
    const listProviders = vi.fn().mockResolvedValue(providers)
    window.go = { desktop: { Tools: { List: listTools }, Providers: { List: listProviders } } }

    await expect(api.listTools()).resolves.toBe(tools)
    await expect(api.listProviders()).resolves.toBe(providers)
    expect(listTools).toHaveBeenCalledOnce()
    expect(listProviders).toHaveBeenCalledOnce()
  })

  it('uses Wails struct service names when generated bindings include them', async () => {
    const tools = [{ name: 'codex', command: 'codex', description: 'Codex', enabled: true, installed: false }]
    const providers = [{ name: 'local', endpoint: 'http://localhost:4000/v1', apiKeyEnv: 'LOCAL_KEY', supportedClient: 'codex', clients: ['codex'], models: ['gpt-4.1'], keepProxyConfig: false, useProxy: false, enabled: true }]
    const listTools = vi.fn().mockResolvedValue(tools)
    const listProviders = vi.fn().mockResolvedValue(providers)
    window.go = { desktop: { ToolService: { List: listTools }, ProviderService: { List: listProviders } } }

    await expect(api.listTools()).resolves.toBe(tools)
    await expect(api.listProviders()).resolves.toBe(providers)
    expect(listTools).toHaveBeenCalledOnce()
    expect(listProviders).toHaveBeenCalledOnce()
  })
})
