import { useEffect, useMemo, useState } from 'react'
import { api } from '../services/api'
import type { Provider, Tool } from '../services/types'
import { Page } from './Page'
import { ExpandableTable, type Column } from '../components/ExpandableTable'
import { useLanguage } from '../services/i18n'

// Run-command overrides for agents whose recommended invocation differs from the
// bare binary name (e.g. permission-skipping flags used during local dev).
const commandOverrides: Record<string, string> = {
  claude: 'claude --allow-dangerously-skip-permissions --dangerously-skip-permissions',
  codex: 'codex --yolo',
}

type AgentCommand = Tool & { runCommand: string }

// Per-agent provider choice is a local UI preference: the Agents page documents
// how to run each agent, and the chosen provider is the one the user intends to
// point that agent at. It is persisted in localStorage so it survives reloads.
const PREF_KEY = 'cam.agentProviders'

function loadPrefs(): Record<string, string> {
  try {
    const raw = localStorage.getItem(PREF_KEY)
    if (raw) return JSON.parse(raw) as Record<string, string>
  } catch {
    // localStorage unavailable or corrupt — fall back to empty.
  }
  return {}
}

function savePrefs(prefs: Record<string, string>) {
  try { localStorage.setItem(PREF_KEY, JSON.stringify(prefs)) } catch { /* ignore */ }
}

// Agents lists the coding agents CAM manages, one row per agent in a compact
// table. Each row lets the user pick the provider to target and expands to show
// the run command and detection status — the "usage and details" a user needs
// before launching from a terminal. This replaces the former card grid.
export function Agents() {
  const { t } = useLanguage()
  const [tools, setTools] = useState<Tool[]>([])
  const [providers, setProviders] = useState<Provider[]>([])
  const [prefs, setPrefs] = useState<Record<string, string>>(() => loadPrefs())

  useEffect(() => { void api.listTools().then(setTools) }, [])
  useEffect(() => { void api.listProviders().then(setProviders) }, [])

  const commands = useMemo<AgentCommand[]>(() => tools.map((tool) => ({
    ...tool,
    runCommand: commandOverrides[tool.command] ?? tool.command,
  })), [tools])

  function selectProvider(toolName: string, providerName: string) {
    setPrefs((current) => {
      const next = { ...current, [toolName]: providerName }
      savePrefs(next)
      return next
    })
  }

  const columns: Column<AgentCommand>[] = [
    { header: t('agents.title'), cell: (tool) => <strong>{tool.name}</strong> },
    { header: t('agents.provider'), cell: (tool) => (
      <select
        aria-label={`${t('agents.provider')} ${tool.name}`}
        value={prefs[tool.name] ?? ''}
        onChange={(event) => selectProvider(tool.name, event.target.value)}
      >
        <option value="">—</option>
        {providers.map((p) => <option key={p.name} value={p.name}>{p.name}</option>)}
      </select>
    ) },
    { header: 'Status', cell: (tool) => tool.installed ? t('agents.detected', { version: tool.version }) : t('agents.notDetected') },
    { header: 'Command', cell: (tool) => <code>{tool.runCommand}</code> },
  ]

  return <Page title={t('agents.title')} description={t('agents.description')}>
    <ExpandableTable
      ariaLabel={t('agents.title')}
      columns={columns}
      rows={commands}
      rowKey={(tool) => tool.name}
      renderExpanded={(tool) => (
        <p>{tool.description}</p>
      )}
    />
  </Page>
}
