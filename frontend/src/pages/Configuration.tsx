import { useEffect, useState } from 'react'
import { api } from '../services/api'
import type { ConfigFile } from '../services/types'
import { Page } from './Page'
import { ExpandableTable, type Column } from '../components/ExpandableTable'

export function Configuration() {
  const [files, setFiles] = useState<ConfigFile[]>([])
  useEffect(() => { void api.listConfigFiles().then(setFiles) }, [])

  const columns: Column<ConfigFile>[] = [
    { header: 'App', cell: (f) => f.app },
    { header: 'Scope', cell: (f) => f.scope },
    { header: 'Format', cell: (f) => f.format },
    { header: 'Path', cell: (f) => <code>{f.path}</code> },
    { header: 'Status', cell: (f) => f.exists ? 'Exists' : 'Missing' },
  ]

  return <Page title="Configuration" description="Inspect CAM and editor configuration files across user and project scopes.">
    <ExpandableTable
      ariaLabel="Configuration files"
      columns={columns}
      rows={files}
      rowKey={(f) => `${f.app}-${f.scope}-${f.path}`}
      renderExpanded={(f) => <p>{f.description || '—'}</p>}
    />
  </Page>
}
