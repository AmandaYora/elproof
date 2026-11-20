package main

import (
	"context"

	offersapp "elproof/internal/modules/offers/application"
	offersdomain "elproof/internal/modules/offers/domain"
	"elproof/internal/modules/platform"
)

// quotationProfile adapts platform's business profile and Penawaran settings
// to offers' WorkspaceProfile port (ADR-0049: offers never imports platform).
type quotationProfile struct {
	platform *platform.Module
}

func (p quotationProfile) QuotationProfile(ctx context.Context, tenantID int64) (offersapp.QuotationProfile, error) {
	l, err := p.platform.LetterheadProfile(ctx, tenantID)
	if err != nil {
		return offersapp.QuotationProfile{}, err
	}
	return offersapp.QuotationProfile{
		DocumentCode: l.DocumentCode, IsPKP: l.IsPKP, DefaultValidityDays: l.DefaultValidityDays, DefaultTermsText: l.DefaultTermsText,
		Identity: offersdomain.WorkspaceIdentity{
			Name: l.BusinessName, LogoKey: l.LogoStoragePath, Address: l.Address, City: l.City, WhatsApp: l.WhatsApp,
			BankName: l.BankName, BankAccountNumber: l.BankAccountNumber, BankAccountHolder: l.BankAccountHolder,
		},
	}, nil
}
