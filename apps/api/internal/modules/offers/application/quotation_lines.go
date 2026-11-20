package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"elproof/internal/modules/offers/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/etag"
)

// Messages of a quotation's lines and terms.
const (
	MsgLinesInvalid        = "Rincian belum sesuai"
	MsgTooManyLines        = "Rincian paling banyak 300 baris"
	MsgLineDescription     = "Isi uraian baris"
	MsgPackageInactive     = "Paket ini nonaktif. Pilih paket lain atau aktifkan dulu di Katalog."
	MsgComponentTopLevel   = "Isi paket dan tambahan tamu hanya ada di dalam paketnya"
	MsgLineKindUnknown     = "Jenis baris tidak dikenal"
	MsgQuantityRange       = "Jumlah minimal 1"
	MsgUnitPriceRange      = "Harga satuan tidak boleh kurang dari 0"
	MsgDiscountAmount      = "Isi nominal potongan, lebih dari 0"
	MsgLineTooLarge        = "Nominal baris terlalu besar"
	MsgLineEventUnknown    = "Acara tidak ditemukan di penawaran ini"
	MsgLineSupplierUnknown = "Vendor atau venue tidak ditemukan di penawaran ini"
	MsgLineSupplierClient  = "Pemasok baris ini harus vendor atau venue yang dibayar WO"
	MsgPackageNoEventFmt   = "Paket ini mencakup %s. Tambahkan acaranya dulu."
	MsgDiscountExceedsFmt  = "Potongan melebihi subtotal (%s). Kurangi potongan."
	MsgTermsRequired       = "Tambahkan minimal satu termin"
	MsgTooManyTerms        = "Termin paling banyak 12"
	MsgTermLabel           = "Isi nama termin"
	MsgTermValue           = "Isi persen atau nominal termin (salah satu)"
	MsgTermPercentRange    = "Persen termin antara 1 dan 100"
	MsgTermAmountRange     = "Nominal termin harus lebih dari 0"
	MsgTermsPercentOver    = "Jumlah persen termin melebihi 100%"
	MsgTermDueBasis        = "Pilih jatuh tempo: saat diterima atau dari tanggal acara"
	MsgTermOffsetRange     = "Jatuh tempo terlalu jauh"
	MsgTermEventUnknown    = "Acara tidak ditemukan di penawaran ini"
	ExtraGuestsDescription = "Tambahan tamu"
)

// QuotationComponentInput is one "Isi paket" row of a package line.
type QuotationComponentInput struct {
	Description string
	Details     string
	CategoryID  *int64
}

// QuotationLineInput is one top-level line as sent ("Simpan rincian").
// Package lines carry their components; extra guests lines are never sent:
// the server computes them.
type QuotationLineInput struct {
	// ID is the line's id when it was loaded: a package line keeps the copy
	// of the package rule it took, unless RefreshFromCatalog.
	ID                 int64
	Kind               domain.LineKind
	PackageID          *int64
	RefreshFromCatalog bool
	// Components nil = keep the package line's (or copy the catalog's for a
	// new package line).
	Components  []QuotationComponentInput
	Description string
	Details     string
	CategoryID  *int64
	EventID     *int64
	Quantity    *int64
	Unit        *string
	UnitPrice   *int64
	Amount      *int64
	IsBonus     bool
	SupplierRef *int64
}

// LinesResult is lines as they will be stored, with the totals and how each
// package's extra guests were counted (by row position).
type LinesResult struct {
	Rows        []LineRow
	Totals      domain.Totals
	ExtraGuests map[int]domain.ExtraGuestsResult
}

func linesErr(err error, gross int64) error {
	switch {
	case errors.Is(err, domain.ErrDiscountExceedsSubtotal):
		msg := fmt.Sprintf(MsgDiscountExceedsFmt, Rupiah(gross))
		return apperror.Validation(msg, map[string][]string{"lines": {msg}}).WithCode("discount_exceeds_subtotal")
	case errors.Is(err, domain.ErrNegativeAmount), errors.Is(err, domain.ErrUnknownLineKind):
		return apperror.Validation(MsgLinesInvalid, map[string][]string{"lines": {MsgLinesInvalid}})
	}
	return err
}

