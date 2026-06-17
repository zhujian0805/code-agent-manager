import { useEffect, useState } from 'react'
import { api } from '../services/api'
import type { Provider } from '../services/types'
import { Page } from './Page'

export function Providers() {
  const [providers, setProviders] = useState<Provider[]>([])
  const [name, setName] = useState('')
  const [endpoint, setEndpoint] = useState('')

  async function reload() { setProviders(await api.listProviders()) }
  useEffect(() => { void reload() }, [])

  async function addProvider() {
    if (!name || !endpoint) return
    const created = await api.addProvider({ name, endpoint, clients: ['claude'], models: ['default'], enabled: true })
    setProviders((items) => [...items.filter((item) => item.name !== created.name), created])
    setName(''); setEndpoint('')
  }

  async function toggle(provider: Provider) {
    const updated = await api.toggleProvider(provider.name, !provider.enabled)
    setProviders((items) => items.map((item) => item.name === provider.name ? updated : item))
  }

  async function remove(provider: Provider) {
    await api.removeProvider(provider.name)
    setProviders((items) => items.filter((item) => item.name !== provider.name))
  }

  return <Page title="Providers" description="Manage providers.json entries, models, API key env vars, and enablement.">
    <section className="card"><h2>Add Provider</h2><div className="inline-form"><input aria-label="Provider name" placeholder="name" value={name} onChange={(event) => setName(event.target.value)} /><input aria-label="Provider endpoint" placeholder="https://host/v1" value={endpoint} onChange={(event) => setEndpoint(event.target.value)} /><button onClick={addProvider}>Add provider</button></div></section>
    <table><thead><tr><th>Name</th><th>Endpoint</th><th>Models</th><th>Status</th><th>Actions</th></tr></thead><tbody>{providers.map((provider) => <tr key={provider.name}><td>{provider.name}</td><td>{provider.endpoint}</td><td>{provider.models.join(', ')}</td><td>{provider.enabled ? 'Enabled' : 'Disabled'}</td><td><button onClick={() => toggle(provider)}>{provider.enabled ? 'Disable' : 'Enable'}</button><button onClick={() => remove(provider)}>Remove</button></td></tr>)}</tbody></table>
  </Page>
}
