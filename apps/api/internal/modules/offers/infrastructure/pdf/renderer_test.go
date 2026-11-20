package pdf

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"elproof/internal/modules/offers/application"
	"elproof/internal/modules/offers/domain"
)

func i64(v int64) *int64 { return &v }
func pct(v int) *int     { return &v }
func str(v string) *string {
	return &v
}

// snapshot is a revision as the service freezes it: a reception at a venue,
// a package, items, a bonus, a discount, a client-paid vendor with an agreed
// cost, 30/40/30 terms, and terms text.
func snapshot(isPKP bool, items int) domain.Snapshot {
	venue := 0
	pkg := 0
	ev := 0
	lines := []domain.SnapshotLine{
		{Kind: domain.LinePackage, Description: "Paket Resepsi Gold", Amount: 120_000_000, EventIndex: &ev,
			PackageRule: &domain.PackageRule{Name: "Paket Resepsi Gold", Price: 120_000_000, IncludedGuests: pct(500)}},
		{Kind: domain.LineComponent, ParentIndex: &pkg, Description: "Dekorasi pelaminan"},
		{Kind: domain.LineComponent, ParentIndex: &pkg, Description: "Katering 500 porsi"},
	}
	var gross int64 = 120_000_000
	for i := 0; i < items; i++ {
		amount := int64(1_000_000 + i*25_000)
		lines = append(lines, domain.SnapshotLine{Kind: domain.LineItem, Description: fmt.Sprintf("Item tambahan %d", i+1),
			Details: "Termasuk pemasangan dan pembongkaran di lokasi acara", Quantity: i64(1), Unit: str("paket"), UnitPrice: i64(amount), Amount: amount})
		gross += amount
	}
	lines = append(lines,
		domain.SnapshotLine{Kind: domain.LineItem, Description: "Photo booth", IsBonus: true},
		domain.SnapshotLine{Kind: domain.LineDiscount, Description: "Potongan pameran", Amount: 2_000_000},
	)
	totals := domain.Totals{Gross: gross, Discount: 2_000_000, Subtotal: gross - 2_000_000, Total: gross - 2_000_000}
	if isPKP {
		totals.Tax = totals.Subtotal * 11 / 100
		totals.Total = totals.Subtotal + totals.Tax
	}
	amounts, _ := domain.TermAmounts(domain.DefaultTerms(), totals.Total)
	var terms []domain.SnapshotTerm
	for i, t := range domain.DefaultTerms() {
		terms = append(terms, domain.SnapshotTerm{Label: t.Label, Percent: t.Percent, DueBasis: t.DueBasis, DueOffsetDays: t.DueOffsetDays, Computed: amounts[i]})
	}
	return domain.Snapshot{
		SchemaVersion: domain.SnapshotSchemaVersion, Number: "AMR-PNW-261120-001", RevisionN: 1,
		IssuedAt: time.Date(2026, 11, 20, 3, 0, 0, 0, time.UTC), ValidUntil: "2026-12-04", PICName: "Sari",
		Workspace: domain.WorkspaceIdentity{Name: "Amora Wedding", Address: "Jl. Merdeka 10", City: "Bandung", WhatsApp: "081234567890",
			BankName: "BCA", BankAccountNumber: "1234567890", BankAccountHolder: "PT Amora"},
		CategoryNames: map[int64]string{},
		Content: domain.Content{
			CoupleNames: "Rina & Bayu", IsPKP: isPKP,
			Events: []domain.SnapshotEvent{{EventKindID: 1, KindName: "Resepsi", EventDate: str("2027-03-14"), StartTime: str("11:00"),
				EndTime: str("14:00"), VenueIndex: &venue, Layout: domain.LayoutSeated, Sessions: []domain.SnapshotSession{{Label: "Sesi 1", Guests: 450}}}},
			Suppliers: []domain.SnapshotSupplier{
				{Kind: domain.SupplierVenue, Name: "Gedung Serbaguna", PaidBy: domain.PayerWO, ShowName: true},
				{Kind: domain.SupplierVendor, Name: "Katering Sedap", PaidBy: domain.PayerClient, ShowName: true, AgreedCost: i64(87_654_321)},
			},
			Lines: lines, Terms: terms, Totals: totals,
			TermsText: "DP tidak dapat dikembalikan.\nPerubahan jumlah tamu paling lambat H-14.",
		},
	}
}

