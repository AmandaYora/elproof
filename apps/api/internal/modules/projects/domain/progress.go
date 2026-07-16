package domain

import "time"

// MilestoneStats/ComputeMilestoneStats port mock/selectors.ts's
// computeMilestoneStats exactly — same field names, same ratio formula —
// so backend numbers match the pre-integration mock as a regression baseline
// (PLAN.md Fase 4 checkpoint).
type MilestoneStats struct {
	Total      int
	Completed  int
	InProgress int
	Blocked    int
	NotStarted int
	Cancelled  int
	Overdue    int
	Ratio      float64
}

type MilestoneLike struct {
	Status     MilestoneStatus
	TargetDate time.Time
}

// IsMilestoneOverdue takes "now" explicitly (asOf) rather than reading the
// wall clock — see ADR-0007: the caller (application layer) is responsible
// for supplying the single, consistent "now" for a whole computation.
func IsMilestoneOverdue(status MilestoneStatus, targetDate time.Time, asOf time.Time) bool {
	if status == MilestoneCompleted || status == MilestoneCancelled {
		return false
	}
	return targetDate.Before(asOf)
}

func ComputeMilestoneStats(milestones []MilestoneLike, asOf time.Time) MilestoneStats {
	var stats MilestoneStats
	relevantTotal := 0
	for _, m := range milestones {
		if m.Status == MilestoneCancelled {
			stats.Cancelled++
			continue
		}
		relevantTotal++
		switch m.Status {
		case MilestoneCompleted:
			stats.Completed++
		case MilestoneInProgress:
			stats.InProgress++
		case MilestoneBlocked:
			stats.Blocked++
		case MilestoneNotStarted:
			stats.NotStarted++
		}
		if IsMilestoneOverdue(m.Status, m.TargetDate, asOf) {
			stats.Overdue++
		}
	}
	stats.Total = relevantTotal
	if relevantTotal == 0 {
		stats.Ratio = 0
	} else {
		stats.Ratio = float64(stats.Completed) / float64(relevantTotal)
	}
	return stats
}

type ProjectCondition string

const (
	ConditionOnTrack  ProjectCondition = "On Track"
	ConditionAttention ProjectCondition = "Attention"
	ConditionAtRisk    ProjectCondition = "At Risk"
)

type ProjectProgress struct {
	ProjectMilestoneStats     MilestoneStats
	VendorMilestoneStats      MilestoneStats
	OverallPercent            int
	Condition                 ProjectCondition
	OpenIssueCount            int
	CriticalOrHighOpenCount   int
	OverdueMilestoneCount     int
	IncompleteEvidenceCount   int
}

// ComputeProjectProgress ports mock/selectors.ts's computeProjectProgress —
// same weighting (50/50 project vs vendor milestone ratio, falling back to
// whichever side has data), same condition thresholds.
func ComputeProjectProgress(
	projectStats MilestoneStats,
	vendorStats MilestoneStats,
	openIssues []VendorIssue,
	incompleteEvidenceCount int,
) ProjectProgress {
	var overallPercent int
	if projectStats.Total == 0 && vendorStats.Total == 0 {
		overallPercent = 0
	} else {
		pRatio := vendorStats.Ratio
		if projectStats.Total > 0 {
			pRatio = projectStats.Ratio
		}
		vRatio := projectStats.Ratio
		if vendorStats.Total > 0 {
			vRatio = vendorStats.Ratio
		}
		overallPercent = int(((pRatio * 0.5) + (vRatio * 0.5)) * 100)
	}

	criticalOrHigh := 0
	hasCritical := false
	for _, issue := range openIssues {
		if issue.Impact == ImpactHigh || issue.Impact == ImpactCritical {
			criticalOrHigh++
		}
		if issue.Impact == ImpactCritical {
			hasCritical = true
		}
	}
	overdueMilestoneCount := projectStats.Overdue + vendorStats.Overdue

	condition := ConditionOnTrack
	switch {
	case hasCritical || (criticalOrHigh > 0 && overdueMilestoneCount > 0):
		condition = ConditionAtRisk
	case len(openIssues) > 0 || overdueMilestoneCount > 0 || incompleteEvidenceCount > 0:
		condition = ConditionAttention
	}

	return ProjectProgress{
		ProjectMilestoneStats:   projectStats,
		VendorMilestoneStats:    vendorStats,
		OverallPercent:          overallPercent,
		Condition:               condition,
		OpenIssueCount:          len(openIssues),
		CriticalOrHighOpenCount: criticalOrHigh,
		OverdueMilestoneCount:   overdueMilestoneCount,
		IncompleteEvidenceCount: incompleteEvidenceCount,
	}
}
