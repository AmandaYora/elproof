package application

import (
	"context"
	"testing"

	"elproof/internal/modules/platform/domain"
)

type quoteRepo struct {
	TenantRepository
	tenant *domain.Tenant
	stored *domain.QuoteSettings
	saves  int
}

func (r *quoteRepo) FindByID(_ context.Context, _ int64) (*domain.Tenant, error) {
	return r.tenant, nil
}

func (r *quoteRepo) GetQuoteSettings(_ context.Context, _ int64) (domain.QuoteSettings, bool, error) {
	if r.stored == nil {
		return domain.QuoteSettings{}, false, nil
	}
	return *r.stored, r.stored.DocumentCode != "", nil
}

func (r *quoteRepo) SaveQuoteSettings(_ context.Context, _ int64, q domain.QuoteSettings) error {
	r.stored = &q
	r.saves++
	return nil
}

func TestGetQuoteSettingsFillsDefaultsOnce(t *testing.T) {
	repo := &quoteRepo{tenant: &domain.Tenant{ID: 1, BusinessName: "Melati Organizer"}}
	s := &TenantService{repo: repo, access: newAccessCache(accessCacheTTL)}
	ctx := context.Background()

	q, err := s.GetQuoteSettings(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if q.DocumentCode != "MO" || q.DefaultValidityDays != 14 || q.IsPKP || q.DefaultTermsText == "" {
		t.Fatalf("defaults = %+v", q)
	}
	repo.tenant.BusinessName = "Bunga Wedding Organizer"
	if q, _ = s.GetQuoteSettings(ctx, 1); q.DocumentCode != "MO" || repo.saves != 1 {
		t.Fatalf("second read changed the code: %+v, saves %d", q, repo.saves)
	}
}

func TestSaveQuoteSettingsValidates(t *testing.T) {
	repo := &quoteRepo{}
	s := &TenantService{repo: repo}
	ctx := context.Background()

	q, err := s.SaveQuoteSettings(ctx, 1, QuoteSettingsInput{DocumentCode: " bwo ", IsPKP: true, DefaultValidityDays: 30, DefaultTermsText: "  1. DP hangus  "})
	if err != nil || q.DocumentCode != "BWO" || q.DefaultTermsText != "1. DP hangus" || !repo.stored.IsPKP {
		t.Fatalf("save = %+v, %v", q, err)
	}
	_, err = s.SaveQuoteSettings(ctx, 1, QuoteSettingsInput{DocumentCode: "M-O", DefaultValidityDays: 0})
	f := fieldsOf(err)
	if f["documentCode"] == nil || f["defaultValidityDays"] == nil {
		t.Fatalf("fields = %v", f)
	}
}
