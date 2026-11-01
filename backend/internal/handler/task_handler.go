package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/tinkhmt/shopback-todo/backend/internal/dto"
	apperrors "github.com/tinkhmt/shopback-todo/backend/internal/errors"
	"github.com/tinkhmt/shopback-todo/backend/internal/model"
	"github.com/tinkhmt/shopback-todo/backend/internal/service"
	"github.com/tinkhmt/shopback-todo/backend/internal/util"
)

// TaskHandler is the HTTP boundary of tasks. Every route sits behind
// RequireSession, so a user ID is always present.
type TaskHandler struct {
	service *service.TaskService
}

func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{service: svc}
}

func toTaskResponse(task *model.Task) dto.TaskResponse {
	return dto.TaskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Completed:   task.Completed,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
}

func (h *TaskHandler) Create(c *gin.Context) {
	userID, ok := util.UserID(c)
	if !ok {
		apperrors.Fail(c, apperrors.Unauthorized("authentication required"))
		return
	}

	var req dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.Fail(c, apperrors.BindError(err))
		return
	}

	task, err := h.service.Create(c.Request.Context(), userID, req.Title, req.Description)
	if err != nil {
		apperrors.Fail(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.TaskEnvelope{Task: toTaskResponse(task)})
}

func (h *TaskHandler) List(c *gin.Context) {
	userID, ok := util.UserID(c)
	if !ok {
		apperrors.Fail(c, apperrors.Unauthorized("authentication required"))
		return
	}

	tasks, err := h.service.List(c.Request.Context(), userID, c.Query("status"))
	if err != nil {
		apperrors.Fail(c, err)
		return
	}

	resp := dto.TasksResponse{Tasks: make([]dto.TaskResponse, 0, len(tasks))}
	for i := range tasks {
		resp.Tasks = append(resp.Tasks, toTaskResponse(&tasks[i]))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *TaskHandler) Update(c *gin.Context) {
	userID, ok := util.UserID(c)
	if !ok {
		apperrors.Fail(c, apperrors.Unauthorized("authentication required"))
		return
	}

	id, ok := parseTaskID(c)
	if !ok {
		return
	}

	var req dto.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperrors.Fail(c, apperrors.BindError(err))
		return
	}

	task, err := h.service.Update(c.Request.Context(), userID, id, req.Title, req.Description, req.Completed)
	if err != nil {
		apperrors.Fail(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.TaskEnvelope{Task: toTaskResponse(task)})
}

func (h *TaskHandler) Delete(c *gin.Context) {
	userID, ok := util.UserID(c)
	if !ok {
		apperrors.Fail(c, apperrors.Unauthorized("authentication required"))
		return
	}

	id, ok := parseTaskID(c)
	if !ok {
		return
	}

	if err := h.service.Delete(c.Request.Context(), userID, id); err != nil {
		apperrors.Fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func parseTaskID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		apperrors.Fail(c, apperrors.BadRequest("invalid_id", "task id must be a positive integer"))
		return 0, false
	}
	return id, true
}
