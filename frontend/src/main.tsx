import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'

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
    <BrowserRouter>
      <AuthProvider>
        <App />
      </AuthProvider>
    </BrowserRouter>
  </StrictMode>,
)
