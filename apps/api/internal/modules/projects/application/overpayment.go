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
// against (what the client or a supplier is owed). Everything it reads belongs
// to this module.
type ContractReader interface {
	// ProjectContractValue is the project's contract value with the client.
	ProjectContractValue(ctx context.Context, tenantID, projectID int64) (int64, error)
	// Supplier is one live vendor or venue of the project (a not found when there
	// is none): its payer and the cost the WO agreed to pay it.
	Supplier(ctx context.Context, projectID, projectSupplierID int64) (*domain.ProjectSupplier, error)
}

type contractReader struct {
	projects  *ProjectService
	suppliers SupplierRepository
}

// NewContractReader builds the ContractReader the payment services share.
func NewContractReader(projects *ProjectService, suppliers SupplierRepository) ContractReader {
	return &contractReader{projects: projects, suppliers: suppliers}
}

func (c *contractReader) ProjectContractValue(ctx context.Context, tenantID, projectID int64) (int64, error) {
	p, err := c.projects.Get(ctx, tenantID, projectID)
	if err != nil {
		return 0, err
	}
	return p.ContractTotal, nil
}

func (c *contractReader) Supplier(ctx context.Context, projectID, projectSupplierID int64) (*domain.ProjectSupplier, error) {
	s, err := c.suppliers.FindByID(ctx, projectID, projectSupplierID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, errSupplierNotFound
	}
	return s, nil
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
