import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { Library } from './Library'

describe('Library page', () => {
  it('switches entity kinds and searches', async () => {
    const user = userEvent.setup()
    render(<Library />)
    expect(await screen.findByText(/golang-testing/i)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /agents/i }))
    expect(await screen.findByText(/code-reviewer/i)).toBeInTheDocument()
    await user.type(screen.getByLabelText(/library search/i), 'code')
    await user.click(screen.getByRole('button', { name: /search/i }))
    expect(await screen.findByText(/code-reviewer/i)).toBeInTheDocument()
  })
})
