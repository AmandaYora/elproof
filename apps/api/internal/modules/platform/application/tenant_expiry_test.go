package application

import (
	"testing"
	"time"

	billingcontracts "elproof/internal/modules/billing/contracts"
	"elproof/internal/modules/platform/domain"
)

func TestComputeNewExpiryAt(t *testing.T) {
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	future := now.AddDate(0, 2, 0)
	past := now.AddDate(0, -1, 0)
	trialEnd := now.AddDate(0, 0, 10)

	cases := []struct {
		name   string
		tenant domain.Tenant
		want   time.Time
	}{
		{"renewal stacks on remaining paid time", domain.Tenant{SubscriptionExpiresAt: &future}, future.AddDate(1, 0, 0)},
		{"lapsed subscription restarts now", domain.Tenant{SubscriptionExpiresAt: &past}, now.AddDate(1, 0, 0)},
		{"paying during trial starts after the trial", domain.Tenant{TrialEndsAt: &trialEnd}, trialEnd.AddDate(1, 0, 0)},
		{"paid period wins over an old trial", domain.Tenant{SubscriptionExpiresAt: &past, TrialEndsAt: &trialEnd}, now.AddDate(1, 0, 0)},
		{"legacy tenant starts now", domain.Tenant{}, now.AddDate(1, 0, 0)},
	}
	for _, c := range cases {
		if got := computeNewExpiryAt(&c.tenant, 12, now); !got.Equal(c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestTxTypeFor(t *testing.T) {
	paid := time.Now()
	if txTypeFor(&domain.Tenant{}) != billingcontracts.TransactionNew {
		t.Error("never-paid tenant must record a new subscription")
	}
	if txTypeFor(&domain.Tenant{SubscriptionExpiresAt: &paid}) != billingcontracts.TransactionRenewal {
		t.Error("previously paid tenant must record a renewal")
	}
}
