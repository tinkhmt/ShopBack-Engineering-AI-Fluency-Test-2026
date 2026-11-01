import { useCallback, useEffect, useState } from 'react'

import { ApiError } from '../api/client'
import { tasksApi } from '../api/tasks'
import { useAuth } from '../auth/AuthContext'
import type { Task, UpdateTaskInput } from '../types'

export interface TaskValues {
  title: string
  description: string
}

export interface TasksValue {
  /** null while the first load (or a retry) is in flight. */
  tasks: Task[] | null
  loadError: string | null
  actionError: string | null
  busyId: number | null
  reload: () => void
  create: (values: TaskValues) => Promise<void>
  save: (task: Task, values: TaskValues) => Promise<void>
  toggle: (task: Task) => Promise<void>
  remove: (task: Task) => Promise<void>
  dismissActionError: () => void
}

/**
 * One list, no pagination: the whole list is fetched once and the screen filters
 * it. Mutations are server-first and replace the row with what the API returns,
 * so counts and filters can never disagree with the data.
 */
export function useTasks(): TasksValue {
  const { clearSession } = useAuth()
  const [tasks, setTasks] = useState<Task[] | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<number | null>(null)

  const load = useCallback(async () => {
    setTasks(null)
    setLoadError(null)
    try {
      const { tasks } = await tasksApi.list()
      setTasks(tasks)
    } catch (error) {
      if (expired(error, clearSession)) return
      setLoadError(messageOf(error))
    }
  }, [clearSession])

  useEffect(() => {
    void load()
  }, [load])

  const replace = useCallback((updated: Task) => {
    setTasks(
      (current) => current?.map((task) => (task.id === updated.id ? updated : task)) ?? null,
    )
  }, [])

  const create = useCallback(async (values: TaskValues) => {
    try {
      const { task } = await tasksApi.create(values)
      setTasks((current) => (current ? [task, ...current] : [task]))
    } catch (error) {
      expired(error, clearSession)
      throw error
    }
  }, [clearSession])

  // Only changed keys are sent, which is the point of the pointer-based PATCH.
  const save = useCallback(
    async (task: Task, values: TaskValues) => {
      const patch: UpdateTaskInput = {}
      if (values.title !== task.title) patch.title = values.title
      if (values.description !== task.description) patch.description = values.description
      if (Object.keys(patch).length === 0) return

      try {
        const { task: updated } = await tasksApi.update(task.id, patch)
        replace(updated)
      } catch (error) {
        expired(error, clearSession)
        throw error
      }
    },
    [clearSession, replace],
  )

  const toggle = useCallback(
    async (task: Task) => {
      setActionError(null)
      setBusyId(task.id)
      try {
        const { task: updated } = await tasksApi.update(task.id, { completed: !task.completed })
        replace(updated)
      } catch (error) {
        if (expired(error, clearSession)) return
        setActionError(`Couldn’t update “${task.title}”. ${messageOf(error)}`)
      } finally {
        setBusyId(null)
      }
    },
    [clearSession, replace],
  )

  const remove = useCallback(
    async (task: Task) => {
      setActionError(null)
      setBusyId(task.id)
      try {
        await tasksApi.remove(task.id)
        drop(task.id)
      } catch (error) {
        if (expired(error, clearSession)) return
        // 404 means it is already gone; the user's intent is satisfied.
        if (error instanceof ApiError && error.status === 404) {
          drop(task.id)
          return
        }
        setActionError(`Couldn’t delete “${task.title}”. ${messageOf(error)}`)
      } finally {
        setBusyId(null)
      }
    },
    [clearSession],
  )

  function drop(id: number) {
    setTasks((current) => current?.filter((task) => task.id !== id) ?? null)
  }

  return {
    tasks,
    loadError,
    actionError,
    busyId,
    reload: load,
    create,
    save,
    toggle,
    remove,
    dismissActionError: () => setActionError(null),
  }
}

/** An expired session is the guard's job, not an error to render. */
function expired(error: unknown, clearSession: () => void): boolean {
  if (error instanceof ApiError && error.status === 401) {
    clearSession()
    return true
  }
  return false
}

function messageOf(error: unknown): string {
  return error instanceof ApiError ? error.message : 'Try again.'
}
