import { useAuth } from './auth/AuthContext'

/**
 * Foundation shell. It renders the real session state against the real API;
 * the sign-in forms and the task list replace its body in the next tasks.
 */
export default function App() {
  const { status, user } = useAuth()

  return (
    <div className="shell">
      <a className="skip" href="#main">
        Skip to content
      </a>

      <main className="canvas" id="main">
        <span className="mark" aria-hidden="true" />
        <h1 className="title">ShopBack To-Do</h1>
        <p className="lead">One list per account: add what needs doing, tick it off, filter what’s left.</p>

        <section className="panel" aria-labelledby="session-heading">
          <h2 className="panel-title" id="session-heading">
            Session
          </h2>
          <p className="status" aria-live="polite">
            <span className="dot" data-state={status} aria-hidden="true" />
            {status === 'loading' && 'Checking your session…'}
            {status === 'authenticated' && `Signed in as ${user?.email ?? 'your account'}`}
            {status === 'anonymous' && 'Not signed in'}
          </p>
          <p className="note">Sign-in and the task list arrive in the next step.</p>
        </section>
      </main>
    </div>
  )
}
