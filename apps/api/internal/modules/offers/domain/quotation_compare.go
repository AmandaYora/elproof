package domain

import (
	"encoding/json"
	"fmt"
)

// ChangeKind says how an item differs between two revisions.
type ChangeKind string

const (
	ChangeAdded   ChangeKind = "added"
	ChangeChanged ChangeKind = "changed"
	ChangeRemoved ChangeKind = "removed"
)

// LineChange is one line that differs ("Bandingkan revisi", DESAIN §6.4).
type LineChange struct {
	Change ChangeKind
	Before *SnapshotLine
	After  *SnapshotLine
	// Parent is the description of the package a component belongs to.
	Parent string
}

// EventChange is one event that differs.
type EventChange struct {
	Change ChangeKind
	Before *SnapshotEvent
	After  *SnapshotEvent
}

// SupplierChange is one vendor or venue that differs.
type SupplierChange struct {
	Change ChangeKind
	Before *SnapshotSupplier
	After  *SnapshotSupplier
}

// Comparison is what changed from one revision to another.
type Comparison struct {
	Lines              []LineChange
	UnchangedLines     int
	Events             []EventChange
	UnchangedEvents    int
	Suppliers          []SupplierChange
	UnchangedSuppliers int
	TermsChanged       bool
	TermsTextChanged   bool
	HeaderChanged      bool
	TotalsBefore       Totals
	TotalsAfter        Totals
}

func same(a, b any) bool {
	ra, _ := json.Marshal(a)
	rb, _ := json.Marshal(b)
	return string(ra) == string(rb)
}

// keyed pairs items of two lists by a key, in order of occurrence: the n-th
// item with a key in one list meets the n-th with that key in the other.
func keyed(n int, key func(i int) string) map[string][]int {
	out := map[string][]int{}
	for i := 0; i < n; i++ {
		k := key(i)
		out[k] = append(out[k], i)
	}
	return out
}

func lineKey(c Content, i int) string {
	l := c.Lines[i]
	parent := ""
	if l.ParentIndex != nil && *l.ParentIndex < len(c.Lines) {
		parent = c.Lines[*l.ParentIndex].Description
	}
	return fmt.Sprintf("%s|%s|%s", l.Kind, parent, l.Description)
}

func parentOf(c Content, l SnapshotLine) string {
	if l.ParentIndex != nil && *l.ParentIndex < len(c.Lines) {
		return c.Lines[*l.ParentIndex].Description
	}
	return ""
}

// comparableLine is a line without its positions, which shift whenever a line
// is added above.
func comparableLine(c Content, l SnapshotLine) any {
	ev := ""
	if l.EventIndex != nil && *l.EventIndex < len(c.Events) {
		ev = c.Events[*l.EventIndex].KindName
	}
	sup := ""
	if l.SupplierIndex != nil && *l.SupplierIndex < len(c.Suppliers) {
		sup = c.Suppliers[*l.SupplierIndex].Name
	}
	l.ParentIndex, l.EventIndex, l.SupplierIndex = nil, nil, nil
	return struct {
		L   SnapshotLine
		Ev  string
		Sup string
	}{l, ev, sup}
}

func comparableEvent(c Content, e SnapshotEvent) any {
	venue := ""
	if e.VenueIndex != nil && *e.VenueIndex < len(c.Suppliers) {
		venue = c.Suppliers[*e.VenueIndex].Name
	}
	e.VenueIndex = nil
	return struct {
		E     SnapshotEvent
		Venue string
	}{e, venue}
}

// Compare lists what changed from before to after.
func Compare(before, after Content) Comparison {
	out := Comparison{TotalsBefore: before.Totals, TotalsAfter: after.Totals}

	bl := keyed(len(before.Lines), func(i int) string { return lineKey(before, i) })
	matched := map[int]bool{}
	seen := map[string]int{}
	for i, l := range after.Lines {
		k := lineKey(after, i)
		n := seen[k]
		seen[k]++
		a := after.Lines[i]
		if n < len(bl[k]) {
			bi := bl[k][n]
			matched[bi] = true
			b := before.Lines[bi]
			if same(comparableLine(before, b), comparableLine(after, l)) {
				out.UnchangedLines++
				continue
			}
			out.Lines = append(out.Lines, LineChange{Change: ChangeChanged, Before: &b, After: &a, Parent: parentOf(after, a)})
			continue
		}
		out.Lines = append(out.Lines, LineChange{Change: ChangeAdded, After: &a, Parent: parentOf(after, a)})
	}
	for i := range before.Lines {
		if !matched[i] {
			b := before.Lines[i]
			out.Lines = append(out.Lines, LineChange{Change: ChangeRemoved, Before: &b, Parent: parentOf(before, b)})
		}
	}

	be := keyed(len(before.Events), func(i int) string { return before.Events[i].KindName })
	matchedEv := map[int]bool{}
	seenEv := map[string]int{}
	for i := range after.Events {
		a := after.Events[i]
		n := seenEv[a.KindName]
		seenEv[a.KindName]++
		if n < len(be[a.KindName]) {
			bi := be[a.KindName][n]
			matchedEv[bi] = true
			b := before.Events[bi]
			if same(comparableEvent(before, b), comparableEvent(after, a)) {
				out.UnchangedEvents++
				continue
			}
			out.Events = append(out.Events, EventChange{Change: ChangeChanged, Before: &b, After: &a})
			continue
		}
		out.Events = append(out.Events, EventChange{Change: ChangeAdded, After: &a})
	}
	for i := range before.Events {
		if !matchedEv[i] {
			b := before.Events[i]
			out.Events = append(out.Events, EventChange{Change: ChangeRemoved, Before: &b})
		}
	}

	supKey := func(s SnapshotSupplier) string { return fmt.Sprintf("%s|%d", s.Kind, s.SupplierID) }
	bs := keyed(len(before.Suppliers), func(i int) string { return supKey(before.Suppliers[i]) })
	matchedSup := map[int]bool{}
	for i := range after.Suppliers {
		a := after.Suppliers[i]
		if idx := bs[supKey(a)]; len(idx) > 0 {
			matchedSup[idx[0]] = true
			b := before.Suppliers[idx[0]]
			if same(b, a) {
				out.UnchangedSuppliers++
				continue
			}
			out.Suppliers = append(out.Suppliers, SupplierChange{Change: ChangeChanged, Before: &b, After: &a})
			continue
		}
		out.Suppliers = append(out.Suppliers, SupplierChange{Change: ChangeAdded, After: &a})
	}
	for i := range before.Suppliers {
		if !matchedSup[i] {
			b := before.Suppliers[i]
			out.Suppliers = append(out.Suppliers, SupplierChange{Change: ChangeRemoved, Before: &b})
		}
	}

	out.TermsChanged = !same(before.Terms, after.Terms)
	out.TermsTextChanged = before.TermsText != after.TermsText
	out.HeaderChanged = before.CoupleNames != after.CoupleNames || before.ContactWhatsApp != after.ContactWhatsApp ||
		before.PICStaffID != after.PICStaffID || before.IsPKP != after.IsPKP
	return out
}