func kindNames(ctx context.Context, kinds EventKindRepository, tenantID int64) (map[int64]string, error) {
	list, err := kinds.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]string, len(list))
	for _, k := range list {
		out[k.ID] = k.Name
	}
	return out, nil
}

// buildLines validates the lines sent against the working copy and turns
// them into rows: each package followed by its components and its computed
// extra guests line.
func (s *QuotationService) buildLines(ctx context.Context, tenantID int64, w *domain.WorkingCopy, inputs []QuotationLineInput) (*LinesResult, error) {
	f := newFieldErrs()
	now := s.now()
	existing := map[int64]domain.QuotationLine{}
	children := map[int64][]domain.QuotationLine{}
	for _, l := range domain.OrderedLines(w.Lines) {
		existing[l.ID] = l
		if l.ParentLineID != nil && l.Kind == domain.LineComponent {
			children[*l.ParentLineID] = append(children[*l.ParentLineID], l)
		}
	}
	events := map[int64]bool{}
	haveKind := map[int64]bool{}
	for _, e := range w.Events {
		events[e.ID] = true
		haveKind[e.EventKindID] = true
	}
	suppliers := map[int64]domain.QuotationSupplier{}
	for _, sup := range w.Suppliers {
		suppliers[sup.ID] = sup
	}
	guests := domain.EventGuestsOf(w.Events)
	var names map[int64]string
	categoryOK := map[int64]bool{}
	checkCategory := func(field string, id *int64) error {
		if id == nil {
			return nil
		}
		ok, seen := categoryOK[*id]
		if !seen {
			var err error
			if ok, err = s.directory.CategoryExists(ctx, tenantID, *id); err != nil {
				return err
			}
			categoryOK[*id] = ok
		}
		if !ok {
			f.add(field, MsgCategoryNotFound)
		}
		return nil
	}
	text := func(field, v string, max int, required bool, msg string) string {
		v = strings.TrimSpace(v)
		if required && v == "" {
			f.add(field, msg)
		}
		if utf8.RuneCountInString(v) > max {
			f.add(field, fmt.Sprintf("Maksimal %d karakter", max))
		}
		return v
	}

	res := &LinesResult{ExtraGuests: map[int]domain.ExtraGuestsResult{}}
	for i, in := range inputs {
		p := fmt.Sprintf("lines.%d.", i)
		line := domain.QuotationLine{QuotationID: w.Quotation.ID, Kind: in.Kind, SortOrder: i}
		line.Details = text(p+"details", in.Details, domain.MaxLineDetailsLen, false, "")
		if in.EventID != nil {
			if !events[*in.EventID] {
				f.add(p+"eventId", MsgLineEventUnknown)
			} else {
				id := *in.EventID
				line.EventID = &id
			}
		}
		if in.SupplierRef != nil {
			sup, ok := suppliers[*in.SupplierRef]
			switch {
			case !ok:
				f.add(p+"supplierRef", MsgLineSupplierUnknown)
			case sup.PaidBy != domain.PayerWO:
				f.add(p+"supplierRef", MsgLineSupplierClient)
			default:
				id := sup.ID
				line.SupplierRef = &id
			}
		}
		if err := checkCategory(p+"categoryId", in.CategoryID); err != nil {
			return nil, err
		}
		line.CategoryID = in.CategoryID

		switch in.Kind {
		case domain.LinePackage:
			var comps []domain.QuotationLine
			old, kept := existing[in.ID]
			if kept && old.Kind == domain.LinePackage && old.PackageRule != nil && !in.RefreshFromCatalog &&
				(in.PackageID == nil || old.PackageID == nil || *in.PackageID == *old.PackageID) {
				rule := *old.PackageRule
				line.PackageRule, line.PackageID, line.CopiedAt = &rule, old.PackageID, old.CopiedAt
				comps = children[old.ID]
			} else {
				if in.PackageID == nil || *in.PackageID <= 0 {
					f.add(p+"packageId", "Pilih paket")
					continue
				}
				pkg, err := s.packages.FindByID(ctx, tenantID, *in.PackageID)
				if err != nil {
					return nil, err
				}
				if pkg == nil {
					f.add(p+"packageId", MsgPackageNotFound)
					continue
				}
				newPick := !kept || old.PackageID == nil || *old.PackageID != pkg.ID
				if newPick && !pkg.IsActive {
					f.add(p+"packageId", MsgPackageInactive)
				}
				rule := pkg.Rule()
				at := now
				id := pkg.ID
				line.PackageRule, line.PackageID, line.CopiedAt = &rule, &id, &at
				for _, c := range pkg.Components {
					comps = append(comps, domain.QuotationLine{Kind: domain.LineComponent, Description: c.Description, CategoryID: c.CategoryID})
				}
			}
			rule := *line.PackageRule
			covered := false
			for _, k := range rule.EventKindIDs {
				if haveKind[k] {
					covered = true
				}
			}
			if !covered {
				if names == nil {
					var err error
					if names, err = kindNames(ctx, s.kinds, tenantID); err != nil {
						return nil, err
					}
				}
				var kn []string
				for _, k := range rule.EventKindIDs {
					if n, ok := names[k]; ok {
						kn = append(kn, n)
					}
				}
				f.add(p+"packageId", fmt.Sprintf(MsgPackageNoEventFmt, strings.Join(kn, ", ")))
			}
			line.Description = text(p+"description", in.Description, domain.MaxLineDescriptionLen, false, "")
			if line.Description == "" {
				line.Description = rule.Name
			}
			line.Amount = rule.Price
			if in.Components != nil {
				comps = comps[:0:0]
				for j, c := range in.Components {
					cp := fmt.Sprintf("%scomponents.%d.", p, j)
					cl := domain.QuotationLine{Kind: domain.LineComponent}
					cl.Description = text(cp+"description", c.Description, domain.MaxLineDescriptionLen, true, MsgComponentText)
					cl.Details = text(cp+"details", c.Details, domain.MaxLineDetailsLen, false, "")
					if err := checkCategory(cp+"categoryId", c.CategoryID); err != nil {
						return nil, err
					}
					cl.CategoryID = c.CategoryID
					comps = append(comps, cl)
				}
			}
			if len(comps) > domain.MaxPackageComponents {
				f.add(p+"components", MsgTooManyComponents)
			}
			parent := len(res.Rows)
			res.Rows = append(res.Rows, LineRow{Line: line, Parent: -1})
			for j, c := range comps {
				c.ID, c.ParentLineID, c.Amount, c.SortOrder, c.QuotationID = 0, nil, 0, j, w.Quotation.ID
				c.EventID, c.SupplierRef, c.PackageRule, c.PackageID = nil, nil, nil, nil
				res.Rows = append(res.Rows, LineRow{Line: c, Parent: parent})
			}
			if domain.HasExtraGuestPricing(rule) {
				extra := domain.ExtraGuests(rule, guests)
				res.ExtraGuests[parent] = extra
				res.Rows = append(res.Rows, LineRow{Line: domain.QuotationLine{QuotationID: w.Quotation.ID, Kind: domain.LineExtraGuests,
					Description: ExtraGuestsDescription, Amount: extra.Amount, SortOrder: len(comps)}, Parent: parent})
			}
			continue
		case domain.LineItem:
			line.Description = text(p+"description", in.Description, domain.MaxLineDescriptionLen, true, MsgLineDescription)
			qty := int64(1)
			if in.Quantity != nil {
				qty = *in.Quantity
			}
			if qty < 1 || qty > domain.MaxQuantity {
				f.add(p+"quantity", MsgQuantityRange)
			}
			price := int64(0)
			if in.UnitPrice != nil {
				price = *in.UnitPrice
			}
			if price < 0 || price > domain.MaxMoney {
				f.add(p+"unitPrice", MsgUnitPriceRange)
			}
			line.IsBonus = in.IsBonus
			if line.IsBonus {
				price = 0
			}
			if in.Unit != nil {
				u := text(p+"unit", *in.Unit, 20, false, "")
				if u != "" {
					line.Unit = &u
				}
			}
			line.Quantity, line.UnitPrice = &qty, &price
			if qty > 0 && price > 0 && price > domain.MaxMoney/qty {
				f.add(p+"unitPrice", MsgLineTooLarge)
			} else {
				line.Amount = qty * price
			}
		case domain.LineDiscount:
			line.Description = text(p+"description", in.Description, domain.MaxLineDescriptionLen, false, "")
			if line.Description == "" {
				line.Description = "Potongan"
			}
			if in.Amount == nil || *in.Amount <= 0 {
				f.add(p+"amount", MsgDiscountAmount)
			} else if *in.Amount > domain.MaxMoney {
				f.add(p+"amount", MsgLineTooLarge)
			} else {
				line.Amount = *in.Amount
			}
			line.SupplierRef = nil
		case domain.LineComponent, domain.LineExtraGuests:
			f.add(p+"kind", MsgComponentTopLevel)
			continue
		default:
			f.add(p+"kind", MsgLineKindUnknown)
			continue
		}
		res.Rows = append(res.Rows, LineRow{Line: line, Parent: -1})
	}
	if len(res.Rows) > domain.MaxQuotationLines {
		f.add("lines", MsgTooManyLines)
	}
	if err := f.err(); err != nil {
		return nil, err
	}
	lines := make([]domain.QuotationLine, len(res.Rows))
	for i, r := range res.Rows {
		lines[i] = r.Line
	}
	totals, err := domain.TotalsOf(lines, w.Quotation.IsPKP)
	if err != nil {
		var gross int64
		for _, l := range lines {
			if l.Kind != domain.LineDiscount {
				gross += l.Amount
			}
		}
		return nil, linesErr(err, gross)
	}
	res.Totals = totals
	return res, nil
}

