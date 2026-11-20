package application

import (
	"context"
	"strings"
	"unicode/utf8"

	"elproof/internal/modules/platform/domain"
	"elproof/internal/shared/apperror"
)

// QuoteSettingsInput is the Owner's "Penawaran" section of Profil usaha.
type QuoteSettingsInput struct {
	DocumentCode        string
	IsPKP               bool
	DefaultValidityDays int
	DefaultTermsText    string
}

// GetQuoteSettings returns the workspace's Penawaran settings. A workspace that
// has none yet gets them now: the document code from its business name, a
// 14-day validity, and the example terms. Reading twice never changes a code the
// Owner already chose, and quotation numbers already issued keep theirs.
func (s *TenantService) GetQuoteSettings(ctx context.Context, tenantID int64) (domain.QuoteSettings, error) {
	q, found, err := s.repo.GetQuoteSettings(ctx, tenantID)
	if err != nil || found {
		return q, err
	}
	tenant, err := s.Get(ctx, tenantID)
	if err != nil {
		return domain.QuoteSettings{}, err
	}
	q = domain.QuoteSettings{
		DocumentCode:        domain.DocumentCodeFromName(tenant.BusinessName),
		DefaultValidityDays: domain.DefaultQuoteValidityDays,
		DefaultTermsText:    domain.DefaultQuoteTerms,
	}
	if err := s.repo.SaveQuoteSettings(ctx, tenantID, q); err != nil {
		return domain.QuoteSettings{}, err
	}
	return q, nil
}

// SaveQuoteSettings stores the Owner's choices. The document code only shapes
// numbers created from now on (PLAN §5.1); it need not be unique across
// workspaces, because a number never leaves its own WO.
func (s *TenantService) SaveQuoteSettings(ctx context.Context, tenantID int64, in QuoteSettingsInput) (domain.QuoteSettings, error) {
	q := domain.QuoteSettings{
		DocumentCode:        domain.NormalizeDocumentCode(in.DocumentCode),
		IsPKP:               in.IsPKP,
		DefaultValidityDays: in.DefaultValidityDays,
		DefaultTermsText:    strings.TrimSpace(in.DefaultTermsText),
	}
	fields := map[string][]string{}
	if !domain.ValidDocumentCode(q.DocumentCode) {
		fields["documentCode"] = []string{"Kode usaha 2–5 huruf atau angka, contoh MO"}
	}
	if q.DefaultValidityDays < 1 || q.DefaultValidityDays > domain.MaxQuoteValidityDays {
		fields["defaultValidityDays"] = []string{"Masa berlaku 1–365 hari"}
	}
	if utf8.RuneCountInString(q.DefaultTermsText) > domain.MaxDefaultTermsTextLen {
		fields["defaultTermsText"] = []string{"Syarat & ketentuan maksimal 5.000 karakter"}
	}
	if len(fields) > 0 {
		return domain.QuoteSettings{}, apperror.Validation("Pengaturan penawaran belum benar", fields)
	}
	if err := s.repo.SaveQuoteSettings(ctx, tenantID, q); err != nil {
		return domain.QuoteSettings{}, err
	}
	return q, nil
}
