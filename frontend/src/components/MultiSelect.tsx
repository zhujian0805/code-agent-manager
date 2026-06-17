import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { ChevronDown } from 'lucide-react'

export type MultiSelectOption = {
  value: string
  label: string
  installed?: boolean
}

type MultiSelectProps = {
  options: MultiSelectOption[]
  value: string[]
  onChange: (value: string[]) => void
  placeholder: string
  triggerAriaLabel: string
  listboxAriaLabel: string
  disabled?: boolean
}

// MultiSelect is a multi-value dropdown. The floating panel uses position:fixed
// (sized from the trigger's bounding rect) so it overlays surrounding content
// instead of being clipped by the table's overflow:hidden, and it flips above
// the trigger when there isn't room below. Selection persists across toggles;
// the panel only closes on outside-click, Escape, or the trigger button.
export function MultiSelect({ options, value, onChange, placeholder, triggerAriaLabel, listboxAriaLabel, disabled }: MultiSelectProps) {
  const [open, setOpen] = useState(false)
  const [panelRect, setPanelRect] = useState<{ top?: number; bottom?: number; left: number; width: number } | null>(null)
  const rootRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)

  const placePanel = useCallback(() => {
    const trigger = triggerRef.current
    if (!trigger) return
    const rect = trigger.getBoundingClientRect()
    const panelHeight = Math.min(options.length * 36 + 18, 280)
    const spaceBelow = window.innerHeight - rect.bottom
    const openUp = spaceBelow < panelHeight + 8 && rect.top > panelHeight + 8
    setPanelRect({
      left: rect.left,
      width: Math.max(rect.width, 220),
      top: openUp ? undefined : rect.bottom + 6,
      bottom: openUp ? window.innerHeight - rect.top + 6 : undefined,
    })
  }, [options.length])

  useLayoutEffect(() => {
    if (!open) return
    placePanel()
    const onPointerDown = (event: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) setOpen(false)
    }
    const onKey = (event: KeyboardEvent) => { if (event.key === 'Escape') setOpen(false) }
    window.addEventListener('scroll', placePanel, true)
    window.addEventListener('resize', placePanel)
    document.addEventListener('mousedown', onPointerDown)
    document.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('scroll', placePanel, true)
      window.removeEventListener('resize', placePanel)
      document.removeEventListener('mousedown', onPointerDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open, placePanel])

  function toggle(optionValue: string) {
    onChange(value.includes(optionValue) ? value.filter((v) => v !== optionValue) : [...value, optionValue])
  }

  const count = value.length
  const label = count > 0 ? `${placeholder} (${count})` : placeholder

  return (
    <div className="multiselect" ref={rootRef}>
      <button
        ref={triggerRef}
        type="button"
        className={`multiselect-trigger${open ? ' open' : ''}`}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={triggerAriaLabel}
        disabled={disabled}
        onClick={() => setOpen((current) => !current)}
      >
        <span className="multiselect-trigger-label">{label}</span>
        <ChevronDown size={14} className={`multiselect-caret${open ? ' flipped' : ''}`} aria-hidden="true" />
      </button>
      {open && panelRect && (
        <div
          className="multiselect-panel"
          role="listbox"
          aria-label={listboxAriaLabel}
          aria-multiselectable="true"
          style={{ position: 'fixed', top: panelRect.top, bottom: panelRect.bottom, left: panelRect.left, width: panelRect.width }}
        >
          {options.map((option) => {
            const checked = value.includes(option.value)
            return (
              <label key={option.value} className={`multiselect-option${checked ? ' checked' : ''}${option.installed ? ' installed' : ''}`}>
                <input type="checkbox" checked={checked} onChange={() => toggle(option.value)} aria-label={option.label} />
                <span className="multiselect-option-label">{option.label}</span>
                {option.installed && <span className="multiselect-installed" aria-hidden="true">✓</span>}
              </label>
            )
          })}
        </div>
      )}
    </div>
  )
}
