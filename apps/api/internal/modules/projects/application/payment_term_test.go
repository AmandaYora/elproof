package application

import (
	"testing"
	"time"

	"elproof/internal/shared/apperror"
)

func TestBuildTermsReportsEveryRowProblem(t *testing.T) {
	due := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	_, _, err := buildTerms(7, []PaymentTermInput{
		{Label: "DP", Amount: 5_000_000, DueDate: due},
		{Label: " ", Amount: 0, DueDate: due},                // label and amount wrong
		{Label: "Pelunasan", Amount: 3_000_000},              // no date
		{Label: "Tambahan", Amount: 1_000_000, DueDate: due}, // fine
	})
	ae, ok := apperror.As(err)
	if !ok || ae.Kind != apperror.KindValidation {
		t.Fatalf("want a validation error, got %v", err)
	}
	for _, key := range []string{"terms[1].label", "terms[1].amount", "terms[2].dueDate"} {
		if len(ae.Fields[key]) == 0 {
			t.Errorf("missing field error %s in %v", key, ae.Fields)
		}
	}
	if len(ae.Fields) != 3 {
		t.Errorf("only the broken rows are reported (rows 0 and 3 are fine): %v", ae.Fields)
	}
}

func TestBuildTermsAcceptsGoodRows(t *testing.T) {
	due := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	terms, total, err := buildTerms(7, []PaymentTermInput{{Label: " DP ", Amount: 2_000_000, DueDate: due}, {Label: "Pelunasan", Amount: 3_000_000, DueDate: due}})
	if err != nil || total != 5_000_000 || len(terms) != 2 || terms[0].Label != "DP" {
		t.Fatalf("terms=%v total=%d err=%v", terms, total, err)
	}
}
