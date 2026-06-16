import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Settings } from './Settings'

describe('Settings page', () => {
  it('shows app metadata', () => {
    render(<Settings />)
    expect(screen.getByRole('heading', { name: /settings/i })).toBeInTheDocument()
    expect(screen.getByText(/shares providers/i)).toBeInTheDocument()
  })
})
