import { useCallback, useEffect, useMemo, useState } from 'react'
import { api } from '../services/api'
import type { Instruction, InstructionInstall, InstructionTarget } from '../services/types'
import { Page } from './Page'
import { ExpandableTable, type Column } from '../components/ExpandableTable'
import { MultiSelect } from '../components/MultiSelect'
import { useLanguage } from '../services/i18n'

// extractError pulls the human message out of a sidecar error. The request
// helper throws `new Error(responseBodyText)`, and the body is JSON of the form
// {"error": "..."} — so we parse it back to a readable string.
function extractError(err: unknown): string {
  const raw = err instanceof Error ? err.message : String(err)
  try {
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed.error === 'string') return parsed.error
  } catch {
    // not JSON; fall through to the raw text
  }
  return raw
}

const NAME_PATTERN = /^[A-Za-z0-9._-]+$/

export function Instructions() {
  const { t } = useLanguage()
  const [items, setItems] = useState<Instruction[]>([])
  const [targets, setTargets] = useState<InstructionTarget[]>([])
  const [query, setQuery] = useState('')
  const [status, setStatus] = useState('')
  const [copyNotice, setCopyNotice] = useState(false)
  const [editing, setEditing] = useState<Instruction | null>(null)
  const [creating, setCreating] = useState(false)

  const reload = useCallback(async () => {
    try {
      const list = await api.listInstructions()
      setItems(list ?? [])
    } catch (err) {
      setStatus(extractError(err))
    }
  }, [])

  useEffect(() => { void reload() }, [reload])
  useEffect(() => { void api.instructionTargets().then(setTargets).catch(() => setTargets([])) }, [])

  const filtered = query.trim()
    ? items.filter((it) => `${it.name} ${it.description}`.toLowerCase().includes(query.trim().toLowerCase()))
    : items

  async function onUninstall(installId: number) {
    try {
      await api.uninstallInstruction(installId)
      await reload()
    } catch (err) {
      setStatus(extractError(err))
    }
  }

  function noteCopyFallback(install: InstructionInstall) {
    if (install.link_kind === 'copy' && !copyNotice) setCopyNotice(true)
  }

  async function onDelete(instruction: Instruction) {
    // eslint-disable-next-line no-alert
    if (typeof window !== 'undefined' && !window.confirm(t('instructions.confirmDelete'))) return
    try {
      await api.deleteInstruction(instruction.id)
      await reload()
    } catch (err) {
      setStatus(extractError(err))
    }
  }

  const columns: Column<Instruction>[] = [
    { header: t('instructions.colName'), cell: (it) => <span className="row-name">{it.name}</span> },
    { header: t('instructions.colDescription'), cell: (it) => <span>{it.description || t('library.noDescription')}</span> },
    { header: t('instructions.colInstalled'), cell: (it) => (
      <div className="badges" aria-label={t('instructions.colInstalled')}>
        {(it.installs ?? []).length === 0
          ? <span className="badge badge-not-installed">{t('instructions.notInstalled')}</span>
          : (it.installs ?? []).map((ins) => <InstalledChip key={ins.id} install={ins} onUninstall={onUninstall} />)}
      </div>
    ) },
    { header: t('instructions.colActions'), cell: (it) => (
      <RowActions
        instruction={it}
        targets={targets}
        onEdit={() => setEditing(it)}
        onDelete={() => onDelete(it)}
        onInstalled={async (install) => { noteCopyFallback(install); await reload() }}
        onError={setStatus}
      />
    ) },
  ]

  return <Page title={t('instructions.title')} description={t('instructions.description')}>
    <div className="inline-form">
      <input aria-label={t('instructions.searchPlaceholder')} value={query} onChange={(e) => setQuery(e.target.value)} placeholder={t('instructions.searchPlaceholder')} />
      <button className="primary" onClick={() => setCreating(true)}>{t('instructions.new')}</button>
    </div>
    {copyNotice && <p className="status-line" role="status">{t('instructions.copyFallbackBanner')}</p>}
    {status && <p className="status-line error-text" role="alert">{status}</p>}
    <ExpandableTable
      ariaLabel={t('instructions.title')}
      columns={columns}
      rows={filtered}
      rowKey={(it) => String(it.id)}
      empty={<p>{t('instructions.empty')}</p>}
      renderExpanded={(it) => (
        <div className="detail-panel">
          <pre className="detail-content">{it.content || t('library.noDescription')}</pre>
          {(it.installs ?? []).length > 0 && (
            <ul className="install-list">
              {(it.installs ?? []).map((ins) => (
                <li key={ins.id}>{ins.app} ({ins.level}) → <code>{ins.target_path}</code>{ins.link_kind === 'copy' ? ` [${t('instructions.copyBadge')}]` : ''}</li>
              ))}
            </ul>
          )}
        </div>
      )}
    />
    {(creating || editing) && (
      <EditorModal
        instruction={editing}
        existingNames={items.map((it) => it.name)}
        onClose={() => { setCreating(false); setEditing(null) }}
        onSaved={async () => { setCreating(false); setEditing(null); await reload() }}
        onError={setStatus}
      />
    )}
  </Page>
}

