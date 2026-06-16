import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { MCP } from './MCP'

describe('MCP page', () => {
  it('shows clients and installed servers', async () => {
    render(<MCP />)
    expect(await screen.findByRole('heading', { name: /mcp servers/i })).toBeInTheDocument()
    expect(await screen.findByText(/github/i)).toBeInTheDocument()
  })
})
