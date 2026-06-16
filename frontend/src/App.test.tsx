import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import App from './App'

describe('App shell', () => {
  it('renders launch dashboard and navigates to all primary pages', async () => {
    const user = userEvent.setup()
    render(<App />)
    expect(await screen.findByRole('heading', { name: /dashboard \/ launch/i })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /providers/i }))
    expect(await screen.findByRole('heading', { name: /providers/i })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /mcp servers/i }))
    expect(await screen.findByRole('heading', { name: /mcp servers/i })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /library/i }))
    expect(await screen.findByRole('heading', { name: /library/i })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /configuration/i }))
    expect(await screen.findByRole('heading', { name: /configuration/i })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /diagnostics/i }))
    expect(await screen.findByRole('heading', { name: /diagnostics/i })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /settings/i }))
    expect(await screen.findByRole('heading', { name: /settings/i })).toBeInTheDocument()
  })
})
