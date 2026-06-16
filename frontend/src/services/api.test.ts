import { describe, expect, it } from 'vitest'
import { api } from './api'

describe('api mock fallback', () => {
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
