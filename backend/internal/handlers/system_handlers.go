package handlers

import (
	"strconv"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// SystemHandler handles automations, calendar events, and system-level actions.
type SystemHandler struct {
	taskRepo    repository.TaskRepository
	projectRepo repository.ProjectRepository
}

func NewSystemHandler(taskRepo repository.TaskRepository, projectRepo repository.ProjectRepository) *SystemHandler {
	return &SystemHandler{taskRepo: taskRepo, projectRepo: projectRepo}
}

// ── Automations ────────────────────────────────────────────────────────────────

// GetAutomations returns all automation tasks for a project.
func (h *SystemHandler) GetAutomations(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}
	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)
	tasks, err := h.taskRepo.FindAllByProject(uint(projectID))
	if err != nil {
		utils.InternalError(c, "Failed to fetch automations")
		return
	}

	// Filter to automation-type tasks only
	automations := []gin.H{}
	for _, t := range tasks {
		if t.Source == "automation" {
			automations = append(automations, gin.H{
				"id":       t.ID,
				"name":     t.Title,
				"trigger":  "engagement drops below 2%",
				"action":   "Generate content suggestions",
				"status":   !t.Completed,
				"lastRun":  t.UpdatedAt.Format("Jan 2, 3:04 PM"),
				"source":   t.Source,
			})
		}
	}
	utils.Success(c, automations, nil)
}

type AutomationRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Trigger   string `json:"trigger"`
	Action    string `json:"action"`
}

// CreateAutomation creates a new automation workflow (stored as a Task with source=automation).
func (h *SystemHandler) CreateAutomation(c *gin.Context) {
	var req AutomationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}

	title := req.Name
	if req.Trigger != "" && req.Action != "" {
		title = req.Name + " [If: " + req.Trigger + " → Then: " + req.Action + "]"
	}

	task := &models.Task{
		ProjectID: req.ProjectID,
		Title:     title,
		Source:    "automation",
		Completed: false,
	}
	if err := h.taskRepo.Create(task); err != nil {
		utils.InternalError(c, "Failed to create automation")
		return
	}
	utils.Success(c, gin.H{
		"id":      task.ID,
		"name":    req.Name,
		"trigger": req.Trigger,
		"action":  req.Action,
		"status":  true,
		"lastRun": "Never",
	}, nil)
}

// ToggleAutomation toggles an automation's active state.
func (h *SystemHandler) ToggleAutomation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid ID", "INVALID_ID")
		return
	}

	task, err := h.taskRepo.FindByID(uint(id))
	if err != nil {
		utils.NotFound(c, "Automation not found")
		return
	}
	task.Completed = !task.Completed
	if err := h.taskRepo.Update(task); err != nil {
		utils.InternalError(c, "Failed to toggle automation")
		return
	}
	utils.Success(c, gin.H{"id": task.ID, "status": !task.Completed}, nil)
}

// ── Calendar ──────────────────────────────────────────────────────────────────

// GetCalendar returns all tasks with due dates as calendar events.
func (h *SystemHandler) GetCalendar(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		utils.BadRequest(c, "project_id is required", "MISSING_PROJECT_ID")
		return
	}
	projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)
	tasks, err := h.taskRepo.FindAllByProject(uint(projectID))
	if err != nil {
		utils.InternalError(c, "Failed to fetch calendar")
		return
	}

	events := []gin.H{}
	for _, t := range tasks {
		if t.DueDate != nil {
			events = append(events, gin.H{
				"id":       t.ID,
				"title":    t.Title,
				"day":      t.DueDate.Day(),
				"month":    int(t.DueDate.Month()),
				"year":     t.DueDate.Year(),
				"time":     t.DueDate.Format("3:04 PM"),
				"platform": platformFromSource(t.Source),
				"done":     t.Completed,
			})
		}
	}
	utils.Success(c, events, nil)
}

type CalendarEventRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	Title     string `json:"title" binding:"required"`
	Platform  string `json:"platform"`
	DueDate   string `json:"due_date"` // ISO 8601: 2026-05-15T10:00:00Z
}

// CreateCalendarEvent creates a new task with a due date (appears on calendar).
func (h *SystemHandler) CreateCalendarEvent(c *gin.Context) {
	var req CalendarEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}

	var dueDate *time.Time
	if req.DueDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.DueDate)
		if err == nil {
			dueDate = &parsed
		}
	}

	source := "manual"
	if req.Platform != "" {
		source = req.Platform
	}

	task := &models.Task{
		ProjectID: req.ProjectID,
		Title:     req.Title,
		Source:    source,
		DueDate:   dueDate,
	}
	if err := h.taskRepo.Create(task); err != nil {
		utils.InternalError(c, "Failed to create calendar event")
		return
	}

	event := gin.H{
		"id":       task.ID,
		"title":    task.Title,
		"platform": req.Platform,
	}
	if dueDate != nil {
		event["day"] = dueDate.Day()
		event["month"] = int(dueDate.Month())
		event["year"] = dueDate.Year()
		event["time"] = dueDate.Format("3:04 PM")
	}
	utils.Success(c, event, nil)
}

func platformFromSource(source string) string {
	switch source {
	case "instagram", "facebook", "twitter", "linkedin":
		return source
	default:
		return "blog"
	}
}
