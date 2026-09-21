package handlers

import (
	"regexp"
	"strconv"
	"strings"

	"backend/internal/models"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// roleCodeRegex matches plan codes: lowercase letter, then letters/digits/underscore.
var roleCodeRegex = regexp.MustCompile(`^[a-z][a-z0-9_]{1,40}$`)

type adminRoleDTO struct {
	ID          uint                `json:"id"`
	Code        string              `json:"code"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	IsSystem    bool                `json:"is_system"`
	Permissions []models.Permission `json:"permissions"`
	UserCount   int64               `json:"user_count"`
}

type roleWriteRequest struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

func toRoleDTO(r models.Role, userCount int64) adminRoleDTO {
	perms := r.Permissions
	if perms == nil {
		perms = []models.Permission{}
	}
	return adminRoleDTO{
		ID:          r.ID,
		Code:        r.Code,
		Name:        r.Name,
		Description: r.Description,
		IsSystem:    r.IsSystem,
		Permissions: perms,
		UserCount:   userCount,
	}
}

func (h *AdminHandler) roleDTO(r models.Role) adminRoleDTO {
	return toRoleDTO(r, h.countUsersWithRole(r.Code))
}

// ListRoles returns staff roles (not customer owner/viewer).
func (h *AdminHandler) ListRoles(c *gin.Context) {
	var roles []models.Role
	if err := h.db.Preload("Permissions").Order("is_system DESC, code ASC").Find(&roles).Error; err != nil {
		utils.InternalError(c, "Failed to list roles")
		return
	}
	out := make([]adminRoleDTO, 0, len(roles))
	for _, r := range roles {
		out = append(out, h.roleDTO(r))
	}
	utils.Success(c, out, nil)
}

// ListPermissions returns the developer catalog (not leftover orphan codes).
func (h *AdminHandler) ListPermissions(c *gin.Context) {
	catalog := models.PermissionCatalog()
	codes := make([]string, 0, len(catalog))
	for _, p := range catalog {
		codes = append(codes, p.Code)
	}
	var rows []models.Permission
	if err := h.db.Where("code IN ?", codes).Order("category ASC, code ASC").Find(&rows).Error; err != nil {
		utils.InternalError(c, "Failed to list permissions")
		return
	}
	utils.Success(c, rows, nil)
}

// CreateRole inserts a custom staff role. Code is immutable after create.
func (h *AdminHandler) CreateRole(c *gin.Context) {
	var req roleWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}
	code := strings.TrimSpace(strings.ToLower(req.Code))
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		utils.BadRequest(c, "Code and name are required", "VALIDATION_ERROR")
		return
	}
	if !roleCodeRegex.MatchString(code) {
		utils.BadRequest(c, "Code must be lowercase letters, digits, and underscores, starting with a letter", "INVALID_CODE")
		return
	}
	if isReservedRoleCode(code) {
		utils.BadRequest(c, "That role code is reserved", "RESERVED_CODE")
		return
	}
	perms, err := h.loadCatalogPerms(req.Permissions)
	if err != nil {
		utils.BadRequest(c, err.Error(), "INVALID_PERMISSIONS")
		return
	}

	role := models.Role{
		Code:        code,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		IsSystem:    false,
	}
	if err := h.db.Create(&role).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			utils.BadRequest(c, "Role code already exists", "CODE_TAKEN")
			return
		}
		utils.InternalError(c, "Failed to create role")
		return
	}
	if err := h.replaceRolePerms(role.ID, perms); err != nil {
		utils.InternalError(c, "Failed to assign permissions")
		return
	}
	if err := h.db.Preload("Permissions").First(&role, role.ID).Error; err != nil {
		utils.InternalError(c, "Failed to load role")
		return
	}
	h.writeAudit(c, 0, "role.create", map[string]any{
		"code":        role.Code,
		"name":        role.Name,
		"permissions": permCodes(perms),
	})
	utils.Created(c, h.roleDTO(role))
}

// UpdateRole changes name, description, and permission set. System roles are frozen.
func (h *AdminHandler) UpdateRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "Invalid role id", "INVALID_ID")
		return
	}
	var req roleWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}
	var role models.Role
	if err := h.db.Preload("Permissions").First(&role, uint(id)).Error; err != nil {
		utils.NotFound(c, "Role not found")
		return
	}
	if role.IsSystem || role.Code == models.RoleCodeSuperAdmin {
		utils.BadRequest(c, "Cannot edit a system role", "SYSTEM_ROLE")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		utils.BadRequest(c, "Name is required", "VALIDATION_ERROR")
		return
	}
	perms, err := h.loadCatalogPerms(req.Permissions)
	if err != nil {
		utils.BadRequest(c, err.Error(), "INVALID_PERMISSIONS")
		return
	}
	before := map[string]any{
		"name":        role.Name,
		"description": role.Description,
		"permissions": permCodes(role.Permissions),
	}
	if err := h.db.Model(&role).Updates(map[string]any{
		"name":        name,
		"description": strings.TrimSpace(req.Description),
	}).Error; err != nil {
		utils.InternalError(c, "Failed to update role")
		return
	}
	if err := h.replaceRolePerms(role.ID, perms); err != nil {
		utils.InternalError(c, "Failed to assign permissions")
		return
	}
	if err := h.db.Preload("Permissions").First(&role, role.ID).Error; err != nil {
		utils.InternalError(c, "Failed to load role")
		return
	}
	h.writeAudit(c, 0, "role.update", map[string]any{
		"code":   role.Code,
		"before": before,
		"after": map[string]any{
			"name":        role.Name,
			"description": role.Description,
			"permissions": permCodes(role.Permissions),
		},
	})
	utils.Success(c, h.roleDTO(role), nil)
}

// DeleteRole removes a custom role with no users. Super Admin cannot be deleted.
func (h *AdminHandler) DeleteRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "Invalid role id", "INVALID_ID")
		return
	}
	var role models.Role
	if err := h.db.First(&role, uint(id)).Error; err != nil {
		utils.NotFound(c, "Role not found")
		return
	}
	if role.IsSystem || role.Code == models.RoleCodeSuperAdmin {
		utils.BadRequest(c, "Cannot delete a system role", "SYSTEM_ROLE")
		return
	}
	if n := h.countUsersWithRole(role.Code); n > 0 {
		utils.BadRequest(c, "Role is assigned to users", "ROLE_IN_USE")
		return
	}
	if err := h.db.Where("role_id = ?", role.ID).Delete(&models.RolePermission{}).Error; err != nil {
		utils.InternalError(c, "Failed to delete role")
		return
	}
	if err := h.db.Delete(&role).Error; err != nil {
		utils.InternalError(c, "Failed to delete role")
		return
	}
	h.writeAudit(c, 0, "role.delete", map[string]any{"code": role.Code, "name": role.Name})
	utils.Success(c, gin.H{"id": role.ID, "code": role.Code}, nil)
}

func isReservedRoleCode(code string) bool {
	switch code {
	case models.RoleAdmin, models.RoleOwner, models.RoleViewer:
		return true
	default:
		return false
	}
}

func (h *AdminHandler) loadCatalogPerms(codes []string) ([]models.Permission, error) {
	codes = uniqStrings(codes)
	if len(codes) == 0 {
		return nil, errNeedAdminAccess
	}
	allowed := map[string]bool{}
	for _, p := range models.PermissionCatalog() {
		allowed[p.Code] = true
	}
	for _, c := range codes {
		if !allowed[c] {
			return nil, errUnknownPerm
		}
	}
	var perms []models.Permission
	if err := h.db.Where("code IN ?", codes).Find(&perms).Error; err != nil {
		return nil, err
	}
	if len(perms) != len(codes) {
		return nil, errUnknownPerm
	}
	hasAccess := false
	for _, p := range perms {
		if p.Code == models.PermAdminAccess {
			hasAccess = true
			break
		}
	}
	if !hasAccess {
		return nil, errNeedAdminAccess
	}
	return perms, nil
}

func (h *AdminHandler) replaceRolePerms(roleID uint, perms []models.Permission) error {
	if err := h.db.Where("role_id = ?", roleID).Delete(&models.RolePermission{}).Error; err != nil {
		return err
	}
	for _, p := range perms {
		row := models.RolePermission{RoleID: roleID, PermissionID: p.ID}
		if err := h.db.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func permCodes(perms []models.Permission) []string {
	out := make([]string, 0, len(perms))
	for _, p := range perms {
		out = append(out, p.Code)
	}
	return out
}

func uniqStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

type permError string

func (e permError) Error() string { return string(e) }

const (
	errNeedAdminAccess permError = "Staff roles must include admin.access"
	errUnknownPerm     permError = "Unknown permission code"
)
