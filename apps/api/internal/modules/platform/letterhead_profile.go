package platform

import "context"

// LetterheadProfile is what a quotation needs of the workspace: the
// Penawaran settings and the business profile its PDF prints (Tahap 5).
// Plain values so the reader (offers, through an adaptor in cmd/server)
// never imports platform.
type LetterheadProfile struct {
	DocumentCode        string
	IsPKP               bool
	DefaultValidityDays int
	DefaultTermsText    string
	BusinessName        string
	LogoStoragePath     *string
	Address             string
	City                string
	WhatsApp            string
	BankName            string
	BankAccountNumber   string
	BankAccountHolder   string
}

// LetterheadProfile reads a workspace's Penawaran settings (their defaults
// when never saved) and business profile.
func (m *Module) LetterheadProfile(ctx context.Context, tenantID int64) (LetterheadProfile, error) {
	q, err := m.tenantService.GetQuoteSettings(ctx, tenantID)
	if err != nil {
		return LetterheadProfile{}, err
	}
	t, err := m.tenantService.Get(ctx, tenantID)
	if err != nil {
		return LetterheadProfile{}, err
	}
	d, err := m.tenantService.GetBusinessDetails(ctx, tenantID)
	if err != nil {
		return LetterheadProfile{}, err
	}
	return LetterheadProfile{
		DocumentCode: q.DocumentCode, IsPKP: q.IsPKP, DefaultValidityDays: q.DefaultValidityDays, DefaultTermsText: q.DefaultTermsText,
		BusinessName: t.BusinessName, LogoStoragePath: t.LogoStoragePath, Address: d.Address, City: t.City, WhatsApp: t.Phone,
		BankName: d.BankName, BankAccountNumber: d.BankAccountNumber, BankAccountHolder: d.BankAccountHolder,
	}, nil
}
