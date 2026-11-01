import { useAuth } from '../auth/AuthContext'
import AppHeader from '../components/AppHeader'

/** Placeholder list screen: Task 9 replaces the body with the real task list. */
export default function TasksPage() {
  const { user } = useAuth()

  return (
    <div className="shell">
      <a className="skip" href="#main">
        Skip to content
      </a>

      <AppHeader />

      <main className="canvas" id="main">
        <h1 className="title">Your tasks</h1>
        <p className="lead">Signed in as {user?.email ?? 'your account'}.</p>

        <section className="panel" aria-labelledby="list-heading">
          <h2 className="panel-title" id="list-heading">
            The list
          </h2>
          <p className="note">Adding, ticking off and filtering arrive in the next step.</p>
        </section>
      </main>
    </div>
  )
}
