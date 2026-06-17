import { useEffect, useState } from 'react'
import { api } from '../services/api'
import type { DoctorCheck } from '../services/types'
import { Page } from './Page'
import { ExpandableTable, type Column } from '../components/ExpandableTable'

export function Diagnostics() {
  const [checks, setChecks] = useState<DoctorCheck[]>([])
  async function run() { setChecks(await api.runDoctor()) }
  useEffect(() => { void run() }, [])

  const columns: Column<DoctorCheck>[] = [
    { header: 'Check', cell: (c) => <strong>{c.name}</strong> },
    { header: 'Issues', cell: (c) => c.issues },
  ]

  return <Page title="Diagnostics" description="Run doctor checks for installation, config, auth, cache, and tool availability.">
    <button onClick={run}>Run all checks</button>
    <ExpandableTable
      ariaLabel="Doctor checks"
      columns={columns}
      rows={checks}
      rowKey={(c) => c.name}
      renderExpanded={(c) => (
        <ul>{c.messages.map((message, index) => <li className={`message ${message.level}`} key={index}>{message.text}{message.hint ? <small> — {message.hint}</small> : null}</li>)}</ul>
      )}
    />
  </Page>
}
