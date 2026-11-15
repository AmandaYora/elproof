package presentation

import (
	"testing"
	"time"

	"elproof/internal/modules/platform/domain"
)

// EP-09: the CSV says "Dibekukan" for a frozen tenant, even when its paid
// period is still running, and the console's word for every other status.
func TestExportStatus(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	ends := now.AddDate(0, 0, 20)
	paid := domain.Tenant{SubscriptionExpiresAt: &ends}
	frozen := paid
	frozen.IsSuspended = true
	p := domain.DefaultSubscriptionPolicy

	if got := exportStatus(domain.ComputeLifecycle(paid, now, p)); got != "Aktif" {
		t.Errorf("paid tenant: %q, want Aktif", got)
	}
	if got := exportStatus(domain.ComputeLifecycle(frozen, now, p)); got != "Dibekukan" {
		t.Errorf("frozen paid tenant: %q, want Dibekukan", got)
	}
	if got := exportStatus(domain.ComputeLifecycle(domain.Tenant{}, now, p)); got != "Belum Berlangganan" {
		t.Errorf("no period: %q, want Belum Berlangganan", got)
	}
	for status := range statusLabels {
		if statusLabels[status] == "" {
			t.Errorf("status %q has no label", status)
		}
	}
}