// withCoreFont renders in Helvetica so the content streams can be searched as text: the embedded Plus Jakarta Sans
// writes glyph ids instead. What is printed does not depend on the font.
func withCoreFont() Option { return func(r *Renderer) { r.coreFont = true } }

func render(t testing.TB, snap domain.Snapshot, draft bool, opts ...Option) []byte {
	t.Helper()
	data, err := NewRenderer(opts...).Render(application.BuildDocument(snap, draft, "Revisi 1", nil, ""))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return data
}

func TestPDFContent(t *testing.T) {
	data := render(t, snapshot(true, 3), false, WithoutCompression(), withCoreFont())
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("not a PDF")
	}
	for _, want := range []string{
		"Amora Wedding",               // letterhead
		"AMR-PNW-261120-001",          // number
		"Rina & Bayu",                 // for whom
		"Termin pembayaran",           // terms
		"Pelunasan",                   // a term
		"Syarat & ketentuan",          // T&C heading
		"DP tidak dapat dikembalikan", // T&C text
		"PPN",                         // PKP
		"Dibuat dengan ElProof",       // footer
		"Katering Sedap",              // a client-paid vendor, by name
	} {
		if !bytes.Contains(data, []byte(want)) {
			t.Errorf("PDF misses %q", want)
		}
	}
	for _, secret := range []string{"87.654.321", "87654321"} {
		if bytes.Contains(data, []byte(secret)) {
			t.Fatalf("PDF shows an agreed cost (%s)", secret)
		}
	}
	if bytes.Contains(data, []byte("(DRAF)")) {
		t.Fatal("a revision carries no DRAF watermark")
	}
}

func TestPDFWithoutPKPHasNoPPN(t *testing.T) {
	data := render(t, snapshot(false, 3), false, WithoutCompression(), withCoreFont())
	if bytes.Contains(data, []byte("PPN")) {
		t.Fatal("PPN printed for a workspace that is not PKP")
	}
}

func TestPreviewPDFIsMarkedDraft(t *testing.T) {
	data := render(t, snapshot(true, 3), true, WithoutCompression(), withCoreFont())
	if !bytes.Contains(data, []byte("DRAF")) {
		t.Fatal("the preview misses the DRAF watermark")
	}
}

func TestPDFIsSetInPlusJakartaSans(t *testing.T) {
	data := render(t, snapshot(true, 3), false, WithoutCompression())
	for _, want := range []string{"/FontName /utf8plusjakartasans\n", "/FontName /utf8plusjakartasansB", "/FontFile2"} {
		if !bytes.Contains(data, []byte(want)) {
			t.Errorf("PDF does not embed %q", want)
		}
	}
	if bytes.Contains(data, []byte("Helvetica")) {
		t.Error("PDF still uses Helvetica")
	}
}

func TestPDFPrintsLettersOutsideCp1252(t *testing.T) {
	snap := snapshot(true, 3)
	snap.Content.CoupleNames = "Đặng & Łucja ✓" // a core font would print these as "."
	data := render(t, snap, false, WithoutCompression())
	if bytes.Contains(data, []byte("Ng & ")) || bytes.Contains(data, []byte("(.")) {
		t.Fatal("letters outside cp1252 were replaced")
	}
}

func TestPDFIsDeterministic(t *testing.T) {
	a := render(t, snapshot(true, 5), false)
	time.Sleep(1100 * time.Millisecond) // a clock-based date would differ now
	b := render(t, snapshot(true, 5), false)
	if !bytes.Equal(a, b) {
		t.Fatal("the same revision must render to the same bytes")
	}
}

func TestFortyLinePDFStaysSmall(t *testing.T) {
	data := render(t, snapshot(true, 40), false)
	if len(data) >= 30<<20 {
		t.Fatalf("40-line PDF is %d bytes, budget 30 MB", len(data))
	}
	t.Logf("40-line PDF: %d bytes", len(data))
}

func BenchmarkRenderFortyLines(b *testing.B) {
	doc := application.BuildDocument(snapshot(true, 40), false, "Revisi 1", nil, "")
	r := NewRenderer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := r.Render(doc); err != nil {
			b.Fatal(err)
		}
	}
}
