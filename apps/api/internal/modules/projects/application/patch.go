package application

import (
	"context"
	"strings"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/etag"
)

// Partial PATCH support (ADR-0036). A patch carries only what the caller sent:
// every field is an Opt, so "left out" and "sent as the zero value" are
// different things, and a service applies it onto the row it just loaded, checks
// the business rules against the merged result, and writes exactly the fields
// that changed.

// Opt is one optional field of a patch: absent, sent as null, or sent with a value.
type Opt[T any] struct {
	set  bool
	null bool
	val  T
}

// Some is a field sent with a value.
func Some[T any](v T) Opt[T] { return Opt[T]{set: true, val: v} }

// Null is a field sent as null (clear it, where the field allows it).
func Null[T any]() Opt[T] { return Opt[T]{set: true, null: true} }

// Present: the field was sent (with a value or as null).
func (o Opt[T]) Present() bool { return o.set }

// IsNull: the field was sent as null.
func (o Opt[T]) IsNull() bool { return o.set && o.null }

// Value is the sent value (the zero value when absent or null).
func (o Opt[T]) Value() T { return o.val }

// changeSet records which fields a patch really changed: the keys (the
// repository writes exactly these columns) and their Indonesian labels (the
// activity entry names them).
type changeSet struct {
	keys   []string
	labels []string
}

func (c *changeSet) note(key, label string) {
	c.keys = append(c.keys, key)
	if label == "" {
		return
	}
	for _, l := range c.labels {
		if l == label {
			return
		}
	}
	c.labels = append(c.labels, label)
}

func (c *changeSet) any() bool { return len(c.keys) > 0 }

func (c *changeSet) has(key string) bool {
	for _, k := range c.keys {
		if k == key {
			return true
		}
	}
	return false
}

// summary is "a, b, c": the labels of what changed.
func (c *changeSet) summary() string { return strings.Join(c.labels, ", ") }

const dayLayout = "2006-01-02"

// setVal applies a sent value to a field that cannot be null.
func setVal[T comparable](c *changeSet, key, label string, dst *T, o Opt[T]) {
	if !o.Present() || o.IsNull() {
		return
	}
	if *dst != o.Value() {
		*dst = o.Value()
		c.note(key, label)
	}
}

// setNullable applies a sent value (or null) to a nullable pointer field.
func setNullable[T comparable](c *changeSet, key, label string, dst **T, o Opt[T]) {
	if !o.Present() {
		return
	}
	if o.IsNull() {
		if *dst != nil {
			*dst = nil
			c.note(key, label)
		}
		return
	}
	if *dst == nil || **dst != o.Value() {
		v := o.Value()
		*dst = &v
		c.note(key, label)
	}
}

// setDate applies a sent day to a required date field (compared by calendar day).
func setDate(c *changeSet, key, label string, dst *time.Time, o Opt[time.Time]) {
	if !o.Present() || o.IsNull() {
		return
	}
	if dst.Format(dayLayout) != o.Value().Format(dayLayout) {
		*dst = o.Value()
		c.note(key, label)
	}
}

// setNullableDate applies a sent day (or null) to an optional date field.
func setNullableDate(c *changeSet, key, label string, dst **time.Time, o Opt[time.Time]) {
	if !o.Present() {
		return
	}
	if o.IsNull() {
		if *dst != nil {
			*dst = nil
			c.note(key, label)
		}
		return
	}
	if *dst == nil || (*dst).Format(dayLayout) != o.Value().Format(dayLayout) {
		v := o.Value()
		*dst = &v
		c.note(key, label)
	}
}

// finishPatch is what every PATCH does after its write statement. wrote is the
// repository's answer (see infrastructure.patchRow): true when the row was written
// (always true without If-Match). The row is read again, so the answer carries the
// version MySQL just stamped. A write that did not match is told apart by that
// read: the row is gone (not found), still at the version the caller sent
// (MySQL had nothing to change: fine), or at another version (stale: 409 with the
// current row).
func finishPatch[T any](ctx context.Context, expected *time.Time, wrote bool, reload func(context.Context) (*T, error),
	version func(*T) time.Time, missing error) (*T, error) {
	cur, err := reload(ctx)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, missing
	}
	if !wrote && expected != nil && !etag.Same(version(cur), *expected) {
		return nil, apperror.Stale(cur)
	}
	return cur, nil
}

// requiredText reports a sent required text field that is blank.
func requiredText(fe *fieldErrors, key, msg string, o Opt[string]) {
	if o.Present() && !o.IsNull() && strings.TrimSpace(o.Value()) == "" {
		fe.add(msg, key)
	}
}

// PaymentPatch is what PATCH on a client, vendor or venue payment sent
// (ADR-0036). A vendor payment's engagement is not patchable (delete and record
// again). Notes null clears to "".
type PaymentPatch struct {
	Type                           Opt[domain.PaymentType]
	Amount                         Opt[int64]
	PaymentDate                    Opt[time.Time]
	Method, ReferenceNumber, Notes Opt[string]
}

// checkPaymentPatch validates the sent fields (the same rules as recording one).
func checkPaymentPatch(patch PaymentPatch) error {
	fe := newFieldErrors()
	if patch.Type.Present() && !domain.IsValidPaymentType(patch.Type.Value()) {
		fe.add("Pilih jenis pembayaran: DP, Termin, Pelunasan, Tambahan, atau Refund.", "type")
	}
	if patch.Amount.Present() {
		if err := checkAmount(patch.Amount.Value()); err != nil {
			fe.mergeErr(err)
		}
	}
	return fe.err()
}

// applyPayment applies the patch's fields onto the fields of one payment row.
func applyPayment(ch *changeSet, patch PaymentPatch, t *domain.PaymentType, amount *int64, date *time.Time, method, reference, notes *string) {
	setVal(ch, "type", "jenis", t, patch.Type)
	setVal(ch, "amount", "nominal", amount, patch.Amount)
	setDate(ch, "paymentDate", "tanggal", date, patch.PaymentDate)
	setVal(ch, "method", "metode", method, patch.Method)
	setVal(ch, "referenceNumber", "nomor referensi", reference, patch.ReferenceNumber)
	setVal(ch, "notes", "catatan", notes, patch.Notes)
}

// moneyChanged: the change touches what the overpayment rule looks at.
func (c *changeSet) moneyChanged() bool { return c.has("type") || c.has("amount") }

// refuseIfBehind: with If-Match, a change that would be judged on a row that has
// already moved past the version the caller saw is answered as stale (409 with the
// current row) before any rule runs, so the caller never gets a validation message
// about data it has not seen. A patch that changes nothing is not subject to it
// (nothing can be lost). current is the row as it was loaded, before the patch
// was applied to it.
func refuseIfBehind[T any](ch *changeSet, expected *time.Time, loaded time.Time, current *T) error {
	if ch.any() && expected != nil && !etag.Same(loaded, *expected) {
		return apperror.Stale(current)
	}
	return nil
}
