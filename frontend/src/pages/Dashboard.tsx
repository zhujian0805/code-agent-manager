import { ReactNode, useEffect, useMemo, useState } from 'react'
import { api } from '../services/api'
import type { Tool } from '../services/types'

const commandOverrides: Record<string, string> = {
  claude: 'claude --allow-dangerously-skip-permissions --dangerously-skip-permissions',
  codex: 'codex --yolo',
}

type AgentCommand = Tool & { runCommand: string }

export function Dashboard() {
  const [tools, setTools] = useState<Tool[]>([])

  useEffect(() => { void api.listTools().then(setTools) }, [])

  const commands = useMemo<AgentCommand[]>(() => tools.map((tool) => ({
    ...tool,
    runCommand: commandOverrides[tool.command] ?? tool.command,
  })), [tools])

  return <Page title="Launch" description="CAM manages coding-agent configuration. Run agents from your terminal with these commands after configuring providers, MCP servers, prompts, skills, agents, and plugins here.">
    <section className="command-board" aria-label="Coding agent CLI commands">
      {commands.map((tool) => <article className="command-card" key={tool.name}>
        <div>
          <h2>{tool.name}</h2>
          <p>{tool.description}</p>
        </div>
        <code>{tool.runCommand}</code>
        <small>{tool.installed ? `Detected: ${tool.version}` : 'Not detected on PATH'}</small>
      </article>)}
    </section>
  </Page>
}

export function Page({ title, description, children }: { title: string; description: string; children: ReactNode }) {
  return <main className="page"><header><h1>{title}</h1><p>{description}</p></header>{children}</main>
}
