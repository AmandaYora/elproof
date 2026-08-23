package application

import (
	"errors"
	"testing"
	"time"

	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/apperror"
)

// TestMapGatewayStatus locks decideOutcome's mapping (task 8.6/8.6b) — the
// only part of the reconciler testable without a database (§7.1).
func TestMapGatewayStatus(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	maxAge := 24 * time.Hour
	young := now.Add(-1 * time.Hour) // created 1 jam lalu, masih di bawah maxAge
	old := now.Add(-48 * time.Hour)  // created 48 jam lalu, lewat maxAge

	tests := []struct {
		name       string
		status     string
		statusErr  error
		createdAt  time.Time
		wantAction reconcileAction
		wantStatus domain.AppChargeStatus
	}{
		{"paid", gatewayStatusPaid, nil, young, actionMarkPaid, ""},
		{"expired", gatewayStatusExpired, nil, young, actionResolve, domain.AppChargeExpired},
		{"failed", gatewayStatusFailed, nil, young, actionResolve, domain.AppChargeFailed},
		{"refund jadi cancelled (ENUM tidak punya refund)", gatewayStatusRefund, nil, young, actionResolve, domain.AppChargeCancelled},
		{"unpaid masih muda dibiarkan", gatewayStatusUnpaid, nil, young, actionSkip, ""},
		{"unpaid lewat max-age jadi expired", gatewayStatusUnpaid, nil, old, actionResolve, domain.AppChargeExpired},
		{"404 (baris hantu A20) segera cancelled", "", apperror.NotFound("order_ref tidak ditemukan"), young, actionResolve, domain.AppChargeCancelled},
		{"404 tapi masih muda TETAP segera cancelled", "", apperror.NotFound("order_ref tidak ditemukan"), young, actionResolve, domain.AppChargeCancelled},
		{"galat panggilan biasa TIDAK PERNAH dipaksa selesai (D17)", "", errors.New("network timeout"), old, actionSkip, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decideOutcome(tt.status, tt.statusErr, tt.createdAt, now, maxAge)
			if got.action != tt.wantAction {
				t.Fatalf("action = %v, want %v", got.action, tt.wantAction)
			}
			if got.action == actionResolve && got.status != tt.wantStatus {
				t.Fatalf("status = %v, want %v", got.status, tt.wantStatus)
			}
		})
	}
}
