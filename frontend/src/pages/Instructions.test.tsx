import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { describe, expect, it, vi, afterEach } from 'vitest'
import { Instructions } from './Instructions'
import { api } from '../services/api'
import type { Instruction, InstructionInstall, InstructionTarget } from '../services/types'

const targets: InstructionTarget[] = [
  { app: 'claude', supports: { user: true, project: true } },
  { app: 'copilot', supports: { user: false, project: true } },
]

function instruction(overrides: Partial<Instruction>): Instruction {
  return { id: 1, name: 'Instruction01', description: 'first', content: '# hi', installs: [], ...overrides }
}

function install(overrides: Partial<InstructionInstall>): InstructionInstall {
  return { id: 10, app: 'claude', level: 'user', project_dir: '', target_path: '/home/u/.claude/CLAUDE.md', link_kind: 'symlink', ...overrides }
}

describe('Instructions page', () => {
  afterEach(() => vi.restoreAllMocks())

  function stubTargets() {
    vi.spyOn(api, 'instructionTargets').mockResolvedValue(targets)
  }

  it('renders the empty state, then shows a created instruction', async () => {
    stubTargets()
    const listSpy = vi.spyOn(api, 'listInstructions').mockResolvedValueOnce([])
    vi.spyOn(api, 'createInstruction').mockResolvedValue(instruction({}))
    render(<Instructions />)

    expect(await screen.findByText(/no instructions yet/i)).toBeInTheDocument()

    listSpy.mockResolvedValue([instruction({})])
    fireEvent.click(screen.getByRole('button', { name: /new instruction/i }))
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'Instruction01' } })
    fireEvent.click(screen.getByRole('button', { name: /^save$/i }))

    expect(await screen.findByText('Instruction01')).toBeInTheDocument()
  })

  it('validates the name field inline before saving', async () => {
    stubTargets()
    vi.spyOn(api, 'listInstructions').mockResolvedValue([])
    const createSpy = vi.spyOn(api, 'createInstruction')
    render(<Instructions />)
    await screen.findByText(/no instructions yet/i)

    fireEvent.click(screen.getByRole('button', { name: /new instruction/i }))
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'bad name' } })
    fireEvent.click(screen.getByRole('button', { name: /^save$/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/only letters, numbers/i)
    expect(createSpy).not.toHaveBeenCalled()
  })

  it('installs via the popover and shows a chip, then uninstalls it', async () => {
    stubTargets()
    const listSpy = vi.spyOn(api, 'listInstructions').mockResolvedValue([instruction({})])
    vi.spyOn(api, 'installInstruction').mockResolvedValue(install({}))
    render(<Instructions />)
    await screen.findByText('Instruction01')

    fireEvent.click(screen.getByRole('button', { name: /install ▾/i }))
    const dialog = await screen.findByRole('dialog', { name: /install instruction01/i })

    listSpy.mockResolvedValue([instruction({ installs: [install({})] })])
    fireEvent.click(within(dialog).getByRole('button', { name: /^install$/i }))

    const chip = await screen.findByText(/claude \(user\)/i)
    expect(chip).toBeInTheDocument()

    // Uninstall via the chip ×.
    vi.spyOn(api, 'uninstallInstruction').mockResolvedValue(undefined)
    listSpy.mockResolvedValue([instruction({ installs: [] })])
    fireEvent.click(screen.getByRole('button', { name: /uninstall claude/i }))
    await waitFor(() => expect(screen.queryByText(/claude \(user\)/i)).not.toBeInTheDocument())
  })

  it('requires a project directory for project-level installs', async () => {
    stubTargets()
    vi.spyOn(api, 'listInstructions').mockResolvedValue([instruction({})])
    const installSpy = vi.spyOn(api, 'installInstruction')
    render(<Instructions />)
    await screen.findByText('Instruction01')

    fireEvent.click(screen.getByRole('button', { name: /install ▾/i }))
    const dialog = await screen.findByRole('dialog', { name: /install instruction01/i })
    fireEvent.click(within(dialog).getByLabelText(/project/i))
    fireEvent.click(within(dialog).getByRole('button', { name: /^install$/i }))

    expect(await within(dialog).findByRole('alert')).toHaveTextContent(/project directory is required/i)
    expect(installSpy).not.toHaveBeenCalled()
  })

  it('keeps the popover open and shows the conflict on a 409', async () => {
    stubTargets()
    vi.spyOn(api, 'listInstructions').mockResolvedValue([instruction({})])
    vi.spyOn(api, 'installInstruction').mockRejectedValue(new Error(JSON.stringify({ error: 'file already exists at /home/u/.claude/CLAUDE.md; remove it and retry' })))
    render(<Instructions />)
    await screen.findByText('Instruction01')

    fireEvent.click(screen.getByRole('button', { name: /install ▾/i }))
    const dialog = await screen.findByRole('dialog', { name: /install instruction01/i })
    fireEvent.click(within(dialog).getByRole('button', { name: /^install$/i }))

    expect(await within(dialog).findByRole('alert')).toHaveTextContent(/file already exists/i)
    // Popover stays open: the agent selector is still present.
    expect(within(dialog).getByLabelText(/agent/i)).toBeInTheDocument()
  })
})
