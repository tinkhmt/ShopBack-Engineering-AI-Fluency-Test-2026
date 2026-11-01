import { useMemo, useState } from 'react'

import AppHeader from '../components/AppHeader'
import FilterTabs from '../components/FilterTabs'
import TaskForm from '../components/TaskForm'
import TaskItem from '../components/TaskItem'
import { useTasks } from '../tasks/useTasks'
import type { Task, TaskFilter } from '../types'

export default function TasksPage() {
  const {
    tasks,
    loadError,
    actionError,
    busyId,
    reload,
    create,
    save,
    toggle,
    remove,
    dismissActionError,
  } = useTasks()
  const [filter, setFilter] = useState<TaskFilter>('all')
  const [editingId, setEditingId] = useState<number | null>(null)

  const counts = useMemo(() => {
    const total = tasks?.length ?? 0
    const done = tasks?.filter((task) => task.completed).length ?? 0
    return { total, done, left: total - done }
  }, [tasks])

  const visible = useMemo(() => {
    if (!tasks) return []
    if (filter === 'active') return tasks.filter((task) => !task.completed)
    if (filter === 'completed') return tasks.filter((task) => task.completed)
    return tasks
  }, [tasks, filter])

  const summary = loadError
    ? 'Tasks unavailable.'
    : tasks === null
      ? 'Loading tasks…'
      : `${counts.left} left · ${counts.done} done`

  const emptyCopy =
    counts.total === 0
      ? 'Nothing here yet. Add your first task above.'
      : filter === 'active'
        ? 'No active tasks. Everything is done.'
        : 'Nothing completed yet.'

  function startEdit(task: Task) {
    setEditingId(task.id)
  }

  function closeEdit(task: Task) {
    setEditingId(null)
    // The row becomes a button again on the next render; put focus back on it.
    requestAnimationFrame(() => document.getElementById(`task-${task.id}-title`)?.focus())
  }

  async function handleSave(task: Task, values: { title: string; description: string }) {
    await save(task, values)
    closeEdit(task)
  }

  async function handleDelete(task: Task) {
    if (!window.confirm(`Delete “${task.title}”?`)) return
    await remove(task)
    // The row is gone, so hand focus back to the primary action.
    document.getElementById('new-task-title')?.focus()
  }

  return (
    <div className="shell">
      <a className="skip" href="#main">
        Skip to content
      </a>

      <AppHeader />

      <main className="canvas" id="main">
        <div className="page-head">
          <h1 className="title">Your tasks</h1>
          <p className="lead">{summary}</p>
        </div>

        <section className="card" aria-label="Tasks">
          <TaskForm submitLabel="Add" onSubmit={create} titleInputId="new-task-title" />

          <div className="list-tools">
            <FilterTabs value={filter} onChange={setFilter} />
          </div>

          {actionError && (
            <p className="list-notice" role="alert">
              <span>{actionError}</span>
              <button className="text-button" type="button" onClick={dismissActionError}>
                Dismiss
              </button>
            </p>
          )}

          {loadError ? (
            <div className="list-state" role="alert">
              <p>Couldn’t load your tasks. {loadError}</p>
              <button className="text-button" type="button" onClick={reload}>
                Try again
              </button>
            </div>
          ) : tasks === null ? (
            <div className="list-state" aria-busy="true">
              <span className="sr-only">Loading tasks…</span>
              <div className="skeleton-row">
                <span className="skeleton" />
              </div>
              <div className="skeleton-row">
                <span className="skeleton" />
              </div>
              <div className="skeleton-row">
                <span className="skeleton" />
              </div>
            </div>
          ) : visible.length === 0 ? (
            <p className="list-state">{emptyCopy}</p>
          ) : (
            <ul className="task-list">
              {visible.map((task) => (
                <TaskItem
                  key={task.id}
                  task={task}
                  busy={busyId === task.id}
                  editing={editingId === task.id}
                  onToggle={toggle}
                  onEdit={startEdit}
                  onDelete={handleDelete}
                  onSave={handleSave}
                  onCancel={closeEdit}
                />
              ))}
            </ul>
          )}
        </section>
      </main>
    </div>
  )
}
