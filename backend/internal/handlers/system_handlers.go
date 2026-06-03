package handlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// SystemHandler handles automations, calendar events, and system-level actions.
type SystemHandler struct {
	taskRepo    repository.TaskRepository
	projectRepo repository.ProjectRepository
	cfg         *config.Config
}

func NewSystemHandler(taskRepo repository.TaskRepository, projectRepo repository.ProjectRepository, cfg *config.Config) *SystemHandler {
	return &SystemHandler{taskRepo: taskRepo, projectRepo: projectRepo, cfg: cfg}
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
				"id":            t.ID,
				"title":         t.Title,
				"day":           t.DueDate.Day(),
				"month":         int(t.DueDate.Month()),
				"year":          t.DueDate.Year(),
				"time":          t.DueDate.Format("3:04 PM"),
				"platform":      t.Platform,
				"content_type":  t.ContentType,
				"asset_name":    t.AssetName,
				"asset_url":     t.AssetURL,
				"thumbnail_name": t.ThumbnailName,
				"thumbnail_url":  t.ThumbnailURL,
				"caption":       t.Caption,
				"location":      t.Location,
				"music":         t.Music,
				"tags":          t.Tags,
				"publish_status": t.PublishStatus,
				"due_date":      t.DueDate.Format(time.RFC3339),
				"done":          t.Completed,
			})
		}
	}
	utils.Success(c, events, nil)
}

