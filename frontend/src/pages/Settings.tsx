import { Page } from './Page'

export function Settings() {
  return <Page title="Settings" description="Configure desktop preferences and view application metadata.">
    <section className="card"><h2>Appearance</h2><p>Theme follows system preferences. A persistent theme toggle can be wired to AppService.SetTheme.</p></section>
    <section className="card"><h2>About</h2><p>code-agent-manager desktop shares providers, MCP servers, config, and library stores with the CLI.</p></section>
  </Page>
}
