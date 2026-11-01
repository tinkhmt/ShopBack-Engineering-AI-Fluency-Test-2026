import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'

import { authApi } from '../api/auth'
import { ApiError } from '../api/client'
import type { Credentials, User } from '../types'

export type AuthStatus = 'loading' | 'authenticated' | 'anonymous'

export interface AuthValue {
  status: AuthStatus
  user: User | null
  login: (credentials: Credentials) => Promise<User>
  register: (credentials: Credentials) => Promise<User>
  logout: () => Promise<void>
  /** Drop local auth state after a 401 from any other call. */
  clearSession: () => void
}

const AuthContext = createContext<AuthValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>('loading')
  const [user, setUser] = useState<User | null>(null)

  // The cookie is HttpOnly, so the only way to know whether a session exists is
  // to ask the server once. A 401 here is the normal signed-out case.
  useEffect(() => {
    let active = true

    authApi
      .me()
      .then(({ user }) => {
        if (!active) return
        setUser(user)
        setStatus('authenticated')
      })
      .catch((error: unknown) => {
        if (!active) return
        if (!(error instanceof ApiError) || error.status !== 401) {
          console.error('session check failed', error)
        }
        setUser(null)
        setStatus('anonymous')
      })

    return () => {
      active = false
    }
  }, [])

  const clearSession = useCallback(() => {
    setUser(null)
    setStatus('anonymous')
  }, [])

  const login = useCallback(async (credentials: Credentials) => {
    const { user } = await authApi.login(credentials)
    setUser(user)
    setStatus('authenticated')
    return user
  }, [])

  const register = useCallback(async (credentials: Credentials) => {
    const { user } = await authApi.register(credentials)
    setUser(user)
    setStatus('authenticated')
    return user
  }, [])

  const logout = useCallback(async () => {
    try {
      await authApi.logout()
    } finally {
      // The server clears the cookie; local state must not depend on that call
      // succeeding, or a network blip would leave the UI signed in.
      setUser(null)
      setStatus('anonymous')
    }
  }, [])

  const value = useMemo<AuthValue>(
    () => ({ status, user, login, register, logout, clearSession }),
    [status, user, login, register, logout, clearSession],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthValue {
  const value = useContext(AuthContext)
  if (!value) {
    throw new Error('useAuth must be used inside <AuthProvider>')
  }
  return value
}