// PreviewLines prices lines that are still being typed, with the same rules
// as saving, without writing anything (PLAN §2.2 "Total sementara").
func (s *QuotationService) PreviewLines(ctx context.Context, tenantID, quotationID int64, v Viewer, inputs []QuotationLineInput) (*LinesResult, error) {
	w, err := s.Content(ctx, tenantID, quotationID, v)
	if err != nil {
		return nil, err
	}
	return s.buildLines(ctx, tenantID, w, inputs)
}

// SaveLines replaces the lines ("Simpan rincian"; If-Match: linesUpdatedAt).
func (s *QuotationService) SaveLines(ctx context.Context, tenantID, quotationID int64, v Viewer, inputs []QuotationLineInput, expected *time.Time) (*domain.WorkingCopy, error) {
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		q, err := s.lock(ctx, tenantID, quotationID, v)
		if err != nil {
			return err
		}
		if expected != nil && !etag.Same(*expected, q.LinesUpdatedAt) {
			w, err := s.workingCopy(ctx, tenantID, q)
			if err != nil {
				return err
			}
			return apperror.Stale(*w)
		}
		if err := editable(q); err != nil {
			return err
		}
		w, err := s.workingCopy(ctx, tenantID, q)
		if err != nil {
			return err
		}
		res, err := s.buildLines(ctx, tenantID, w, inputs)
		if err != nil {
			return err
		}
		if err := s.repo.ReplaceLines(ctx, tenantID, quotationID, res.Rows); err != nil {
			return err
		}
		if err := s.repo.SetCaches(ctx, tenantID, quotationID, res.Totals.Total, domain.MainEventDate(w.Events), true, false); err != nil {
			return err
		}
		return s.activity(ctx, tenantID, quotationID, v, domain.ActivityLinesChanged, "Rincian disimpan · Nilai kontrak "+Rupiah(res.Totals.Total))
	})
	if err != nil {
		return nil, err
	}
	return s.Content(ctx, tenantID, quotationID, Viewer{AllQuotations: true, CanSeeCost: v.CanSeeCost})
}

