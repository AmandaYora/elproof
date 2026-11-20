package application

import (
	"context"
	"fmt"

	"elproof/internal/shared/apperror"
)

// Messages and codes of the catalog's reference rules (PLAN §9 item 6).
const (
	MsgPackageInQuotation  = "Paket ini dipakai di penawaran. Nonaktifkan saja agar tidak bisa dipilih lagi."
	CodePackageInQuotation = "package_in_quotation"
)

// SetQuotationUsage lets the catalog ask about quotations (wired in
// offers.NewModule): a package a quotation refers to can only be
// deactivated, and the directory's delete checks quotation suppliers too.
func (s *PackageService) SetQuotationUsage(u QuotationUsage) { s.usage = u }

// SetQuotationUsage lets event kinds count the quotations that use them.
func (s *EventKindService) SetQuotationUsage(u QuotationUsage) { s.usage = u }

// packageInQuotation refuses the permanent delete of a package a quotation
// still refers to.
func (s *PackageService) packageInQuotation(ctx context.Context, tenantID, id int64) error {
	if s.usage == nil {
		return nil
	}
	used, err := s.usage.PackageReferenced(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if used {
		return apperror.Validation(MsgPackageInQuotation, map[string][]string{"id": {MsgPackageInQuotation}}).WithCode(CodePackageInQuotation)
	}
	return nil
}

// orQuotation is a package check OR the same check on quotations.
func (s *PackageService) orQuotation(ctx context.Context, inPackages func() (bool, error), inQuotations func(QuotationUsage) (bool, error)) (bool, error) {
	used, err := inPackages()
	if err != nil || used || s.usage == nil {
		return used, err
	}
	return inQuotations(s.usage)
}

// quotationCounts is how many quotations use each kind (empty before wiring).
func (s *EventKindService) quotationCounts(ctx context.Context, tenantID int64) (map[int64]int, error) {
	if s.usage == nil {
		return map[int64]int{}, nil
	}
	return s.usage.EventKindQuotationCounts(ctx, tenantID)
}

func inUseMessage(name string, packages, quotations int) string {
	switch {
	case packages > 0 && quotations > 0:
		return fmt.Sprintf("Jenis acara %s dipakai di %d paket dan %d penawaran. Nonaktifkan saja agar tidak bisa dipilih lagi.", name, packages, quotations)
	case quotations > 0:
		return fmt.Sprintf("Jenis acara %s dipakai di %d penawaran. Nonaktifkan saja agar tidak bisa dipilih lagi.", name, quotations)
	}
	return fmt.Sprintf("Jenis acara %s dipakai di %d paket. Nonaktifkan saja agar tidak bisa dipilih lagi.", name, packages)
}

// CategoryNames resolves the directory's category names for the quotation
// screens (lines and suppliers).
func (s *QuotationService) CategoryNames(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error) {
	if s.directory == nil || len(ids) == 0 {
		return map[int64]string{}, nil
	}
	return s.directory.CategoryNames(ctx, tenantID, uniqueIDs(ids))
}
