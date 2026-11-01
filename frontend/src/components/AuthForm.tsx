import { useRef, useState } from 'react'
import type { FormEvent } from 'react'

import { ApiError } from '../api/client'
import { useAuth } from '../auth/AuthContext'

type Mode = 'login' | 'register'

/**
 * Shared mechanics for both auth screens. The page owns the heading and the
 * link to the other screen; this owns the fields, the request and the errors.
 */
export default function AuthForm({ mode }: { mode: Mode }) {
  const { login, register } = useAuth()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [pending, setPending] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [fields, setFields] = useState<Record<string, string>>({})
  const emailRef = useRef<HTMLInputElement>(null)
  const passwordRef = useRef<HTMLInputElement>(null)

  const isRegister = mode === 'register'
  // Native validation catches the obvious cases; the server stays authoritative
  // and its messages render verbatim.
  const passwordHintId = isRegister ? 'password-hint' : null
  const passwordDescription = [passwordHintId, fields.password ? 'password-error' : null]
    .filter(Boolean)
    .join(' ')

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setPending(true)
    setFormError(null)
    setFields({})

    try {
      const credentials = { email, password }
      await (isRegister ? register(credentials) : login(credentials))
      // Success flips AuthContext to `authenticated`; the page guard redirects.
    } catch (error) {
      if (!(error instanceof ApiError)) {
        setFormError('Something went wrong. Try again.')
        return
      }

      // 409 carries no `fields` because the conflict is on the account, not the
      // request shape; show it on the email input, which is what must change.
      const nextFields = error.code === 'email_taken' ? { email: error.message } : error.fields
      setFields(nextFields)

      // Only repeat the message at form level when no field already carries it.
      if (Object.keys(nextFields).length === 0) {
        setFormError(error.message)
      }

      if (nextFields.email) emailRef.current?.focus()
      else if (nextFields.password) passwordRef.current?.focus()
    } finally {
      setPending(false)
    }
  }

  return (
    <form className="auth-form" onSubmit={handleSubmit}>
      {formError && (
        <p className="form-error" role="alert">
          {formError}
        </p>
      )}

      <div className="field">
        <label className="field-label" htmlFor="email">
          Email
        </label>
        <input
          ref={emailRef}
          className="input"
          id="email"
          name="email"
          type="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          autoComplete="email"
          autoCapitalize="none"
          spellCheck={false}
          required
          aria-invalid={fields.email ? true : undefined}
          aria-describedby={fields.email ? 'email-error' : undefined}
        />
        {fields.email && (
          <p className="field-error" id="email-error">
            {fields.email}
          </p>
        )}
      </div>

      <div className="field">
        <label className="field-label" htmlFor="password">
          Password
        </label>
        <input
          ref={passwordRef}
          className="input"
          id="password"
          name="password"
          type="password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          autoComplete={isRegister ? 'new-password' : 'current-password'}
          required
          minLength={isRegister ? 8 : undefined}
          aria-invalid={fields.password ? true : undefined}
          aria-describedby={passwordDescription || undefined}
        />
        {isRegister && (
          <p className="field-hint" id="password-hint">
            At least 8 characters.
          </p>
        )}
        {fields.password && (
          <p className="field-error" id="password-error">
            {fields.password}
          </p>
        )}
      </div>

      <button className="button" type="submit" disabled={pending}>
        {pending
          ? isRegister
            ? 'Creating account…'
            : 'Signing in…'
          : isRegister
            ? 'Create account'
            : 'Sign in'}
      </button>
    </form>
  )
}
