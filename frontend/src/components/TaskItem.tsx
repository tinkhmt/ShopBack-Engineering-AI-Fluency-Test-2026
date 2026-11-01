import type { Task } from '../types'
import TaskForm from './TaskForm'

interface TaskItemProps {
  task: Task
  busy: boolean
  editing: boolean
  onToggle: (task: Task) => void
  onEdit: (task: Task) => void
  onDelete: (task: Task) => void
  onSave: (task: Task, values: { title: string; description: string }) => Promise<void>
  onCancel: (task: Task) => void
}

export default function TaskItem({
  task,
  busy,
  editing,
  onToggle,
  onEdit,
  onDelete,
  onSave,
  onCancel,
}: TaskItemProps) {
  const titleId = `task-${task.id}-title`

  return (
    <li className="task" data-completed={task.completed}>
      {editing ? (
        <TaskForm
          initial={{ title: task.title, description: task.description }}
          submitLabel="Save"
          onSubmit={(values) => onSave(task, values)}
          onCancel={() => onCancel(task)}
          autoFocus
        />
      ) : (
        <>
          <input
            className="task-check"
            type="checkbox"
            checked={task.completed}
            disabled={busy}
            onChange={() => onToggle(task)}
            aria-label={`Mark “${task.title}” as ${task.completed ? 'active' : 'complete'}`}
          />
          <div className="task-body">
            <button className="task-title" id={titleId} type="button" onClick={() => onEdit(task)}>
              {task.title}
            </button>
            {task.description && <p className="task-note">{task.description}</p>}
          </div>
          <button
            className="task-delete"
            type="button"
            onClick={() => onDelete(task)}
            disabled={busy}
            aria-label={`Delete “${task.title}”`}
          >
            ×
          </button>
        </>
      )}
    </li>
  )
}
