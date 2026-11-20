package contracts

import "elproof/internal/modules/offers/domain"

// TermAmountsOf is what each frozen term asks for of a contract value, by the
// same rule as the quotation (percent rounded down, the rounding left on the
// last percent term when the percents add up to 100 with no fixed amount).
// "Jadikan proyek" uses it on the new project's own contract total.
func TermAmountsOf(terms []SnapshotTerm, total int64) (amounts []int64, mismatch bool) {
	in := make([]domain.QuotationTerm, len(terms))
	for i, t := range terms {
		in[i] = domain.QuotationTerm{Label: t.Label, Percent: t.Percent, Amount: t.Amount, DueBasis: t.DueBasis, DueOffsetDays: t.DueOffsetDays}
	}
	return domain.TermAmounts(in, total)
}
