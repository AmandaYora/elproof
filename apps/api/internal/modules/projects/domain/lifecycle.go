package domain

import "fmt"

// StatusAction is one explicit step in a project's life. Status is changed
// only through these (FLOW-IDEAL 2.4), never by editing the project's fields:
// Draft -> Preparation -> Ready -> Completed, Cancelled from any open step
// (the separate cancel action), and Reopen to bring a finished or cancelled
// project back.
type StatusAction string

const (
	ActionStartPreparation StatusAction = "start_preparation"
	ActionMarkReady        StatusAction = "mark_ready"
	ActionComplete         StatusAction = "complete"
	ActionReopen           StatusAction = "reopen"
)

// IsStatusAction reports whether action is one of the known steps.
func IsStatusAction(action StatusAction) bool {
	switch action {
	case ActionStartPreparation, ActionMarkReady, ActionComplete, ActionReopen:
		return true
	}
	return false
}

// NextStatus returns the status an action leads to, or false when the action
// does not apply to the current status.
func NextStatus(current ProjectStatus, action StatusAction) (ProjectStatus, bool) {
	switch action {
	case ActionStartPreparation:
		if current == StatusDraft {
			return StatusPreparation, true
		}
	case ActionMarkReady:
		if current == StatusPreparation {
			return StatusReady, true
		}
	case ActionComplete:
		if current == StatusReady {
			return StatusCompleted, true
		}
	case ActionReopen:
		if current == StatusCompleted || current == StatusCancelled {
			return StatusPreparation, true
		}
	}
	return "", false
}

// ChecklistItem is one line of the closing checklist shown before a project
// is completed. It informs; it does not block (a WO may close with items
// still open and decide that knowingly).
type ChecklistItem struct {
	Key    string
	Label  string
	Done   bool
	Detail string
}

// SupplierBalance is what one active vendor or venue the WO pays is owed (its
// agreed cost) and what the WO has paid it.
type SupplierBalance struct {
	AgreedCost int64
	Payments   []PaymentLine
}

// ClosingInput is everything the closing checklist is computed from.
type ClosingInput struct {
	ContractValue  int64
	ClientPayments []PaymentLine
	// Suppliers are the active vendors and venues the WO pays that have an
	// agreed cost; a supplier the client pays is settled between them.
	Suppliers  []SupplierBalance
	OpenIssues int
}

func rupiah(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	digits := fmt.Sprintf("%d", n)
	out := make([]byte, 0, len(digits)+len(digits)/3+3)
	for i := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, digits[i])
	}
	s := "Rp" + string(out)
	if neg {
		s = "-" + s
	}
	return s
}

// BuildClosingChecklist computes the checklist from a project's money and issues.
func BuildClosingChecklist(in ClosingInput) []ChecklistItem {
	items := make([]ChecklistItem, 0, 4)

	clientLeft := in.ContractValue - AppliedToContract(in.ClientPayments)
	client := ChecklistItem{Key: "client_paid", Label: "Tagihan klien sudah lunas", Done: clientLeft <= 0}
	if !client.Done {
		client.Detail = "Masih ada sisa " + rupiah(clientLeft)
	}
	items = append(items, client)

	unpaid := 0
	var left int64
	for _, v := range in.Suppliers {
		if l := v.AgreedCost - AppliedToContract(v.Payments); l > 0 {
			unpaid++
			left += l
		}
	}
	suppliers := ChecklistItem{Key: "suppliers_paid", Label: "Pembayaran vendor & venue sudah tuntas", Done: unpaid == 0}
	if !suppliers.Done {
		suppliers.Detail = fmt.Sprintf("%d vendor & venue belum lunas, total sisa %s", unpaid, rupiah(left))
	}
	items = append(items, suppliers)

	issues := ChecklistItem{Key: "issues_closed", Label: "Semua kendala sudah ditutup", Done: in.OpenIssues == 0}
	if !issues.Done {
		issues.Detail = fmt.Sprintf("%d kendala masih terbuka", in.OpenIssues)
	}
	items = append(items, issues)

	return items
}
