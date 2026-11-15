package application

import (
	"context"
	"fmt"

	"elproof/internal/modules/projects/domain"
)

// checkEngagementMoney adds the problems of a vendor engagement's agreed price
// (Nilai Kerja Sama) and DP: both within 0..MaxMoney, and the DP never above
// the agreed price (QA EP-04). An agreed price of 0 means "not set yet" -- the
// same rule the payment ledger uses (domain.ExcessAfterChange) -- so it limits
// nothing.
func checkEngagementMoney(fe *fieldErrors, contract, dp int64) {
	if contract < 0 || contract > domain.MaxMoney {
		fe.add("Nilai kerja sama harus 0 atau lebih dan wajar", "contractValue")
	}
	if dp < 0 || dp > domain.MaxMoney {
		fe.add("Jumlah DP harus 0 atau lebih dan wajar", "dpAmount")
		return
	}
	if contract > 0 && contract <= domain.MaxMoney && dp > contract {
		fe.add(dpOverContractMessage(dp, contract), "dpAmount")
	}
}

func dpOverContractMessage(dp, contract int64) string {
	return fmt.Sprintf("Jumlah DP (%s) melebihi nilai kerja sama (%s). Turunkan DP atau naikkan nilai kerja sama.",
		formatRupiah(dp), formatRupiah(contract))
}

// EngagementMoneyErrors is checkEngagementMoney for the create handler, which
// reports every problem of the body in one 422 (nil when there is none).
func EngagementMoneyErrors(contract, dp int64) map[string][]string {
	fe := newFieldErrors()
	checkEngagementMoney(fe, contract, dp)
	if fe.empty() {
		return nil
	}
	return fe.fields
}

// contractBelowPaid is the problem of setting an engagement's agreed price
// under what has already been paid towards it (DP + Termin + Pelunasan − Refund;
// Tambahan is outside the price), or "" when the change is fine. Like the
// payment ledger, only a change that makes things worse is refused: raising a
// price that is still below what was paid is allowed, and 0 limits nothing.
func contractBelowPaid(oldContract, newContract, paid int64) string {
	if newContract <= 0 || newContract >= paid {
		return ""
	}
	if oldContract > 0 && newContract >= oldContract {
		return ""
	}
	return fmt.Sprintf("Nilai kerja sama tidak bisa di bawah yang sudah dibayar ke vendor (%s). Naikkan nilainya, atau koreksi pembayarannya lebih dulu.",
		formatRupiah(paid))
}

// paidTowardsContract is what has been paid towards one engagement's agreed
// price, from its live payments.
func (s *VendorEngagementService) paidTowardsContract(ctx context.Context, projectID, projectVendorID int64) (int64, error) {
	if s.payments == nil { // only in unit tests that never record a payment
		return 0, nil
	}
	all, err := s.payments.ListByProject(ctx, projectID)
	if err != nil {
		return 0, err
	}
	var lines []domain.PaymentLine
	for _, p := range all {
		if p.ProjectVendorID == projectVendorID {
			lines = append(lines, domain.PaymentLine{Type: p.Type, Amount: p.Amount})
		}
	}
	return domain.AppliedToContract(lines), nil
}
