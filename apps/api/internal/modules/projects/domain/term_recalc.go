package domain

import (
	"errors"
	"math/big"
	"time"
)

// Why a schedule cannot be recalculated to a new contract value.
var (
	ErrRecalcNoContract = errors.New("contract value is not set")
	ErrRecalcNoTerms    = errors.New("schedule is empty")
	ErrRecalcAllPaid    = errors.New("every installment already has money on it")
	ErrRecalcTooSmall   = errors.New("contract value is below what is already paid")
)

// TermsTotal is the sum of the installments.
func TermsTotal(terms []PaymentTerm) int64 {
	var total int64
	for _, t := range terms {
		total += t.Amount
	}
	return total
}

// RecalculateTerms fits the schedule to a new contract value without touching
// money already received. Payments cover the installments earliest due date
// first (AllocateTerms), so the installments that have any money on them (paid
// or partly paid) are always the first ones: they keep their amount. What the
// contract leaves after them is spread over the installments with nothing paid
// yet, in proportion to their current amounts (so a 30/40/30 schedule keeps its
// shape), in whole rupiah, with the rounding remainder on the last one. Labels
// and due dates do not change. The result is in due-date order.
func RecalculateTerms(terms []PaymentTerm, applied, contractValue int64) ([]PaymentTerm, error) {
	if contractValue <= 0 {
		return nil, ErrRecalcNoContract
	}
	if len(terms) == 0 {
		return nil, ErrRecalcNoTerms
	}
	progress := AllocateTerms(terms, applied, time.Time{})
	var locked int64
	var open []int // indexes into progress of installments with nothing paid
	for i, p := range progress {
		if p.Paid > 0 {
			locked += p.Term.Amount
			continue
		}
		open = append(open, i)
	}
	if len(open) == 0 {
		return nil, ErrRecalcAllPaid
	}
	budget := contractValue - locked
	if budget < int64(len(open)) {
		return nil, ErrRecalcTooSmall
	}

	var weight int64
	for _, i := range open {
		weight += progress[i].Term.Amount
	}
	out := make([]PaymentTerm, len(progress))
	for i, p := range progress {
		out[i] = p.Term
	}
	// big.Int: budget × amount can pass int64 near MaxMoney.
	b, w := big.NewInt(budget), big.NewInt(weight)
	var given int64
	for k, i := range open {
		if k == len(open)-1 {
			out[i].Amount = budget - given
			break
		}
		share := budget / int64(len(open)) // equal split when the amounts give no shape
		if weight > 0 {
			share = new(big.Int).Div(new(big.Int).Mul(b, big.NewInt(progress[i].Term.Amount)), w).Int64()
		}
		// Every installment stays above zero, leaving at least 1 for each one after it.
		if share < 1 {
			share = 1
		}
		if room := budget - given - int64(len(open)-k-1); share > room {
			share = room
		}
		out[i].Amount = share
		given += share
	}
	return out, nil
}
