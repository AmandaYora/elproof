// Package domain holds the offers module's entities (event kinds, packages)
// and the pure pricing and capacity rules shared by quotations and projects
// (docs/plan/paket-wo/PLAN.md §2.1). Nothing here reads the database.
package domain

import "time"

// EventKind is one kind of event a WO organises (Akad, Resepsi, …). Events in
// quotations and projects copy its name, so renaming a kind never rewrites an
// earlier document.
type EventKind struct {
	ID       int64
	TenantID int64
	Name     string
	// DefaultKey names which built-in kind this row was seeded as ("akad",
	// "resepsi", …), or "" for a kind the WO added. It survives a rename, so
	// seeding stays idempotent and the partner reference prices of a vendor
	// (Akad / Resepsi / Akad + Resepsi) can still be matched to the kind.
	DefaultKey string
	SortOrder  int
	IsActive   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// MaxEventKindNameLen is the width of event_kinds.name.
const MaxEventKindNameLen = 60

// DefaultEventKind is one of the kinds every WO starts with.
type DefaultEventKind struct {
	Key  string
	Name string
}

// Keys of the built-in kinds that the vendor directory prices refer to.
const (
	EventKindAkad    = "akad"
	EventKindResepsi = "resepsi"
)

// DefaultEventKinds is the starting list of every WO, in display order
// (PLAN §8 Tahap 1). A WO may rename or deactivate them; one that is in use
// cannot be deleted.
var DefaultEventKinds = []DefaultEventKind{
	{"lamaran", "Lamaran"},
	{"pengajian", "Pengajian"},
	{"siraman", "Siraman"},
	{EventKindAkad, "Akad"},
	{EventKindResepsi, "Resepsi"},
	{"ngunduh_mantu", "Ngunduh mantu"},
}
