import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Configuration } from './Configuration'

describe('Configuration page', () => {
  it('lists config files', async () => {
    render(<Configuration />)
    expect(await screen.findByText(/config.yaml/i)).toBeInTheDocument()
    expect(screen.getAllByText(/claude/i).length).toBeGreaterThan(0)
  })
})
