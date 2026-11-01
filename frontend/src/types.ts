/** Wire types. They mirror the Go DTOs exactly; timestamps stay ISO strings. */

export interface User {
  id: number
  email: string
  createdAt: string
}

export interface Task {
  id: number
  title: string
  description: string
  completed: boolean
  createdAt: string
  updatedAt: string
}

export type TaskFilter = 'all' | 'active' | 'completed'

export interface Credentials {
  email: string
  password: string
}

export interface CreateTaskInput {
  title: string
  description?: string
}

/** Absent keys are left unchanged by PATCH; `description: ''` clears the field. */
export interface UpdateTaskInput {
  title?: string
  description?: string
  completed?: boolean
}

export interface AuthResponse {
  user: User
}

export interface TaskResponse {
  task: Task
}

export interface TasksResponse {
  tasks: Task[]
}

export interface ApiErrorBody {
  error: {
    code: string
    message: string
    fields?: Record<string, string>
  }
}
