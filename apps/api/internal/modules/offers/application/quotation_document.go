package application

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"elproof/internal/modules/offers/domain"
)

// QuotationDocument is a quotation PDF with every text already written
// (DESAIN §6.12): the renderer in infrastructure/pdf only lays it out. It
// never carries an agreed cost.
type QuotationDocument struct {
	// Draft prints the diagonal "DRAF" watermark (the preview).
	Draft      bool
	Letterhead DocLetterhead
	Title      string
	// NumberLine is "MO-PNW-261119-001 · Revisi 2 · 18 November 2026".
	NumberLine   string
	ValidityLine string
	ForLine      string
	Events       []DocEvent
	Summary      []DocAmount
	ContractTot  DocAmount
	Terms        []DocTerm
	TermsNote    string
	Packages     []DocPackage
	Items        []DocAmount
	Bonuses      []string
	Discounts    []DocAmount
	Totals       []DocAmount
	ClientPaid   []DocAmount
	ClientNote   string
	Conditions   []string
	BankLines    []string
	Closing      string
	SignLeft     DocSignature
	SignRight    DocSignature
	// FooterRef is "MO-PNW-261119-001 Rev. 2" (with "Halaman n dari m" and
	// "Dibuat dengan ElProof" added by the renderer).
	FooterRef string
	// CreatedAt fixes the PDF's own dates so the same revision always gives
	// the same bytes.
	CreatedAt time.Time
}

// DocLetterhead is the WO's letterhead.
type DocLetterhead struct {
	Name    string
	Contact string
	Address string
	// Logo is a PNG or JPEG ("PNG"/"JPG" in LogoType); nil when there is none.
	Logo     []byte
	LogoType string
}

// DocEvent is one row of the events table.
type DocEvent struct {
	Kind, Date, Venue, Layout, Guests string
}

// DocAmount is a label, an optional detail line, and an amount text.
type DocAmount struct {
	Label  string
	Detail string
	Amount string
}

// DocTerm is one installment.
type DocTerm struct {
	Label, Due, Amount string
}

// DocPackage is a package in the details: its price, what it covers, its
// contents grouped by category (never priced), and its extra guests.
type DocPackage struct {
	Title   string
	Amount  string
	Covers  string
	Groups  []DocGroup
	Extra   *DocAmount
	Details string
}

// DocGroup is a category of a package's contents.
type DocGroup struct {
	Heading string
	Items   []string
}

// DocSignature is one signature block.
type DocSignature struct {
	Heading, Name, Sub string
}

var monthNames = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
var dayNames = [...]string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}

// longDate is "14 Maret 2027".
func longDate(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), monthNames[t.Month()-1], t.Year())
}

// dayDate is "Sabtu, 14 Maret 2027".
func dayDate(t time.Time) string {
	return dayNames[t.Weekday()] + ", " + longDate(t)
}

func thousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// Rupiah is "Rp120.000.000" (a negative amount reads "-Rp5.000.000").
func Rupiah(n int64) string {
	if n < 0 {
		return "-Rp" + thousands(-n)
	}
	return "Rp" + thousands(n)
}

func layoutWord(l domain.Layout) string {
	if l == domain.LayoutSeated {
		return "Duduk"
	}
	return "Berdiri"
}

func parseDay(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil
	}
	return &t
}

// termDue is "saat penawaran diterima", "60 hari sebelum Resepsi", ...
func termDue(t domain.SnapshotTerm, events []domain.SnapshotEvent) string {
	if t.DueBasis == domain.DueOnAcceptance {
		if t.DueOffsetDays == 0 {
			return "saat penawaran diterima"
		}
		return fmt.Sprintf("%d hari setelah penawaran diterima", t.DueOffsetDays)
	}
	kind := "acara"
	switch {
	case t.DueEventIndex != nil && *t.DueEventIndex < len(events):
		kind = events[*t.DueEventIndex].KindName
	case len(events) > 0:
		kind = events[0].KindName
	}
	switch {
	case t.DueOffsetDays < 0:
		return fmt.Sprintf("%d hari sebelum %s", -t.DueOffsetDays, kind)
	case t.DueOffsetDays > 0:
		return fmt.Sprintf("%d hari setelah %s", t.DueOffsetDays, kind)
	}
	return "pada hari " + kind
}

func termLabel(t domain.SnapshotTerm) string {
	if t.Percent != nil {
		return fmt.Sprintf("%s %d%%", t.Label, *t.Percent)
	}
	return t.Label
}

// numberedConditions turns the terms & conditions text into numbered points
// (lines already numbered keep their number).
func numberedConditions(text string) []string {
	var out []string
	n := 0
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		n++
		if i := strings.IndexAny(line, ".)"); i > 0 && i <= 3 {
			if _, err := strconv.Atoi(line[:i]); err == nil {
				out = append(out, line)
				continue
			}
		}
		out = append(out, fmt.Sprintf("%d. %s", n, line))
	}
	return out
}

