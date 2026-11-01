import { request } from './client'
import type { AuthResponse, Credentials } from '../types'

export const authApi = {
  me: () => request<AuthResponse>('GET', '/api/auth/me'),
  login: (credentials: Credentials) => request<AuthResponse>('POST', '/api/auth/login', credentials),
  register: (credentials: Credentials) =>
    request<AuthResponse>('POST', '/api/auth/register', credentials),
  logout: () => request<void>('POST', '/api/auth/logout'),
}