type InstalledChipProps = { install: InstructionInstall; onUninstall: (id: number) => void }

function InstalledChip({ install, onUninstall }: InstalledChipProps) {
  const { t } = useLanguage()
  const isCopy = install.link_kind === 'copy'
  return (
    <span className={`badge badge-installed${isCopy ? ' badge-copy' : ''}`} title={isCopy ? t('instructions.copyTooltip') : install.target_path}>
      {install.app} ({install.level}){isCopy ? ` · ${t('instructions.copyBadge')}` : ''}
      <button type="button" className="chip-remove" aria-label={t('instructions.uninstall', { app: install.app, level: install.level })} onClick={() => onUninstall(install.id)}>×</button>
    </span>
  )
}

type RowActionsProps = {
  instruction: Instruction
  targets: InstructionTarget[]
  onEdit: () => void
  onDelete: () => void
  onInstalled: (install: InstructionInstall) => Promise<void>
  onError: (msg: string) => void
}

function RowActions({ instruction, targets, onEdit, onDelete, onInstalled }: RowActionsProps) {
  const { t } = useLanguage()
  const [open, setOpen] = useState(false)

  return (
    <div className="row-actions">
      <button onClick={onEdit}>{t('instructions.edit')}</button>
      <button onClick={() => setOpen((v) => !v)} aria-expanded={open}>{t('instructions.install')} ▾</button>
      <button className="danger" onClick={onDelete}>{t('instructions.delete')}</button>
      {open && (
        <InstallPopover
          instruction={instruction}
          targets={targets}
          onClose={() => setOpen(false)}
          onInstalled={async (install) => { setOpen(false); await onInstalled(install) }}
        />
      )}
    </div>
  )
}

type InstallPopoverProps = {
  instruction: Instruction
  targets: InstructionTarget[]
  onClose: () => void
  onInstalled: (install: InstructionInstall) => Promise<void>
}

