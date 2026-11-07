package application

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

// ContractReader gives the payment services the amount a payment is checked
// against (what the client, a vendor or the venue is owed). Everything it
// reads belongs to this module.
type ContractReader interface {
	// ProjectContractValue is the project's contract value with the client.
	ProjectContractValue(ctx context.Context, tenantID, projectID int64) (int64, error)
	// VenueCost is the venue's rental price plus charge on the project (0 when none).
	VenueCost(ctx context.Context, tenantID, projectID int64) (int64, error)
	// EngagementContractValue is the agreed price of one vendor engagement.
	EngagementContractValue(ctx context.Context, projectID, projectVendorID int64) (int64, error)
}

type contractReader struct {
	projects *ProjectService
	vendors  VendorEngagementRepository
}

// NewContractReader builds the ContractReader the payment services share.
func NewContractReader(projects *ProjectService, vendors VendorEngagementRepository) ContractReader {
	return &contractReader{projects: projects, vendors: vendors}
}

func (c *contractReader) ProjectContractValue(ctx context.Context, tenantID, projectID int64) (int64, error) {
	p, err := c.projects.Get(ctx, tenantID, projectID)
	if err != nil {
		return 0, err
	}
	return p.ContractValue, nil
}

func (c *contractReader) VenueCost(ctx context.Context, tenantID, projectID int64) (int64, error) {
	p, err := c.projects.Get(ctx, tenantID, projectID)
	if err != nil {
		return 0, err
	}
	var cost int64
	if p.VenueRentalPrice != nil {
		cost += *p.VenueRentalPrice
	}
	if p.VenueCharge != nil {
		cost += *p.VenueCharge
	}
	return cost, nil
}

func (c *contractReader) EngagementContractValue(ctx context.Context, projectID, projectVendorID int64) (int64, error) {
	pv, err := c.vendors.FindByID(ctx, projectID, projectVendorID)
	if err != nil {
		return 0, err
	}
	if pv == nil {
		return 0, apperror.NotFound("Kerja sama vendor tidak ditemukan")
	}
	return pv.ContractValue, nil
}

// formatRupiah renders 1500000 as "Rp1.500.000".
func formatRupiah(n int64) string {
	digits := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, ch := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(ch)
	}
	return "Rp" + b.String()
}

// overpaymentError is the 422 returned when a change would take what is paid
// past what is owed. `owed` names the amount ("nilai kontrak", "harga vendor").
func overpaymentError(excess int64, owed string, removing bool) error {
	msg := fmt.Sprintf(
		"Pembayaran melebihi %s sebesar %s. Kurangi nominalnya, atau pilih jenis Tambahan jika ini biaya di luar kontrak.",
		owed, formatRupiah(excess),
	)
	if removing {
		msg = fmt.Sprintf(
			"Menghapus pembayaran ini membuat total melebihi %s sebesar %s. Turunkan atau hapus pembayaran lain lebih dulu.",
			owed, formatRupiah(excess),
		)
	}
	return apperror.Validation(msg, map[string][]string{"amount": {msg}}).WithCode("payment_exceeds_contract")
}

// checkAmount rejects an amount the ledgers cannot hold: zero, negative or
// absurdly large (which could wrap a sum).
func checkAmount(amount int64) error {
	if amount <= 0 || amount > domain.MaxMoney {
		msg := "Nominal harus lebih dari 0 dan wajar (maksimal Rp1.000.000.000.000.000)"
		return apperror.Validation(msg, map[string][]string{"amount": {msg}})
	}
	return nil
}

// checkChange applies the shared rule to one add, edit or delete.
func checkChange(before, after []domain.PaymentLine, contractValue int64, owed string, removing bool) error {
	if excess := domain.ExcessAfterChange(before, after, contractValue); excess > 0 {
		return overpaymentError(excess, owed, removing)
	}
	return nil
}
