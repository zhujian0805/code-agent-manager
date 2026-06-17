import { useEffect, useMemo, useState } from 'react'
import { api } from '../services/api'
import type { Tool } from '../services/types'
import { Page } from './Page'
import { useLanguage } from '../services/i18n'

// Run-command overrides for agents whose recommended invocation differs from the
// bare binary name (e.g. permission-skipping flags used during local dev).
const commandOverrides: Record<string, string> = {
  claude: 'claude --allow-dangerously-skip-permissions --dangerously-skip-permissions',
  codex: 'codex --yolo',
}

type AgentCommand = Tool & { runCommand: string }

// Agents lists the coding agents CAM manages, one card per agent in a responsive
// row. Each card shows the run command and whether the agent's CLI was detected
// on PATH — the "usage and details" a user needs before launching from a
// terminal. This replaces the former "Launch" page.
export function Agents() {
  const { t } = useLanguage()
  const [tools, setTools] = useState<Tool[]>([])

  useEffect(() => { void api.listTools().then(setTools) }, [])

  const commands = useMemo<AgentCommand[]>(() => tools.map((tool) => ({
    ...tool,
    runCommand: commandOverrides[tool.command] ?? tool.command,
  })), [tools])

  return <Page title={t('agents.title')} description={t('agents.description')}>
    <section className="command-board" aria-label="Coding agent commands">
      {commands.map((tool) => <article className="command-card" key={tool.name}>
        <div>
          <h2>{tool.name}</h2>
          <p>{tool.description}</p>
        </div>
        <code>{tool.runCommand}</code>
        <small>{tool.installed ? t('agents.detected', { version: tool.version }) : t('agents.notDetected')}</small>
      </article>)}
    </section>
  </Page>
}
