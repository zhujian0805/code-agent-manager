import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Dashboard } from './Dashboard'

describe('Dashboard', () => {
  it('shows coding agent CLI commands without launching agents', async () => {
    render(<Dashboard />)

    expect(await screen.findByRole('heading', { name: /launch/i })).toBeInTheDocument()
    expect(await screen.findByText('claude --allow-dangerously-skip-permissions --dangerously-skip-permissions')).toBeInTheDocument()
    expect(await screen.findByText('codex --yolo')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /launch/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /dry-run/i })).not.toBeInTheDocument()
  })
})
