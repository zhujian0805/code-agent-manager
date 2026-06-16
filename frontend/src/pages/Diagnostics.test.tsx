import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { Diagnostics } from './Diagnostics'

describe('Diagnostics page', () => {
  it('runs doctor checks', async () => {
    const user = userEvent.setup()
    render(<Diagnostics />)
    expect(await screen.findByText(/installation check/i)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /run all checks/i }))
    expect(await screen.findByText(/configuration check/i)).toBeInTheDocument()
  })
})
