// Package entitlements is the single source of truth for "what the
// current plan allows." Handlers MUST go through this package — never
// reach into plans.max_sites or the user's subscription directly.
//
// Phase 2 implements the SEO project site-limit guard:
//   - Free plan: max 1 SEO project (goal = seo or both).
//   - Pro plans: max 3.
//   - Adding a new tier later = INSERT a row in plans; no code change.
//
// The "SEO project" definition (which goals count) lives here, not in
// the limit number — that shape is product policy, not data.
package entitlements

import (
	"context"
	"errors"

	"backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

// DefaultPlanCode is the plan a user gets on registration when no
// subscription row exists yet.
const DefaultPlanCode = models.PlanCodeFree

// QuotaExceededError is returned when the user is at or over their plan's
// limit. The handler reads its fields to populate the upgrade prompt.
type QuotaExceededError struct {
	PlanCode  string
	Current   int
	Limit     int
	QuotaName string
}

func (e *QuotaExceededError) Error() string {
	return "quota exceeded"
}

// Service is the entitlement guard. Reads the user's current
// subscription and the underlying plan row at request time.
type Service struct {
	db               *gorm.DB
	subscriptionRepo repository.SubscriptionRepository
	planRepo         repository.PlanRepository
	projectRepo      repository.ProjectRepository
}

// New constructs the service. All dependencies are required.
func New(
	db *gorm.DB,
	sub repository.SubscriptionRepository,
	plan repository.PlanRepository,
	proj repository.ProjectRepository,
) *Service {
	return &Service{
		db:               db,
		subscriptionRepo: sub,
		planRepo:         plan,
		projectRepo:      proj,
	}
}

// ResolveSubscription returns the user's *active* subscription plus the
// underlying plan row. If no subscription exists, returns the default
// (free) plan so callers always have a concrete entitlement to reason
// about. Never returns ErrRecordNotFound unless the default plan row
// is missing — which would be a data problem, not a request problem.
func (s *Service) ResolveSubscription(ctx context.Context, userID uint) (*models.Subscription, *models.Plan, error) {
	sub, err := s.subscriptionRepo.FindCurrentByUser(userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}
	if sub == nil {
		// No row: fall through to the default plan lookup below.
	}
	planCode := DefaultPlanCode
	if sub != nil {
		planCode = sub.PlanCode
	}
	plan, err := s.planRepo.FindByCode(planCode)
	if err != nil {
		return nil, nil, err
	}
	return sub, plan, nil
}

// CanCreateSEOProject reports whether the user may add another SEO
// project. Goal-typed parameters: passing isSEO true means the project
// being created has goal = "seo" or "both". For social-only projects
// the call returns ok=true with the user's current plan info, so
// handlers can include plan details in successful responses too.
//
// Returns:
//   - ok: true if the new project is allowed.
//   - current: SEO project count BEFORE this create.
//   - limit: plan.max_sites (1 for free, 3 for pro).
//   - planCode, status: current plan + status.
//   - err: *QuotaExceededError when ok=false; nil otherwise.
func (s *Service) CanCreateSEOProject(ctx context.Context, userID uint, projectGoal string) (allowed bool, current int, limit int, planCode string, err error) {
	if !isSEOGoal(projectGoal) {
		// Social-only projects are not gated in phase 2.
		current, _ := s.currentSEOCount(userID)
		_, plan, lookupErr := s.ResolveSubscription(ctx, userID)
		if lookupErr != nil {
			return false, current, 0, "", lookupErr
		}
		return true, current, plan.MaxSites, plan.Code, nil
	}

	current, _ = s.currentSEOCount(userID)
	_, plan, lookupErr := s.ResolveSubscription(ctx, userID)
	if lookupErr != nil {
		return false, current, 0, "", lookupErr
	}

	if int(current) >= plan.MaxSites {
		return false, int(current), plan.MaxSites, plan.Code, &QuotaExceededError{
			PlanCode:  plan.Code,
			Current:   int(current),
			Limit:     plan.MaxSites,
			QuotaName: "seo_projects",
		}
	}
	return true, int(current), plan.MaxSites, plan.Code, nil
}

// currentSEOCount returns the live SEO project count for a user.
func (s *Service) currentSEOCount(userID uint) (int, error) {
	n, err := s.projectRepo.CountSEOByUser(userID)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// isSEOGoal reports whether the project goal counts against the SEO
// site limit. "both" counts because such a project surfaces in SEO
// drill-downs and would consume the same slot.
func isSEOGoal(goal string) bool {
	return goal == models.GoalSEO || goal == models.GoalBoth
}
