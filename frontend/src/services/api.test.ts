import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from './api'

const resetSidecar = () => {
  delete window.__CAM_SIDECAR__
  vi.restoreAllMocks()
}

describe('api mock fallback', () => {
  afterEach(resetSidecar)

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

describe('api sidecar transport', () => {
  afterEach(resetSidecar)

  it('uses sidecar base URL and bearer token for provider listing', async () => {
    const providers = [{ name: 'local', endpoint: 'http://localhost:4000/v1', apiKeyEnv: 'LOCAL_KEY', supportedClient: 'claude', clients: ['claude'], models: ['m1'], keepProxyConfig: false, useProxy: false, enabled: true, description: 'local' }]
    window.__CAM_SIDECAR__ = { baseUrl: 'http://127.0.0.1:54321/', token: 'secret' }
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify(providers), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    await expect(api.listProviders()).resolves.toEqual(providers)

    expect(fetchMock).toHaveBeenCalledOnce()
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:54321/api/providers')
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer secret')
  })

  it('posts provider input to sidecar', async () => {
    const provider = { name: 'alpha', endpoint: 'https://alpha.example', apiKeyEnv: '', supportedClient: 'claude', clients: ['claude'], models: [], keepProxyConfig: false, useProxy: false, enabled: true, description: '' }
    window.__CAM_SIDECAR__ = { baseUrl: 'http://127.0.0.1:54321' }
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify(provider), { status: 200, headers: { 'Content-Type': 'application/json' } }))

    await expect(api.addProvider({ name: 'alpha', endpoint: 'https://alpha.example' })).resolves.toEqual(provider)

    const [, init] = fetchMock.mock.calls[0]
    expect(init?.method).toBe('POST')
    expect(init?.body).toContain('alpha')
    expect(new Headers(init?.headers).get('Content-Type')).toBe('application/json')
  })
})
