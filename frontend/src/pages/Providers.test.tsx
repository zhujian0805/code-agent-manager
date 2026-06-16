import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { Providers } from './Providers'

describe('Providers page', () => {
  it('lists and adds providers', async () => {
    const user = userEvent.setup()
    render(<Providers />)
    expect(await screen.findByText('local')).toBeInTheDocument()

    await user.type(screen.getByLabelText(/provider name/i), 'new-provider')
    await user.type(screen.getByLabelText(/provider endpoint/i), 'http://localhost:5000/v1')
    await user.click(screen.getByRole('button', { name: /add provider/i }))

    expect(await screen.findByText('new-provider')).toBeInTheDocument()
  })
})