function InstallPopover({ instruction, targets, onInstalled, onClose }: InstallPopoverProps) {
  const { t } = useLanguage()
  const [apps, setApps] = useState<string[]>(() => targets.length > 0 ? [targets[0].app] : ['claude'])
  const [level, setLevel] = useState<'user' | 'project'>('user')
  const [projectDir, setProjectDir] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const selectedTargets = useMemo(() => targets.filter((tg) => apps.includes(tg.app)), [targets, apps])

  const supportsUser = selectedTargets.length > 0 && selectedTargets.every((tg) => tg.supports.user)
  const supportsProject = selectedTargets.length > 0 && selectedTargets.every((tg) => tg.supports.project)

  // When no selected app supports the current level, snap to the other one.
  useEffect(() => {
    if (selectedTargets.length === 0) return
    if (level === 'user' && !supportsUser && supportsProject) setLevel('project')
    if (level === 'project' && !supportsProject && supportsUser) setLevel('user')
  }, [apps, level, supportsUser, supportsProject, selectedTargets.length])

  async function submit() {
    setError('')
    if (level === 'project' && !projectDir.trim()) {
      setError(t('instructions.projectDirRequired'))
      return
    }
    if (apps.length === 0) {
      setError(t('instructions.noAgentSelected'))
      return
    }
    setBusy(true)
    try {
      const installable = selectedTargets.filter((tg) => {
        return level === 'user' ? tg.supports.user : tg.supports.project
      })
      if (installable.length === 0) {
        setError(t('instructions.noSupportedAgent'))
        setBusy(false)
        return
      }
      for (const tg of installable) {
        const install = await api.installInstruction(instruction.id, { app: tg.app, level, project_dir: level === 'project' ? projectDir : undefined })
        await onInstalled(install)
      }
    } catch (err) {
      setError(extractError(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="install-popover" role="dialog" aria-label={t('instructions.installTitle', { name: instruction.name })}>
      <label>
        {t('instructions.agents')}
        <MultiSelect
          options={targets.map((tg) => ({ value: tg.app, label: tg.app }))}
          value={apps}
          onChange={setApps}
          placeholder={t('instructions.agents')}
          triggerAriaLabel={t('instructions.agents')}
          listboxAriaLabel={t('instructions.agents')}
        />
      </label>
      <fieldset className="level-radios">
        <legend>{t('instructions.level')}</legend>
        <label>
          <input type="radio" name={`level-${instruction.id}`} value="user" checked={level === 'user'} disabled={!supportsUser} onChange={() => setLevel('user')} />
          {t('instructions.levelUser')}
        </label>
        <label>
          <input type="radio" name={`level-${instruction.id}`} value="project" checked={level === 'project'} disabled={!supportsProject} onChange={() => setLevel('project')} />
          {t('instructions.levelProject')}
        </label>
      </fieldset>
      {level === 'project' && (
        <label>
          {t('instructions.projectDir')}
          <input aria-label={t('instructions.projectDir')} value={projectDir} onChange={(e) => setProjectDir(e.target.value)} placeholder="/path/to/project" />
        </label>
      )}
      {error && <p className="error-text" role="alert">{error}</p>}
      <div className="popover-actions">
        <button className="primary" onClick={submit} disabled={busy || apps.length === 0}>{busy ? t('instructions.installing') : t('instructions.installButton')}</button>
        <button onClick={onClose}>{t('instructions.cancel')}</button>
      </div>
    </div>
  )
}

type EditorModalProps = {
  instruction: Instruction | null
  existingNames: string[]
  onClose: () => void
  onSaved: () => Promise<void>
  onError: (msg: string) => void
}

function EditorModal({ instruction, existingNames, onClose, onSaved }: EditorModalProps) {
  const { t } = useLanguage()
  const [name, setName] = useState(instruction?.name ?? '')
  const [description, setDescription] = useState(instruction?.description ?? '')
  const [content, setContent] = useState(instruction?.content ?? '')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  function validate(): boolean {
    if (!NAME_PATTERN.test(name)) {
      setError(t('instructions.nameInvalid'))
      return false
    }
    const taken = existingNames.some((n) => n === name && n !== instruction?.name)
    if (taken) {
      setError(t('instructions.nameTaken', { name }))
      return false
    }
    return true
  }

  async function save() {
    setError('')
    if (!validate()) return
    setBusy(true)
    try {
      if (instruction) await api.updateInstruction(instruction.id, { name, description, content })
      else await api.createInstruction({ name, description, content })
      await onSaved()
    } catch (err) {
      setError(extractError(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" role="dialog" aria-label={instruction ? t('instructions.editTitle') : t('instructions.newTitle')} onClick={(e) => e.stopPropagation()}>
        <h2>{instruction ? t('instructions.editTitle') : t('instructions.newTitle')}</h2>
        <label>
          {t('instructions.name')}
          <input aria-label={t('instructions.name')} value={name} onChange={(e) => { setName(e.target.value); setError('') }} />
        </label>
        <label>
          {t('instructions.descriptionLabel')}
          <input aria-label={t('instructions.descriptionLabel')} value={description} onChange={(e) => setDescription(e.target.value)} />
        </label>
        <label>
          {t('instructions.content')}
          <textarea aria-label={t('instructions.content')} rows={20} value={content} onChange={(e) => setContent(e.target.value)} />
        </label>
        {error && <p className="error-text" role="alert">{error}</p>}
        <div className="modal-actions">
          <button className="primary" onClick={save} disabled={busy}>{busy ? t('instructions.saving') : t('instructions.save')}</button>
          <button onClick={onClose}>{t('instructions.cancel')}</button>
        </div>
      </div>
    </div>
  )
}
