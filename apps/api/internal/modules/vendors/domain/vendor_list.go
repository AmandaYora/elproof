package domain

import "elproof/internal/shared/apperror"

// VendorListFilter narrows GET /vendors (the paginated directory).
type VendorListFilter struct {
	CategoryID *int64
	Search     string
	City       string
	// IsActive keeps only active (true) or inactive (false) vendors; nil = both.
	IsActive *bool
}

// VendorRow is a vendor with the name of its category (this module owns both
// tables, so the list joins them).
type VendorRow struct {
	Vendor
	CategoryName string
}

// VenueListFilter narrows GET /venues.
type VenueListFilter struct {
	Search   string
	City     string
	IsActive *bool
}

// BulkAction is what POST /vendors/bulk and /venues/bulk apply to every id.
type BulkAction string

const (
	BulkActivate   BulkAction = "activate"
	BulkDeactivate BulkAction = "deactivate"
)

// MaxBulkIDs is how many ids one bulk request may carry.
const MaxBulkIDs = 100

// ParseBulkAction validates the action of a bulk request.
func ParseBulkAction(raw string) (BulkAction, bool) {
	switch BulkAction(raw) {
	case BulkActivate, BulkDeactivate:
		return BulkAction(raw), true
	}
	return "", false
}

// Target is the is_active value the action sets.
func (a BulkAction) Target() bool { return a == BulkActivate }

// BulkResult is what a bulk action did: how many rows changed, which ids already
// were in the requested state (nothing to do) and which are not this workspace's
// (or do not exist). The two id lists are never nil, so they encode as [].
type BulkResult struct {
	Updated   int     `json:"updated"`
	Unchanged []int64 `json:"unchanged"`
	NotFound  []int64 `json:"notFound"`
}

// NormalizeBulkIDs validates and deduplicates the ids of a bulk request,
// keeping the order of first appearance. A request carries 1 to MaxBulkIDs ids
// (counted as sent, duplicates included: a body of a thousand repeats is a
// mistake, not a selection), every one a positive integer.
func NormalizeBulkIDs(ids []int64) ([]int64, error) {
	if len(ids) < 1 || len(ids) > MaxBulkIDs {
		return nil, apperror.Validation("Daftar id tidak valid", map[string][]string{"ids": {"Pilih 1 sampai 100 data"}})
	}
	seen := make(map[int64]bool, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id < 1 {
			return nil, apperror.Validation("Daftar id tidak valid", map[string][]string{"ids": {"Setiap id harus berupa bilangan bulat positif"}})
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out, nil
}

// PlanBulk decides what a bulk action does to the requested ids, given the
// current is_active of the ids that exist in the workspace: ids missing from
// `current` are notFound, ids already at target are unchanged, the rest are to
// be updated. Every list keeps the order of `requested`.
func PlanBulk(requested []int64, current map[int64]bool, target bool) (toUpdate, unchanged, notFound []int64) {
	toUpdate, unchanged, notFound = []int64{}, []int64{}, []int64{}
	for _, id := range requested {
		active, exists := current[id]
		switch {
		case !exists:
			notFound = append(notFound, id)
		case active == target:
			unchanged = append(unchanged, id)
		default:
			toUpdate = append(toUpdate, id)
		}
	}
	return toUpdate, unchanged, notFound
}
