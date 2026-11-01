import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Same-origin in development: the browser only talks to :5173, so the session
// cookie stays first-party and the API needs no CORS middleware. nginx repeats
// this proxy in the container.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': { target: 'http://localhost:8080' },
    },
  },
  // `npm run preview` serves the production build; it needs the same proxy so
  // the last design gate sees the real session state.
  preview: {
    port: 4173,
    proxy: {
      '/api': { target: 'http://localhost:8080' },
    },
  },
})
