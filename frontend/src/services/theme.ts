import { useCallback, useEffect, useState } from 'react'

// Theme switching ported from ~/repos/wintoolbox (MainWindow.xaml.cs):
// two named themes ("dark"/"light") selected at runtime, defaulting to dark,
// persisted across launches. Here the active theme is applied by setting a
// data-theme attribute on <html>, which swaps the CSS variable block in
// styles.css (the web analogue of swapping DarkTheme.xaml/LightTheme.xaml).

export type Theme = 'dark' | 'light'

const STORAGE_KEY = 'cam.theme'

export function getStoredTheme(): Theme {
  try {
    const value = localStorage.getItem(STORAGE_KEY)
    if (value === 'light' || value === 'dark') return value
  } catch {
    // localStorage unavailable (SSR/tests) — fall through to default.
  }
  return 'dark' // wintoolbox default when theme is Unknown
}

export function applyTheme(theme: Theme) {
  if (typeof document !== 'undefined') {
    document.documentElement.setAttribute('data-theme', theme)
  }
  try {
    localStorage.setItem(STORAGE_KEY, theme)
  } catch {
    // ignore persistence failures
  }
}

export function useTheme(): { theme: Theme; toggle: () => void } {
  const [theme, setTheme] = useState<Theme>(getStoredTheme)

  useEffect(() => { applyTheme(theme) }, [theme])

  const toggle = useCallback(() => {
    setTheme((current) => (current === 'dark' ? 'light' : 'dark'))
  }, [])

  return { theme, toggle }
}
