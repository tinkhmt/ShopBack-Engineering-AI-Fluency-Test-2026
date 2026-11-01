/** Shown while the session cookie is being checked, so no screen flashes. */
export default function SessionGate() {
  return (
    <main className="gate" aria-live="polite">
      <p className="gate-text">Checking your session…</p>
    </main>
  )
}