// BuildDocument writes the PDF texts of a snapshot. revisionLabel is
// "Revisi 2" (or "Pratinjau"); logo is the letterhead image already in PNG or
// JPEG ("" type: none).
func BuildDocument(snap domain.Snapshot, draft bool, revisionLabel string, logo []byte, logoType string) QuotationDocument {
	c := snap.Content
	id := snap.Workspace
	issued := snap.IssuedAt.In(domain.WIB)
	doc := QuotationDocument{
		Draft: draft, Title: "Penawaran", CreatedAt: snap.IssuedAt,
		Letterhead: DocLetterhead{Name: id.Name, Address: id.Address, Logo: logo, LogoType: logoType},
		NumberLine: snap.Number + " · " + revisionLabel + " · " + longDate(issued),
		ForLine:    "Untuk " + c.CoupleNames,
	}
	if logoType == "" {
		doc.Letterhead.Logo = nil
	}
	var contact []string
	if id.WhatsApp != "" {
		contact = append(contact, "WA "+id.WhatsApp)
	}
	if id.City != "" {
		contact = append(contact, id.City)
	}
	doc.Letterhead.Contact = strings.Join(contact, " · ")
	if d := parseDay(&snap.ValidUntil); d != nil {
		doc.ValidityLine = "Berlaku sampai " + longDate(*d)
	}
	if snap.PICName != "" {
		doc.ForLine += " · PIC " + snap.PICName
	}
	if draft {
		doc.FooterRef = snap.Number + " Pratinjau"
	} else {
		doc.FooterRef = fmt.Sprintf("%s Rev. %d", snap.Number, snap.RevisionN)
	}

	// Events.
	for _, e := range c.Events {
		de := DocEvent{Kind: e.KindName, Date: "Tanggal menyusul", Venue: "Venue menyusul", Layout: layoutWord(e.Layout)}
		if d := parseDay(e.EventDate); d != nil {
			de.Date = dayDate(*d)
		}
		if e.VenueIndex != nil && *e.VenueIndex < len(c.Suppliers) {
			de.Venue = c.Suppliers[*e.VenueIndex].Name
		}
		total := 0
		for _, s := range e.Sessions {
			total += s.Guests
		}
		if len(e.Sessions) > 1 {
			de.Guests = fmt.Sprintf("%d sesi · %s tamu", len(e.Sessions), thousands(int64(total)))
		} else {
			de.Guests = thousands(int64(total)) + " tamu"
		}
		doc.Events = append(doc.Events, de)
	}

	// Lines: details (page 2) and the summary (page 1).
	kindNames := map[int64]string{}
	for _, e := range c.Events {
		kindNames[e.EventKindID] = e.KindName
	}
	var itemsTotal int64
	pkgAt := map[int]int{}
	for i, l := range c.Lines {
		switch l.Kind {
		case domain.LinePackage:
			p := DocPackage{Title: l.Description, Amount: Rupiah(l.Amount), Details: l.Details}
			if l.PackageRule != nil {
				var covers []string
				for _, k := range l.PackageRule.EventKindIDs {
					if n, ok := kindNames[k]; ok {
						covers = append(covers, n)
					}
				}
				if len(covers) > 0 {
					p.Covers = "Termasuk: " + strings.Join(covers, ", ")
				}
				if l.PackageRule.IncludedGuests != nil {
					sep := " · "
					if p.Covers == "" {
						sep = "Termasuk: "
					}
					p.Covers += sep + thousands(int64(*l.PackageRule.IncludedGuests)) + " tamu"
				}
			}
			pkgAt[i] = len(doc.Packages)
			doc.Packages = append(doc.Packages, p)
			doc.Summary = append(doc.Summary, DocAmount{Label: l.Description, Amount: Rupiah(l.Amount)})
		case domain.LineComponent:
			if l.ParentIndex == nil {
				continue
			}
			pi, ok := pkgAt[*l.ParentIndex]
			if !ok {
				continue
			}
			heading := ""
			if l.CategoryID != nil {
				heading = snap.CategoryNames[*l.CategoryID]
			}
			p := &doc.Packages[pi]
			found := false
			for gi := range p.Groups {
				if p.Groups[gi].Heading == heading {
					p.Groups[gi].Items = append(p.Groups[gi].Items, l.Description)
					found = true
				}
			}
			if !found {
				p.Groups = append(p.Groups, DocGroup{Heading: heading, Items: []string{l.Description}})
			}
		case domain.LineExtraGuests:
			if l.ParentIndex == nil {
				continue
			}
			pi, ok := pkgAt[*l.ParentIndex]
			if !ok {
				continue
			}
			parent := c.Lines[*l.ParentIndex]
			extra := DocAmount{Label: l.Description, Amount: Rupiah(l.Amount)}
			if parent.PackageRule != nil && domain.HasExtraGuestPricing(*parent.PackageRule) {
				r := *parent.PackageRule
				guests := make([]domain.EventGuests, len(c.Events))
				for ei, e := range c.Events {
					n := 0
					for _, s := range e.Sessions {
						n += s.Guests
					}
					guests[ei] = domain.EventGuests{EventKindID: e.EventKindID, Guests: n}
				}
				res := domain.ExtraGuests(r, guests)
				if res.Over > 0 {
					extra.Detail = fmt.Sprintf("%s tamu - %s termasuk = %s · %d × %s tamu × %s", thousands(int64(res.Guests)),
						thousands(int64(*r.IncludedGuests)), thousands(int64(res.Over)), res.Steps, thousands(int64(*r.ExtraGuestStep)), Rupiah(*r.ExtraGuestStepPrice))
					doc.Summary = append(doc.Summary, DocAmount{Label: fmt.Sprintf("Tambahan tamu (%s tamu)", thousands(int64(res.Over))), Amount: Rupiah(l.Amount)})
				}
			}
			doc.Packages[pi].Extra = &extra
		case domain.LineItem:
			if l.IsBonus {
				doc.Bonuses = append(doc.Bonuses, l.Description)
				continue
			}
			item := DocAmount{Label: l.Description, Amount: Rupiah(l.Amount)}
			if l.Quantity != nil && l.UnitPrice != nil {
				unit := ""
				if l.Unit != nil {
					unit = " " + *l.Unit
				}
				item.Detail = fmt.Sprintf("%s%s × %s", thousands(*l.Quantity), unit, Rupiah(*l.UnitPrice))
			}
			doc.Items = append(doc.Items, item)
			itemsTotal += l.Amount
		case domain.LineDiscount:
			doc.Discounts = append(doc.Discounts, DocAmount{Label: l.Description, Amount: Rupiah(-l.Amount)})
		}
	}
	if itemsTotal > 0 {
		doc.Summary = append(doc.Summary, DocAmount{Label: "Item tambahan", Amount: Rupiah(itemsTotal)})
	}
	if c.Totals.Discount > 0 {
		doc.Summary = append(doc.Summary, DocAmount{Label: "Potongan", Amount: Rupiah(-c.Totals.Discount)})
	}
	doc.Totals = append(doc.Totals, DocAmount{Label: "Subtotal", Amount: Rupiah(c.Totals.Subtotal)})
	if c.IsPKP {
		doc.Summary = append(doc.Summary, DocAmount{Label: "PPN", Amount: Rupiah(c.Totals.Tax)})
		doc.Totals = append(doc.Totals, DocAmount{Label: "PPN 12% × 11/12", Amount: Rupiah(c.Totals.Tax)})
	}
	doc.ContractTot = DocAmount{Label: "Nilai kontrak", Amount: Rupiah(c.Totals.Total)}
	doc.Totals = append(doc.Totals, doc.ContractTot)

	for _, t := range c.Terms {
		doc.Terms = append(doc.Terms, DocTerm{Label: termLabel(t), Due: termDue(t, c.Events), Amount: Rupiah(t.Computed)})
	}

	for _, s := range c.Suppliers {
		if s.PaidBy != domain.PayerClient {
			continue
		}
		service := ""
		if s.Kind == domain.SupplierVenue {
			service = "Venue"
		} else if s.CategoryID != nil {
			service = snap.CategoryNames[*s.CategoryID]
		}
		doc.ClientPaid = append(doc.ClientPaid, DocAmount{Label: s.Name, Detail: service})
	}
	sort.SliceStable(doc.ClientPaid, func(i, j int) bool { return doc.ClientPaid[i].Label < doc.ClientPaid[j].Label })
	if len(doc.ClientPaid) > 0 {
		doc.ClientNote = "Dibayar langsung oleh klien ke vendor, tidak termasuk nilai kontrak."
	}
	doc.Conditions = numberedConditions(c.TermsText)
	if id.BankName != "" && id.BankAccountNumber != "" && id.BankAccountHolder != "" {
		doc.BankLines = []string{id.BankName + " · " + id.BankAccountNumber, "a.n. " + id.BankAccountHolder}
	}
	doc.Closing = "Terima kasih atas kepercayaan Anda. Kami siap membantu mewujudkan hari bahagia Anda."
	doc.SignLeft = DocSignature{Heading: "Hormat kami", Name: snap.PICName, Sub: id.Name}
	doc.SignRight = DocSignature{Heading: "Disetujui oleh", Name: c.CoupleNames, Sub: "Tanggal: ____________"}
	return doc
}
