package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/logger"
)

// PurgeRepository is what the purge needs from storage: a lock shared by every
// instance of the API, and a batch hard delete.
type PurgeRepository interface {
	// TryLock takes the cross-instance purge lock without waiting; ok is false
	// when another instance holds it. release must be called when ok is true.
	TryLock(ctx context.Context) (release func(), ok bool, err error)
	// PurgeBatch hard-deletes up to `batch` expired rows of each kind
	// (deleted_at <= cutoff) with their evidence rows, in one transaction, and
	// returns what went and the stored object keys to delete afterwards.
	PurgeBatch(ctx context.Context, cutoff time.Time, batch int) (domain.PurgeReport, []string, error)
}

// PurgeService permanently removes what has stayed deleted for longer than the
// retention (ADR-0034): payments and vendor engagements of this module, their
// evidence rows and their stored files.
type PurgeService struct {
	repo     PurgeRepository
	evidence *EvidenceService
}

func NewPurgeService(repo PurgeRepository, evidence *EvidenceService) *PurgeService {
	return &PurgeService{repo: repo, evidence: evidence}
}

const (
	purgeBatchSize = 200
	// purgeMaxPasses bounds one run (batch x passes rows per kind); whatever is
	// left is taken by the next run.
	purgeMaxPasses = 50
)

// Run purges everything due at now. Idempotent: a second run finds nothing. Safe
// with several instances: only the one holding the database lock works, the
// others return ran=false at once. Stored objects are deleted after each batch's
// transaction committed, best effort (an orphan object costs bytes, never a row
// pointing at a missing file).
func (s *PurgeService) Run(ctx context.Context, now time.Time) (report domain.PurgeReport, ran bool, err error) {
	release, ok, err := s.repo.TryLock(ctx)
	if err != nil {
		return report, false, err
	}
	if !ok {
		return report, false, nil
	}
	defer release()
	cutoff := domain.PurgeCutoff(now)
	for pass := 0; pass < purgeMaxPasses; pass++ {
		batch, keys, err := s.repo.PurgeBatch(ctx, cutoff, purgeBatchSize)
		if err != nil {
			return report, true, err
		}
		batch.Objects = s.evidence.DeleteStorageKeys(ctx, keys)
		report.Add(batch)
		if batch.Total() == 0 {
			break
		}
	}
	if report.Total() > 0 {
		logger.Info("purge: removed %d client payments, %d vendor payments, %d venue payments, %d vendor engagements (+%d milestones, %d kendala, %d payments), %d evidence rows, %d stored files",
			report.ClientPayments, report.VendorPayments, report.VenuePayments, report.Engagements,
			report.EngagementMilestones, report.EngagementIssues, report.EngagementPayments, report.Evidence, report.Objects)
	}
	return report, true, nil
}
