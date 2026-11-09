package application

import (
	"sort"
	"strconv"

	"elproof/internal/shared/apperror"
)

// fieldErrors gathers validation problems from every part of one request so
// they can be answered together as ONE 422 (keys such as "venueId",
// "terms[2].amount", "evidence[0]"), instead of making the person fix and resend
// one problem at a time.
type fieldErrors struct {
	fields map[string][]string
	first  string
	issues int
}

func newFieldErrors() *fieldErrors { return &fieldErrors{fields: map[string][]string{}} }

// add records one problem under one or more keys (the same message under each);
// it counts as a single issue however many keys carry it.
func (f *fieldErrors) add(msg string, keys ...string) {
	for _, k := range keys {
		f.fields[k] = append(f.fields[k], msg)
	}
	if f.first == "" {
		f.first = msg
	}
	f.issues++
}

// mergeFields adds every key of m (in sorted order, so the summary message is
// the same on every run).
func (f *fieldErrors) mergeFields(m map[string][]string) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, msg := range m[k] {
			f.add(msg, k)
		}
	}
}

// mergeErr folds a validation error's fields in and reports true; any other
// error (not a validation problem the caller can fix) is left for the caller
// to return as it is.
func (f *fieldErrors) mergeErr(err error) bool {
	if err == nil {
		return true
	}
	if ae, ok := apperror.As(err); ok && ae.Kind == apperror.KindValidation {
		if len(ae.Fields) == 0 {
			f.add(ae.Message, "_")
			return true
		}
		f.mergeFields(ae.Fields)
		return true
	}
	return false
}

func (f *fieldErrors) empty() bool { return len(f.fields) == 0 }

// err is the single 422: the first problem as the message, plus how many more
// there are, and every key with its messages.
func (f *fieldErrors) err() error {
	if f.empty() {
		return nil
	}
	msg := f.first
	if f.issues > 1 {
		msg += " Ada " + strconv.Itoa(f.issues-1) + " isian lain yang perlu diperbaiki."
	}
	return apperror.Validation(msg, f.fields)
}
