import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { applyTheme, getStoredTheme } from './services/theme'
import './styles.css'

// Apply the persisted theme before first paint to avoid a flash of the wrong theme.
applyTheme(getStoredTheme())

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
