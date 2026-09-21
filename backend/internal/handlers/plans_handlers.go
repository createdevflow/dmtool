package handlers

import (
	"backend/internal/repository"
	"backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// PlansHandler exposes plan rows for the marketing /pricing page. It is
// intentionally public (no JWT middleware) so anonymous visitors can
// see pricing before signing up.
type PlansHandler struct {
	planRepo repository.PlanRepository
}

// NewPlansHandler constructs a plans handler.
func NewPlansHandler(planRepo repository.PlanRepository) *PlansHandler {
	return &PlansHandler{planRepo: planRepo}
}

// List returns the active plans, ordered by tier_rank asc. Cached at the
// server (no DB-level cache today) — the pricing page calls this on load.
// Pricing display values come from the plans row directly: monthly_cents
// and yearly_cents. The frontend formats them to /mo and /yr display
// strings.
func (h *PlansHandler) List(c *gin.Context) {
	plans, err := h.planRepo.ListActive()
	if err != nil {
		utils.InternalError(c, "Failed to load plans")
		return
	}
	utils.Success(c, plans, nil)
}
