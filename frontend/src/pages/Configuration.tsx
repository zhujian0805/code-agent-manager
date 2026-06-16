import { useEffect, useState } from 'react'
import { api } from '../services/api'
import type { ConfigFile } from '../services/types'
import { Page } from './Dashboard'

export function Configuration() {
  const [files, setFiles] = useState<ConfigFile[]>([])
  useEffect(() => { void api.listConfigFiles().then(setFiles) }, [])
  return <Page title="Configuration" description="Inspect CAM and editor configuration files across user and project scopes.">
    <table><thead><tr><th>App</th><th>Scope</th><th>Format</th><th>Path</th><th>Status</th></tr></thead><tbody>{files.map((file) => <tr key={`${file.app}-${file.scope}-${file.path}`}><td>{file.app}</td><td>{file.scope}</td><td>{file.format}</td><td><code>{file.path}</code></td><td>{file.exists ? 'Exists' : 'Missing'}</td></tr>)}</tbody></table>
  </Page>
}
