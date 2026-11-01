import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import '@fontsource-variable/geist'
import './styles.css'

import App from './App'
import { AuthProvider } from './auth/AuthContext'

const container = document.getElementById('root')
if (!container) {
  throw new Error('index.html is missing #root')
}

createRoot(container).render(
  <StrictMode>
    <AuthProvider>
      <App />
    </AuthProvider>
  </StrictMode>,
)
