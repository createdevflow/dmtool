package handlers

import (
	"strconv"

	"backend/internal/repository"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	insightRepo repository.InsightRepository
}

func NewTaskHandler(insightRepo repository.InsightRepository) *TaskHandler {
	return &TaskHandler{
		insightRepo: insightRepo,
	}
}

func (h *TaskHandler) Toggle(c *gin.Context) {
	idStr := c.Param("id")
	projectIDStr := c.Query("project_id")
	
	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid task ID", "INVALID_ID")
		return
	}

	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)

	task, err := h.insightRepo.ToggleTask(uint(id), uint(projectID))
	if err != nil {
		utils.NotFound(c, "Task not found")
		return
	}

	utils.Success(c, task, nil)
}
