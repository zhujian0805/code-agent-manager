import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Library } from './Library'

describe('Library page', () => {
  it('renders a dedicated skills page and searches', async () => {
    render(<Library kind="skill" />)
    expect(await screen.findByRole('heading', { name: /skills/i })).toBeInTheDocument()
    expect(await screen.findByText(/golang-testing/i)).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText(/skills search/i), { target: { value: 'golang' } })
    fireEvent.click(screen.getByRole('button', { name: /search/i }))
    expect(await screen.findByText(/golang-testing/i)).toBeInTheDocument()
  })

  it('renders a dedicated agents page', async () => {
    render(<Library kind="agent" />)
    expect(await screen.findByRole('heading', { name: /agents/i })).toBeInTheDocument()
    expect(await screen.findByText(/code-reviewer/i)).toBeInTheDocument()
  })
})
