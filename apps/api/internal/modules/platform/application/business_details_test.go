package application

import (
	"context"
	"testing"

	"elproof/internal/modules/platform/domain"
	"elproof/internal/shared/apperror"
)

type bizRepo struct {
	TenantRepository
	saved domain.BusinessDetails
	calls int
}

func (r *bizRepo) SaveBusinessDetails(_ context.Context, _ int64, d domain.BusinessDetails) error {
	r.saved = d
	r.calls++
	return nil
}

func fieldsOf(err error) map[string][]string {
	if ae, ok := apperror.As(err); ok {
		return ae.Fields
	}
	return nil
}

func TestSaveBusinessDetails(t *testing.T) {
	repo := &bizRepo{}
	s := &TenantService{repo: repo}
	ctx := context.Background()

	// Nothing set is fine: the Owner may share only the address, or nothing.
	if _, err := s.SaveBusinessDetails(ctx, 1, BusinessDetailsInput{Address: "Jl. Mawar 5, Bandung"}); err != nil {
		t.Fatalf("address only: %v", err)
	}

	// A complete account is saved trimmed.
	d, err := s.SaveBusinessDetails(ctx, 1, BusinessDetailsInput{BankName: " BCA ", BankAccountNumber: " 123 456 7890 ", BankAccountHolder: "  WO Cantik  "})
	if err != nil || d.BankName != "BCA" || d.BankAccountNumber != "123 456 7890" || d.BankAccountHolder != "WO Cantik" || !d.HasBank() {
		t.Fatalf("complete account: %+v %v", d, err)
	}

	// An account is all-or-nothing, and nothing is stored when it is not.
	before := repo.calls
	_, err = s.SaveBusinessDetails(ctx, 1, BusinessDetailsInput{BankAccountNumber: "1234567890"})
	f := fieldsOf(err)
	if f["bankName"] == nil || f["bankAccountHolder"] == nil || f["bankAccountNumber"] != nil {
		t.Fatalf("number without bank and holder: %v", f)
	}
	if repo.calls != before {
		t.Fatal("an invalid form must not be stored")
	}

	// Numbers must look like account numbers.
	for _, bad := range []string{"12", "abcd5678", "1234-", "1234567890123456789012345678901234567890123"} {
		_, err = s.SaveBusinessDetails(ctx, 1, BusinessDetailsInput{BankName: "BCA", BankAccountNumber: bad, BankAccountHolder: "WO"})
		if fieldsOf(err)["bankAccountNumber"] == nil {
			t.Errorf("%q should be refused", bad)
		}
	}
}
