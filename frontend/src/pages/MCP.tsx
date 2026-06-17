import { useEffect, useState } from 'react'
import { api } from '../services/api'
import type { MCPClient, MCPServer } from '../services/types'
import { Page } from './Page'

export function MCP() {
  const [clients, setClients] = useState<MCPClient[]>([])
  const [servers, setServers] = useState<MCPServer[]>([])
  const [client, setClient] = useState('claude')

  useEffect(() => { void api.listMCPClients().then((items) => { setClients(items); setClient(items[0]?.name ?? 'claude') }) }, [])
  useEffect(() => { void api.listMCPServers(client, 'user').then(setServers) }, [client])

  return <Page title="MCP Servers" description="Browse supported clients and installed MCP servers.">
    <label>Client<select aria-label="MCP client" value={client} onChange={(event) => setClient(event.target.value)}>{clients.map((item) => <option key={item.name} value={item.name}>{item.name}</option>)}</select></label>
    <section className="card"><h2>Installed servers</h2>{servers.length === 0 ? <p>No MCP servers installed.</p> : <ul>{servers.map((server) => <li key={server.name}><strong>{server.name}</strong> {server.command || server.url}</li>)}</ul>}</section>
    <section className="card"><h2>Registry browser</h2><p>Use search/show operations to discover bundled server schemas and add them to clients.</p></section>
  </Page>
}
