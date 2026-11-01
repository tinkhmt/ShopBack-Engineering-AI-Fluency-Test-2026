import { Link, Navigate } from 'react-router-dom'

import { useAuth } from '../auth/AuthContext'
import AuthForm from '../components/AuthForm'
import SessionGate from '../components/SessionGate'

export default function Register() {
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
        <h1 className="title">Create your account</h1>
        <p className="lead">One list per account. Start with what needs doing today.</p>

        <div className="card">
          <AuthForm mode="register" />
        </div>

        <p className="switch">
          Already have an account? <Link to="/login">Sign in</Link>
        </p>
      </div>
    </main>
  )
}
