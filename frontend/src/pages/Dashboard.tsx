import { ReactNode, useEffect, useState } from 'react'
import { api } from '../services/api'
import type { Provider, Tool } from '../services/types'

type Props = { onDryRun: (result: string) => void }

export function Dashboard({ onDryRun }: Props) {
  const [tools, setTools] = useState<Tool[]>([])
  const [providers, setProviders] = useState<Provider[]>([])
  const [tool, setTool] = useState('')
  const [provider, setProvider] = useState('')
  const [model, setModel] = useState('')

  useEffect(() => {
    void api.listTools().then((items) => {
      setTools(items)
      setTool(items[0]?.command ?? '')
    })
    void api.listProviders().then((items) => {
      setProviders(items)
      setProvider(items[0]?.name ?? '')
      setModel(items[0]?.models[0] ?? '')
    })
  }, [])

  async function dryRun() {
    const plan = await api.dryRun(tool, provider, model)
    onDryRun(`${plan.command} ${plan.args.join(' ')}`.trim())
  }

  return <Page title="Dashboard / Launch" description="Choose a tool, provider, and model, then launch or dry-run the generated command.">
    <div className="form-grid">
      <label>Tool<select aria-label="Tool" value={tool} onChange={(event) => setTool(event.target.value)}>{tools.map((item) => <option key={item.command} value={item.command}>{item.name} ({item.command})</option>)}</select></label>
      <label>Provider<select aria-label="Provider" value={provider} onChange={(event) => setProvider(event.target.value)}>{providers.map((item) => <option key={item.name} value={item.name}>{item.name}</option>)}</select></label>
      <label>Model<input aria-label="Model" value={model} onChange={(event) => setModel(event.target.value)} /></label>
    </div>
    <div className="actions"><button onClick={dryRun}>Dry-run</button><button className="primary">Launch</button></div>
  </Page>
}

export function Page({ title, description, children }: { title: string; description: string; children: ReactNode }) {
  return <main className="page"><header><h1>{title}</h1><p>{description}</p></header>{children}</main>
}
