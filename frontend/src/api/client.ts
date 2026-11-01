import type { ApiErrorBody } from '../types'

/**
 * ApiError mirrors the backend's single error envelope, so a screen can show
 * `message` and attach `fields` to the matching input instead of inventing its
 * own wording. `status` is 0 when the request never reached the server.
 */
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly fields: Record<string, string>

  constructor(status: number, code: string, message: string, fields: Record<string, string> = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
  }
}

type Method = 'GET' | 'POST' | 'PATCH' | 'DELETE'

export async function request<T>(method: Method, path: string, body?: unknown): Promise<T> {
  let response: Response

  try {
    response = await fetch(path, {
      method,
      // Same-origin in dev (Vite proxy) and in production (nginx). The session
      // cookie is HttpOnly, so JavaScript never reads it.
      credentials: 'same-origin',
      headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new ApiError(0, 'network_error', 'Cannot reach the server. Check your connection and try again.')
  }

  // 204 has no body; callers use request<void>.
  if (response.status === 204) {
    return undefined as T
  }

  const payload: unknown = await response.json().catch(() => null)

  if (!response.ok) {
    const error = isApiErrorBody(payload) ? payload.error : null
    throw new ApiError(
      response.status,
      error?.code ?? 'unexpected_error',
      error?.message ?? 'Something went wrong. Try again.',
      error?.fields ?? {},
    )
  }

  return payload as T
}

function isApiErrorBody(value: unknown): value is ApiErrorBody {
  return typeof value === 'object' && value !== null && 'error' in value
}
