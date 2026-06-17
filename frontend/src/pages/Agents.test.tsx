import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Agents } from './Agents'

describe('Agents page', () => {
  it('shows coding agent run commands and detection status, no launch buttons', async () => {
    render(<Agents />)

    expect(await screen.findByRole('heading', { name: /agents/i })).toBeInTheDocument()
    expect(await screen.findByText('claude --allow-dangerously-skip-permissions --dangerously-skip-permissions')).toBeInTheDocument()
    expect(await screen.findByText('codex --yolo')).toBeInTheDocument()
    // It documents commands; it must not launch agents from the GUI.
    expect(screen.queryByRole('button', { name: /launch/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /dry-run/i })).not.toBeInTheDocument()
  })
})
