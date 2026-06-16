import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { Dashboard } from './Dashboard'

describe('Dashboard', () => {
  it('runs a dry-run and reports the generated command', async () => {
    const user = userEvent.setup()
    const onDryRun = vi.fn()
    render(<Dashboard onDryRun={onDryRun} />)

    expect(await screen.findByLabelText(/tool/i)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /dry-run/i }))
    expect(onDryRun).toHaveBeenCalledWith(expect.stringContaining('claude'))
  })
})
