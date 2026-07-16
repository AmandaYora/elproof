package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/logger"
)

type ActivityRepository interface {
	Create(ctx context.Context, entry *domain.ActivityLogEntry) error
	ListByProject(ctx context.Context, projectID int64, limit int) ([]domain.ActivityLogEntry, error)
}

// ActivityService is the append-only recorder every other service in this
// module calls after a mutation (ADR-0007) — never updated, never deleted.
type ActivityService struct {
	repo ActivityRepository
}

func NewActivityService(repo ActivityRepository) *ActivityService {
	return &ActivityService{repo: repo}
}

// Record is best-effort: a failure to log activity should never fail the
// mutation it's describing, so errors are logged, not returned.
func (s *ActivityService) Record(ctx context.Context, projectID *int64, activityType domain.ActivityType, actorStaffID int64, entityType, entityID, entityLabel, description string) {
	entry := &domain.ActivityLogEntry{
		ProjectID: projectID, Type: activityType, ActorStaffID: actorStaffID,
		EntityType: entityType, EntityID: entityID, EntityLabel: entityLabel,
		Description: description, CreatedAt: time.Now(),
	}
	if err := s.repo.Create(ctx, entry); err != nil {
		logger.Error("failed to record activity log entry: %v", err)
	}
}

func (s *ActivityService) ListByProject(ctx context.Context, projectID int64, limit int) ([]domain.ActivityLogEntry, error) {
	return s.repo.ListByProject(ctx, projectID, limit)
}
