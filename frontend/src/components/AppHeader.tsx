import { useState } from 'react'
import { Link } from 'react-router-dom'

import { useAuth } from '../auth/AuthContext'

export default function AppHeader() {
  const { user, logout } = useAuth()
  const [pending, setPending] = useState(false)

  async function handleSignOut() {
    setPending(true)
    // logout() clears local state even if the request fails, so the protected
    // route redirects to /login on its own.
    await logout()
  }

  return (
    <header className="header">
      <div className="header-inner">
        <Link className="wordmark" to="/tasks">
          <span className="mark" aria-hidden="true" />
          ShopBack To-Do
        </Link>

        <div className="header-session">
          <span className="header-email">{user?.email}</span>
          <button className="signout" type="button" onClick={handleSignOut} disabled={pending}>
            {pending ? 'Signing out…' : 'Sign out'}
          </button>
        </div>
      </div>
    </header>
  )
}
