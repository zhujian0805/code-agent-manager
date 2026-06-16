import { useState } from 'react'
import { Activity, Boxes, FileCog, HeartPulse, Home, Plug, Settings as SettingsIcon, Server } from 'lucide-react'
import { Dashboard } from './pages/Dashboard'
import { Providers } from './pages/Providers'
import { MCP } from './pages/MCP'
import { Library } from './pages/Library'
import { Configuration } from './pages/Configuration'
import { Diagnostics } from './pages/Diagnostics'
import { Settings } from './pages/Settings'

type Route = 'dashboard' | 'providers' | 'mcp' | 'library' | 'config' | 'diagnostics' | 'settings'

const nav: { route: Route; label: string; icon: typeof Home }[] = [
  { route: 'dashboard', label: 'Launch', icon: Home },
  { route: 'providers', label: 'Providers', icon: Server },
  { route: 'mcp', label: 'MCP Servers', icon: Plug },
  { route: 'library', label: 'Library', icon: Boxes },
  { route: 'config', label: 'Configuration', icon: FileCog },
  { route: 'diagnostics', label: 'Diagnostics', icon: HeartPulse },
  { route: 'settings', label: 'Settings', icon: SettingsIcon },
]

export function App() {
  const [route, setRoute] = useState<Route>('dashboard')
  const [lastDryRun, setLastDryRun] = useState('')

  return <div className="app-shell">
    <aside className="sidebar"><div className="brand"><Activity /> <span>CAM Desktop</span></div><nav>{nav.map((item) => { const Icon = item.icon; return <button key={item.route} className={route === item.route ? 'active' : ''} onClick={() => setRoute(item.route)}><Icon size={18} />{item.label}</button> })}</nav></aside>
    <div className="content">
      {lastDryRun ? <div role="status" className="toast">Dry-run: <code>{lastDryRun}</code></div> : null}
      {route === 'dashboard' && <Dashboard onDryRun={setLastDryRun} />}
      {route === 'providers' && <Providers />}
      {route === 'mcp' && <MCP />}
      {route === 'library' && <Library />}
      {route === 'config' && <Configuration />}
      {route === 'diagnostics' && <Diagnostics />}
      {route === 'settings' && <Settings />}
    </div>
  </div>
}

export default App