// QuotationTermInput is one term as sent.
type QuotationTermInput struct {
	Label         string
	Percent       *int
	Amount        *int64
	DueBasis      domain.DueBasis
	DueOffsetDays int
	DueEventID    *int64
}

func buildTerms(w *domain.WorkingCopy, inputs []QuotationTermInput) ([]domain.QuotationTerm, error) {
	f := newFieldErrs()
	switch {
	case len(inputs) == 0:
		f.add("terms", MsgTermsRequired)
	case len(inputs) > domain.MaxQuotationTerms:
		f.add("terms", MsgTooManyTerms)
	}
	events := map[int64]bool{}
	for _, e := range w.Events {
		events[e.ID] = true
	}
	sumPercent := 0
	out := make([]domain.QuotationTerm, 0, len(inputs))
	for i, in := range inputs {
		p := fmt.Sprintf("terms.%d.", i)
		t := domain.QuotationTerm{QuotationID: w.Quotation.ID, SortOrder: i, DueBasis: in.DueBasis, DueOffsetDays: in.DueOffsetDays}
		t.Label = strings.TrimSpace(in.Label)
		if t.Label == "" {
			f.add(p+"label", MsgTermLabel)
		} else if utf8.RuneCountInString(t.Label) > domain.MaxTermLabelLen {
			f.add(p+"label", fmt.Sprintf("Nama termin maksimal %d karakter", domain.MaxTermLabelLen))
		}
		switch {
		case (in.Percent == nil) == (in.Amount == nil):
			f.add(p+"percent", MsgTermValue)
		case in.Percent != nil:
			if *in.Percent < 1 || *in.Percent > 100 {
				f.add(p+"percent", MsgTermPercentRange)
			}
			v := *in.Percent
			t.Percent = &v
			sumPercent += v
		default:
			if *in.Amount < 1 || *in.Amount > domain.MaxMoney {
				f.add(p+"amount", MsgTermAmountRange)
			}
			v := *in.Amount
			t.Amount = &v
		}
		if t.DueBasis == "" {
			t.DueBasis = domain.DueOnAcceptance
		}
		if !domain.ValidDueBasis(t.DueBasis) {
			f.add(p+"dueBasis", MsgTermDueBasis)
		}
		if t.DueBasis == domain.DueOnAcceptance && (t.DueOffsetDays < 0 || t.DueOffsetDays > 365) {
			f.add(p+"dueOffsetDays", MsgTermOffsetRange)
		}
		if t.DueBasis == domain.DueOnEvent && (t.DueOffsetDays < -730 || t.DueOffsetDays > 365) {
			f.add(p+"dueOffsetDays", MsgTermOffsetRange)
		}
		if in.DueEventID != nil && t.DueBasis == domain.DueOnEvent {
			if !events[*in.DueEventID] {
				f.add(p+"dueEventId", MsgTermEventUnknown)
			} else {
				v := *in.DueEventID
				t.DueEventID = &v
			}
		}
		out = append(out, t)
	}
	if sumPercent > 100 {
		f.add("terms", MsgTermsPercentOver)
	}
	if err := f.err(); err != nil {
		return nil, err
	}
	return out, nil
}

