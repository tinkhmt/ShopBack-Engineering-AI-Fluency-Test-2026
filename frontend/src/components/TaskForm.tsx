import { useId, useRef, useState } from 'react'
import type { FormEvent, KeyboardEvent } from 'react'

import { ApiError } from '../api/client'

interface TaskFormProps {
  /** Present in edit mode; the fields start from these values. */
  initial?: { title: string; description: string }
  submitLabel: string
  onSubmit: (values: { title: string; description: string }) => Promise<void>
  onCancel?: () => void
  autoFocus?: boolean
  titleInputId?: string
}

export default function TaskForm({
  initial,
  submitLabel,
  onSubmit,
  onCancel,
  autoFocus,
  titleInputId,
}: TaskFormProps) {
  // Create and edit forms can be on screen together, so error ids must be unique.
  const errorId = useId()
  const [title, setTitle] = useState(initial?.title ?? '')
  const [description, setDescription] = useState(initial?.description ?? '')
  const [pending, setPending] = useState(false)
  const [fields, setFields] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState<string | null>(null)
  const titleRef = useRef<HTMLInputElement>(null)
  const descriptionRef = useRef<HTMLInputElement>(null)
  const isEdit = initial !== undefined

  function handleKeyDown(event: KeyboardEvent<HTMLFormElement>) {
    if (event.key === 'Escape' && onCancel) {
      event.preventDefault()
      onCancel()
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setPending(true)
    setFields({})
    setFormError(null)

    try {
      await onSubmit({ title, description })
      if (!isEdit) {
        setTitle('')
        setDescription('')
        titleRef.current?.focus()
      }
    } catch (error) {
      // The session ended: clearSession() has already sent the guard to /login.
      if (error instanceof ApiError && error.status === 401) return
      if (!(error instanceof ApiError)) {
        setFormError('Something went wrong. Try again.')
        return
      }
      setFields(error.fields)
      if (Object.keys(error.fields).length === 0) setFormError(error.message)
      if (error.fields.title) titleRef.current?.focus()
      else if (error.fields.description) descriptionRef.current?.focus()
    } finally {
      setPending(false)
    }
  }

  return (
    <form className="task-form" onSubmit={handleSubmit} onKeyDown={handleKeyDown}>
      {formError && (
        <p className="form-error" role="alert">
          {formError}
        </p>
      )}

      <div className="field">
        <input
          ref={titleRef}
          className="input"
          id={titleInputId}
          value={title}
          onChange={(event) => setTitle(event.target.value)}
          placeholder="What needs doing?"
          aria-label="Task title"
          autoComplete="off"
          maxLength={255}
          required
          autoFocus={autoFocus}
          aria-invalid={fields.title ? true : undefined}
          aria-describedby={fields.title ? `${errorId}-title` : undefined}
        />
        {fields.title && (
          <p className="field-error" id={`${errorId}-title`}>
            {fields.title}
          </p>
        )}
      </div>

      <div className="field">
        <input
          ref={descriptionRef}
          className="input"
          value={description}
          onChange={(event) => setDescription(event.target.value)}
          placeholder="Add a note (optional)"
          aria-label="Task note"
          autoComplete="off"
          maxLength={2000}
          aria-invalid={fields.description ? true : undefined}
          aria-describedby={fields.description ? `${errorId}-description` : undefined}
        />
        {fields.description && (
          <p className="field-error" id={`${errorId}-description`}>
            {fields.description}
          </p>
        )}
      </div>

      <div className="task-actions">
        <button className="button" type="submit" disabled={pending}>
          {pending ? (isEdit ? 'Saving…' : 'Adding…') : submitLabel}
        </button>
        {onCancel && (
          <button className="text-button" type="button" onClick={onCancel} disabled={pending}>
            Cancel
          </button>
        )}
      </div>
    </form>
  )
}
