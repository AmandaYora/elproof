package domain

import (
	"errors"
	"sort"
	"strings"
	"time"
)

// The Client Portal's two write paths (ADR-0047): the couple sends a transfer
// proof (it becomes an unverified client payment) and marks a task done (a
// Timeline step staff flagged "for the client"). The rules live here so the
// application layer and its tests share one source.

// PaymentSource says who put a client payment on the ledger.
type PaymentSource string

const (
	SourceStaff  PaymentSource = "staff"
	SourceClient PaymentSource = "client"
)

// Source is "client" when the couple sent the payment from the portal, "staff"
// otherwise.
func (p ClientPayment) Source() PaymentSource {
	if p.SubmittedByClientID != 0 {
		return SourceClient
	}
	return SourceStaff
}

// SubmissionType is the payment type a client submission is recorded with (the
// portal does not ask the couple to pick one): DP while nothing has been paid
// toward the contract yet, Pelunasan when the amount settles exactly what is
// left, Termin otherwise. Staff may change it before verifying. `applied` is
// domain.AppliedToContract of the live payments.
func SubmissionType(applied, contractValue, amount int64) PaymentType {
	switch {
	case applied <= 0:
		return PaymentDP
	case contractValue > 0 && applied+amount == contractValue:
		return PaymentPelunasan
	default:
		return PaymentTermin
	}
}

// Errors of marking a task done from the portal.
var (
	// ErrNotClientTask: the step is not flagged for the client.
	ErrNotClientTask = errors.New("this step is not a task for the client")
	// ErrClientTaskCancelled: the WO cancelled the step.
	ErrClientTaskCancelled = errors.New("this task was cancelled")
)

// CompleteByClient marks a client task done by `clientID` on `today` (the
// Jakarta calendar day). Idempotent: a step that is already Completed is left
// exactly as it is and changed is false. A step that is not for the client, or
// was cancelled, is refused.
func (m *ProjectMilestone) CompleteByClient(clientID int64, today time.Time) (changed bool, err error) {
	if !m.ForClient {
		return false, ErrNotClientTask
	}
	switch m.Status {
	case MilestoneCompleted:
		return false, nil
	case MilestoneCancelled:
		return false, ErrClientTaskCancelled
	}
	d := dayOf(today)
	m.Status, m.CompletedDate, m.CompletedByClientID = MilestoneCompleted, &d, clientID
	return true, nil
}

// maxCompletedClientTasks is how many finished tasks the portal home lists.
const maxCompletedClientTasks = 5

// SelectClientTasks is the portal home's task list: the steps flagged for the
// client that are not cancelled — open ones first by target date (then the
// schedule's own order), then the five most recently completed.
func SelectClientTasks(milestones []ProjectMilestone) []ProjectMilestone {
	var open, done []ProjectMilestone
	for _, m := range milestones {
		if !m.ForClient || m.Status == MilestoneCancelled {
			continue
		}
		if m.Status == MilestoneCompleted {
			done = append(done, m)
		} else {
			open = append(open, m)
		}
	}
	sort.SliceStable(open, func(i, j int) bool {
		a, b := open[i], open[j]
		if !a.TargetDate.Equal(b.TargetDate) {
			return a.TargetDate.Before(b.TargetDate)
		}
		if a.SortOrder != b.SortOrder {
			return a.SortOrder < b.SortOrder
		}
		return a.ID < b.ID
	})
	sort.SliceStable(done, func(i, j int) bool {
		a, b := completedOn(done[i]), completedOn(done[j])
		if !a.Equal(b) {
			return a.After(b)
		}
		return done[i].ID > done[j].ID
	})
	if len(done) > maxCompletedClientTasks {
		done = done[:maxCompletedClientTasks]
	}
	return append(open, done...)
}

func completedOn(m ProjectMilestone) time.Time {
	if m.CompletedDate == nil {
		return time.Time{}
	}
	return *m.CompletedDate
}

// PortalWhatsApp is the number the Client Portal's WhatsApp buttons open: the
// project PIC's phone when it is a usable WhatsApp number, otherwise the
// business WhatsApp of the workspace profile when that one is (a staff phone
// is free text: "-" or a half-typed number must not hide a good business
// number). Empty when neither is usable. The number is returned as typed.
func PortalWhatsApp(picPhone, businessWhatsApp string) string {
	if usableWhatsApp(picPhone) {
		return strings.TrimSpace(picPhone)
	}
	if usableWhatsApp(businessWhatsApp) {
		return strings.TrimSpace(businessWhatsApp)
	}
	return ""
}

// usableWhatsApp mirrors the web's whatsappNumber (shared/lib/access-link.ts):
// the digits, with 0… or 8… read as an Indonesian number, must be 62 followed
// by a non-zero digit and 8 to 13 digits in all.
func usableWhatsApp(phone string) bool {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := b.String()
	switch {
	case strings.HasPrefix(d, "0"):
		d = "62" + d[1:]
	case strings.HasPrefix(d, "8"):
		d = "62" + d
	}
	if !strings.HasPrefix(d, "62") || len(d) < 2+8 || len(d) > 2+13 {
		return false
	}
	return d[2] != '0'
}
