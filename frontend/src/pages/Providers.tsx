import { useEffect, useState } from 'react'
import { api } from '../services/api'
import type { Provider } from '../services/types'
import { Page } from './Page'
import { ExpandableTable, type Column } from '../components/ExpandableTable'
import { useLanguage } from '../services/i18n'

export function Providers() {
  const { t } = useLanguage()
  const [providers, setProviders] = useState<Provider[]>([])
  const [name, setName] = useState('')
  const [endpoint, setEndpoint] = useState('')
  const [apiKeyEnv, setApiKeyEnv] = useState('')

  async function reload() { setProviders(await api.listProviders()) }
  useEffect(() => { void reload() }, [])

  async function addProvider() {
    if (!name || !endpoint) return
    const created = await api.addProvider({ name, endpoint, apiKeyEnv, clients: ['claude'], models: ['default'], enabled: true })
    setProviders((items) => [...items.filter((item) => item.name !== created.name), created])
    setName(''); setEndpoint(''); setApiKeyEnv('')
  }

  async function toggle(provider: Provider) {
    const updated = await api.toggleProvider(provider.name, !provider.enabled)
    setProviders((items) => items.map((item) => item.name === provider.name ? updated : item))
  }

  async function remove(provider: Provider) {
    await api.removeProvider(provider.name)
    setProviders((items) => items.filter((item) => item.name !== provider.name))
  }

  const columns: Column<Provider>[] = [
    { header: 'Name', cell: (p) => <strong>{p.name}</strong> },
    { header: 'Endpoint', cell: (p) => <code>{p.endpoint}</code> },
    { header: 'Models', cell: (p) => p.models.join(', ') || '—' },
    { header: 'Status', cell: (p) => p.enabled ? 'Enabled' : 'Disabled' },
    { header: 'Actions', className: 'row-actions', cell: (p) => (
      <div className="row-actions">
        <button onClick={() => toggle(p)}>{p.enabled ? 'Disable' : 'Enable'}</button>
        <button onClick={() => remove(p)}>Remove</button>
      </div>
    ) },
  ]

  return <Page title="Providers" description="Manage providers.json entries, models, API key env vars, and enablement.">
    <section className="card"><h2>Add Provider</h2>
      <div className="inline-form">
        <input aria-label="Provider name" placeholder="name" value={name} onChange={(event) => setName(event.target.value)} />
        <input aria-label="Provider endpoint" placeholder="https://host/v1" value={endpoint} onChange={(event) => setEndpoint(event.target.value)} />
        <input aria-label={t('providers.apiKeyEnv')} placeholder="OPENAI_API_KEY" value={apiKeyEnv} onChange={(event) => setApiKeyEnv(event.target.value)} title={t('providers.apiKeyEnvHint')} />
        <button onClick={addProvider}>Add provider</button>
      </div>
    </section>
    <ExpandableTable
      ariaLabel="Providers"
      columns={columns}
      rows={providers}
      rowKey={(p) => p.name}
      renderExpanded={(p) => (
        <dl className="row-meta">
          <div><dt>{t('providers.description')}</dt><dd>{p.description || '—'}</dd></div>
          <div><dt>{t('providers.apiKeyEnv')}</dt><dd>{p.apiKeyEnv ? <code>{p.apiKeyEnv}</code> : '—'}</dd></div>
          <div><dt>{t('providers.maskedKey')}</dt><dd>{p.maskedApiKey ? <code>{p.maskedApiKey}</code> : '—'}</dd></div>
          <div><dt>{t('providers.clients')}</dt><dd>{p.clients.join(', ') || '—'}</dd></div>
        </dl>
      )}
    />
  </Page>
}