// SaveTerms replaces the payment terms (If-Match: termsUpdatedAt).
func (s *QuotationService) SaveTerms(ctx context.Context, tenantID, quotationID int64, v Viewer, inputs []QuotationTermInput, expected *time.Time) (*domain.WorkingCopy, error) {
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		q, err := s.lock(ctx, tenantID, quotationID, v)
		if err != nil {
			return err
		}
		w, err := s.workingCopy(ctx, tenantID, q)
		if err != nil {
			return err
		}
		if expected != nil && !etag.Same(*expected, q.TermsUpdatedAt) {
			return apperror.Stale(*w)
		}
		if err := editable(q); err != nil {
			return err
		}
		terms, err := buildTerms(w, inputs)
		if err != nil {
			return err
		}
		if err := s.repo.ReplaceTerms(ctx, tenantID, quotationID, terms); err != nil {
			return err
		}
		if err := s.repo.SetCaches(ctx, tenantID, quotationID, q.ContractTotal, domain.MainEventDate(w.Events), false, true); err != nil {
			return err
		}
		return s.activity(ctx, tenantID, quotationID, v, domain.ActivityTermsChanged, "Termin diubah")
	})
	if err != nil {
		return nil, err
	}
	return s.Content(ctx, tenantID, quotationID, Viewer{AllQuotations: true, CanSeeCost: v.CanSeeCost})
}
