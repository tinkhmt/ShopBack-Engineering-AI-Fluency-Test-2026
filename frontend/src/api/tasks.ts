import { request } from './client'
import type { CreateTaskInput, TaskResponse, TasksResponse, UpdateTaskInput } from '../types'

export const tasksApi = {
  list: () => request<TasksResponse>('GET', '/api/tasks'),
  create: (input: CreateTaskInput) => request<TaskResponse>('POST', '/api/tasks', input),
  update: (id: number, input: UpdateTaskInput) =>
    request<TaskResponse>('PATCH', `/api/tasks/${id}`, input),
  remove: (id: number) => request<void>('DELETE', `/api/tasks/${id}`),
}