type CalendarEventRequest struct {
	ProjectID   uint   `json:"project_id" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Platform    string `json:"platform"`
	ContentType string `json:"content_type"`
	AssetName   string `json:"asset_name"`
	AssetURL    string `json:"asset_url"`
	AssetPath     string `json:"asset_path"`
	AssetMime     string `json:"asset_mime"`
	ThumbnailName string `json:"thumbnail_name"`
	ThumbnailURL  string `json:"thumbnail_url"`
	ThumbnailPath string `json:"thumbnail_path"`
	ThumbnailMime string `json:"thumbnail_mime"`
	Caption       string `json:"caption"`
	Location      string `json:"location"`
	Music       string `json:"music"`
	Tags        string `json:"tags"`
	DueDate     string `json:"due_date"` // ISO 8601: 2026-05-15T10:00:00Z
}

// CreateCalendarEvent creates a new task with a due date (appears on calendar).
func (h *SystemHandler) CreateCalendarEvent(c *gin.Context) {
	var req CalendarEventRequest
	if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		req.ProjectID = mustParseUint(c.PostForm("project_id"))
		req.Title = c.PostForm("title")
		req.Platform = c.PostForm("platform")
		req.ContentType = c.PostForm("content_type")
		req.Caption = c.PostForm("caption")
		req.Location = c.PostForm("location")
		req.Music = c.PostForm("music")
		req.Tags = c.PostForm("tags")
		req.DueDate = c.PostForm("due_date")
		if fileHeader, err := c.FormFile("file"); err == nil && fileHeader != nil {
			assetName, assetURL, assetPath, assetMime, saveErr := h.saveUploadedAsset(fileHeader)
			if saveErr != nil {
				utils.InternalError(c, "Failed to save uploaded file")
				return
			}
			req.AssetName = assetName
			req.AssetURL = assetURL
			req.AssetPath = assetPath
			req.AssetMime = assetMime
		}
		if fileHeader, err := c.FormFile("thumbnail"); err == nil && fileHeader != nil {
			thumbnailName, thumbnailURL, thumbnailPath, thumbnailMime, saveErr := h.saveUploadedAsset(fileHeader)
			if saveErr != nil {
				utils.InternalError(c, "Failed to save uploaded thumbnail")
				return
			}
			req.ThumbnailName = thumbnailName
			req.ThumbnailURL = thumbnailURL
			req.ThumbnailPath = thumbnailPath
			req.ThumbnailMime = thumbnailMime
		}
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.ValidationError(c, err)
			return
		}
	}

	if req.ProjectID == 0 || req.Title == "" {
		utils.ValidationError(c, fmt.Errorf("project_id and title are required"))
		return
	}

	var dueDate *time.Time
	if req.DueDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.DueDate)
		if err == nil {
			utcParsed := parsed.UTC()
			dueDate = &utcParsed
		}
	}

	source := "manual"
	if req.Platform != "" {
		source = req.Platform
	}

	task := &models.Task{
		ProjectID:     req.ProjectID,
		Title:         req.Title,
		Platform:      req.Platform,
		ContentType:   req.ContentType,
		AssetName:     req.AssetName,
		AssetURL:      req.AssetURL,
		AssetPath:     req.AssetPath,
		AssetMime:     req.AssetMime,
		ThumbnailName: req.ThumbnailName,
		ThumbnailURL:  req.ThumbnailURL,
		ThumbnailPath: req.ThumbnailPath,
		ThumbnailMime: req.ThumbnailMime,
		Caption:       req.Caption,
		Location:      req.Location,
		Music:         req.Music,
		Tags:          req.Tags,
		Source:        source,
		DueDate:       dueDate,
	}
	if err := h.taskRepo.Create(task); err != nil {
		utils.InternalError(c, "Failed to create calendar event")
		return
	}

	event := gin.H{
		"id":            task.ID,
		"title":         task.Title,
		"platform":      req.Platform,
		"content_type":  task.ContentType,
		"asset_name":    task.AssetName,
		"asset_url":     task.AssetURL,
		"thumbnail_name": task.ThumbnailName,
		"thumbnail_url":  task.ThumbnailURL,
		"caption":       task.Caption,
		"location":      task.Location,
		"music":         task.Music,
		"tags":          task.Tags,
	}
	if dueDate != nil {
		event["day"] = dueDate.Day()
		event["month"] = int(dueDate.Month())
		event["year"] = dueDate.Year()
		event["time"] = dueDate.Format("3:04 PM")
	}
	utils.Success(c, event, nil)
}

func (h *SystemHandler) UpdateCalendarEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid event ID", "INVALID_ID")
		return
	}

	task, err := h.taskRepo.FindByID(uint(id))
	if err != nil {
		utils.NotFound(c, "Scheduled event not found")
		return
	}

	var req CalendarEventRequest
	if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		req.ProjectID = task.ProjectID
		req.Title = c.PostForm("title")
		req.Platform = c.PostForm("platform")
		req.ContentType = c.PostForm("content_type")
		req.Caption = c.PostForm("caption")
		req.Location = c.PostForm("location")
		req.Music = c.PostForm("music")
		req.Tags = c.PostForm("tags")
		req.DueDate = c.PostForm("due_date")
		if fileHeader, err := c.FormFile("file"); err == nil && fileHeader != nil {
			assetName, assetURL, assetPath, assetMime, saveErr := h.saveUploadedAsset(fileHeader)
			if saveErr != nil {
				utils.InternalError(c, "Failed to save uploaded file")
				return
			}
			req.AssetName = assetName
			req.AssetURL = assetURL
			req.AssetPath = assetPath
			req.AssetMime = assetMime
		}
		if fileHeader, err := c.FormFile("thumbnail"); err == nil && fileHeader != nil {
			thumbnailName, thumbnailURL, thumbnailPath, thumbnailMime, saveErr := h.saveUploadedAsset(fileHeader)
			if saveErr != nil {
				utils.InternalError(c, "Failed to save uploaded thumbnail")
				return
			}
			req.ThumbnailName = thumbnailName
			req.ThumbnailURL = thumbnailURL
			req.ThumbnailPath = thumbnailPath
			req.ThumbnailMime = thumbnailMime
		}
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.ValidationError(c, err)
			return
		}
	}

	if req.Title != "" {
		task.Title = req.Title
	}
	if req.Platform != "" {
		task.Platform = req.Platform
	}
	if req.ContentType != "" {
		task.ContentType = req.ContentType
	}
	if req.AssetName != "" {
		task.AssetName = req.AssetName
	}
	if req.AssetURL != "" {
		task.AssetURL = req.AssetURL
	}
	if req.AssetPath != "" {
		task.AssetPath = req.AssetPath
	}
	if req.AssetMime != "" {
		task.AssetMime = req.AssetMime
	}
	if req.ThumbnailName != "" {
		task.ThumbnailName = req.ThumbnailName
	}
	if req.ThumbnailURL != "" {
		task.ThumbnailURL = req.ThumbnailURL
	}
	if req.ThumbnailPath != "" {
		task.ThumbnailPath = req.ThumbnailPath
	}
	if req.ThumbnailMime != "" {
		task.ThumbnailMime = req.ThumbnailMime
	}
	if req.Caption != "" {
		task.Caption = req.Caption
	}
	if req.Location != "" {
		task.Location = req.Location
	}
	if req.Music != "" {
		task.Music = req.Music
	}
	if req.Tags != "" {
		task.Tags = req.Tags
	}
	if req.DueDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.DueDate)
		if err == nil {
			utcParsed := parsed.UTC()
			task.DueDate = &utcParsed
		}
	}

	if err := h.taskRepo.Update(task); err != nil {
		utils.InternalError(c, "Failed to update calendar event")
		return
	}

	utils.Success(c, task, nil)
}

func (h *SystemHandler) DeleteCalendarEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid event ID", "INVALID_ID")
		return
	}

	if err := h.taskRepo.Delete(uint(id)); err != nil {
		utils.InternalError(c, "Failed to delete calendar event")
		return
	}

	utils.Success(c, gin.H{"id": id}, nil)
}

func mustParseUint(value string) uint {
	parsed, _ := strconv.ParseUint(value, 10, 32)
	return uint(parsed)
}

func (h *SystemHandler) saveUploadedAsset(fileHeader *multipart.FileHeader) (string, string, string, string, error) {
	if err := os.MkdirAll(h.cfg.UploadDir, 0755); err != nil {
		return "", "", "", "", err
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", "", "", "", err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	fileName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(h.cfg.UploadDir, fileName)
	dst, err := os.Create(filePath)
	if err != nil {
		return "", "", "", "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", "", "", "", err
	}

	assetURL := strings.TrimRight(h.cfg.PublicBaseURL, "/") + "/uploads/" + fileName
	assetMime := fileHeader.Header.Get("Content-Type")
	return fileHeader.Filename, assetURL, filePath, assetMime, nil
}

func platformFromSource(source string) string {
	switch source {
	case "instagram", "facebook", "twitter", "linkedin":
		return source
	default:
		return "blog"
	}
}
