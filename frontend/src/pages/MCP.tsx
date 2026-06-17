import { useEffect, useState } from 'react'
import { api } from '../services/api'
import type { MCPClient, MCPServer } from '../services/types'
import { Page } from './Page'
import { ExpandableTable, type Column } from '../components/ExpandableTable'

export function MCP() {
  const [clients, setClients] = useState<MCPClient[]>([])
  const [servers, setServers] = useState<MCPServer[]>([])
  const [client, setClient] = useState('claude')

  useEffect(() => { void api.listMCPClients().then((items) => { setClients(items); setClient(items[0]?.name ?? 'claude') }) }, [])
  useEffect(() => { void api.listMCPServers(client, 'user').then(setServers) }, [client])

  const columns: Column<MCPServer>[] = [
    { header: 'Name', cell: (s) => <strong>{s.name}</strong> },
    { header: 'Command / URL', cell: (s) => <code>{s.command || s.url || ''}</code> },
    { header: 'Type', cell: (s) => s.type ?? '—' },
  ]

  return <Page title="MCP Servers" description="Browse supported clients and installed MCP servers.">
    <label>Client<select aria-label="MCP client" value={client} onChange={(event) => setClient(event.target.value)}>{clients.map((item) => <option key={item.name} value={item.name}>{item.name}</option>)}</select></label>
    <ExpandableTable
      ariaLabel="Installed MCP servers"
      columns={columns}
      rows={servers}
      rowKey={(s) => s.name}
      empty={<p>No MCP servers installed.</p>}
      renderExpanded={(s) => (
        <dl className="row-meta">
          <div><dt>Scope</dt><dd>{s.scope ?? '—'}</dd></div>
          <div><dt>Args</dt><dd>{s.args?.length ? s.args.join(' ') : '—'}</dd></div>
        </dl>
      )}
    />
    <section className="card"><h2>Registry browser</h2><p>Use search/show operations to discover bundled server schemas and add them to clients.</p></section>
  </Page>
}
