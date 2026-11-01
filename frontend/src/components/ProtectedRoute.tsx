import { Navigate, Outlet } from 'react-router-dom'

import { useAuth } from '../auth/AuthContext'
import SessionGate from './SessionGate'

export default function ProtectedRoute() {
  const { status } = useAuth()

  // Waiting beats a flash of the login form on a page the cookie will unlock.
  if (status === 'loading') {
    return <SessionGate />
  }

  if (status === 'anonymous') {
    return <Navigate to="/login" replace />
  }

  return <Outlet />
}
