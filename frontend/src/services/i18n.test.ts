import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { getStoredLanguage, translate } from './i18n'

// jsdom's localStorage is unreliable under vitest here (Node's experimental
// localStorage shim can leave setItem missing), so back it with a plain in-memory
// store for these tests — the same surface production code reads/writes through.
beforeEach(() => {
  const store = new Map<string, string>()
  vi.stubGlobal('localStorage', {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => { store.set(key, value) },
    removeItem: (key: string) => { store.delete(key) },
    clear: () => store.clear(),
    key: () => null,
    length: 0,
  })
})

describe('i18n service', () => {
  afterEach(() => { vi.unstubAllGlobals() })

  it('defaults to English when nothing is stored', () => {
    expect(getStoredLanguage()).toBe('en')
  })

  it('reads a persisted language', () => {
    localStorage.setItem('cam.lang', 'zh')
    expect(getStoredLanguage()).toBe('zh')
  })

  it('translates known keys per language', () => {
    expect(translate('en', 'nav.agents')).toBe('Agents')
    expect(translate('zh', 'nav.agents')).toBe('智能体')
    expect(translate('en', 'nav.subagents')).toBe('Subagents')
    expect(translate('zh', 'nav.subagents')).toBe('子智能体')
  })

  it('interpolates placeholders', () => {
    expect(translate('en', 'library.installTo', { target: 'codex' })).toBe('Install to codex')
    expect(translate('zh', 'agents.detected', { version: '1.2.3' })).toBe('已检测：1.2.3')
  })

  it('falls back to English then to the raw key for missing translations', () => {
    // A key absent from zh resolves via en.
    expect(translate('zh', 'theme.light')).toBe('浅色模式')
    // A key absent everywhere returns itself, so gaps are visible, not blank.
    expect(translate('en', 'totally.unknown.key')).toBe('totally.unknown.key')
  })
})
