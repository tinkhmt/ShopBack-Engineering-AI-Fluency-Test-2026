import { Link, Navigate } from 'react-router-dom'

import { useAuth } from '../auth/AuthContext'
import AuthForm from '../components/AuthForm'
import SessionGate from '../components/SessionGate'

export default function Login() {
  const { status } = useAuth()

  if (status === 'loading') return <SessionGate />
  if (status === 'authenticated') return <Navigate to="/tasks" replace />

  return (
    <main className="shell">
      <div className="auth">
        <div className="brand">
          <span className="mark" aria-hidden="true" />
          ShopBack To-Do
        </div>
        <h1 className="title">Sign in</h1>
        <p className="lead">Your list is waiting. Sign in to pick up where you left off.</p>

        <div className="card">
          <AuthForm mode="login" />
        </div>

        <p className="switch">
          No account yet? <Link to="/register">Create one</Link>
        </p>
      </div>
    </main>
  )
}
