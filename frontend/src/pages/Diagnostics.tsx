import { useEffect, useState } from 'react'
import { api } from '../services/api'
import type { DoctorCheck } from '../services/types'
import { Page } from './Page'

export function Diagnostics() {
  const [checks, setChecks] = useState<DoctorCheck[]>([])
  async function run() { setChecks(await api.runDoctor()) }
  useEffect(() => { void run() }, [])
  return <Page title="Diagnostics" description="Run doctor checks for installation, config, auth, cache, and tool availability.">
    <button onClick={run}>Run all checks</button>
    <div className="cards">{checks.map((check) => <article className="card" key={check.name}><h2>{check.name}</h2><p>{check.issues} issue(s)</p><ul>{check.messages.map((message, index) => <li className={`message ${message.level}`} key={index}>{message.text}{message.hint ? <small> — {message.hint}</small> : null}</li>)}</ul></article>)}</div>
  </Page>
}
