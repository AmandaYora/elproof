// Package pdf lays out a quotation PDF (PLAN §5.5, DESAIN §6.12) with
// go-pdf/fpdf: pure Go, no browser, small memory. It is a technical utility:
// every text arrives already written in application.QuotationDocument, and
// nothing here decides what a quotation says.
package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"

	"elproof/internal/modules/offers/application"
)

// Renderer turns a QuotationDocument into PDF bytes. The same document always
// gives the same bytes: the PDF's dates come from the document, the catalog is
// sorted, and nothing reads the clock.
type Renderer struct {
	compress bool
	coreFont bool // Helvetica instead of the embedded Plus Jakarta Sans (tests only, see setupFonts)
}

// Option changes a Renderer.
type Option func(*Renderer)

// WithoutCompression writes plain content streams (tests read the text).
func WithoutCompression() Option { return func(r *Renderer) { r.compress = false } }

func NewRenderer(opts ...Option) *Renderer {
	r := &Renderer{compress: true}
	for _, o := range opts {
		o(r)
	}
	return r
}

var _ application.DocumentRenderer = (*Renderer)(nil)

// Page geometry (A4, millimetres).
const (
	pageW        = 210.0
	pageH        = 297.0
	margin       = 18.0
	bottomMargin = 20.0
	contentW     = pageW - 2*margin
	lineH        = 5.0
)

// Colours: ink, a muted grey for secondary text, hairlines, and the light
// band behind the contract value.
var (
	ink   = [3]int{33, 33, 38}
	muted = [3]int{110, 110, 120}
	rule  = [3]int{205, 205, 212}
	band  = [3]int{244, 242, 238}
	mark  = [3]int{200, 60, 60}
)

type writer struct {
	pdf   *fpdf.Fpdf
	fonts fontSet
	doc   application.QuotationDocument
}

func (w *writer) tr(s string) string { return w.fonts.translate(s) }

// split wraps already translated text to width. A core font's text is cp1252
// bytes, which fpdf's SplitText (rune based) would misread, so it is split by
// bytes; an embedded UTF-8 font is split by runes.
func (w *writer) split(text string, width float64) []string {
	if w.fonts.utf8 {
		return w.pdf.SplitText(text, width)
	}
	parts := w.pdf.SplitLines([]byte(text), width)
	lines := make([]string, len(parts))
	for i, p := range parts {
		lines[i] = string(p)
	}
	return lines
}

func (w *writer) font(style string, size float64) { w.pdf.SetFont(w.fonts.family, style, size) }

func (w *writer) color(c [3]int) { w.pdf.SetTextColor(c[0], c[1], c[2]) }

// ensure starts a new page when h more millimetres do not fit.
func (w *writer) ensure(h float64) {
	if w.pdf.GetY()+h > pageH-bottomMargin {
		w.pdf.AddPage()
	}
}

func (w *writer) hairline() {
	w.pdf.SetDrawColor(rule[0], rule[1], rule[2])
	w.pdf.SetLineWidth(0.2)
	y := w.pdf.GetY()
	w.pdf.Line(margin, y, pageW-margin, y)
}

func (w *writer) heading(text string) {
	w.ensure(14)
	w.pdf.Ln(4)
	w.font("B", 11)
	w.color(ink)
	w.pdf.CellFormat(contentW, 6, w.tr(text), "", 1, "L", false, 0, "")
	w.hairline()
	w.pdf.Ln(2)
}

// paragraph writes wrapped text across the content width.
func (w *writer) paragraph(text string, style string, size float64, c [3]int) {
	if text == "" {
		return
	}
	w.font(style, size)
	w.color(c)
	lines := w.split(w.tr(text), contentW)
	w.ensure(float64(len(lines)) * lineH)
	for _, l := range lines {
		w.ensure(lineH)
		w.pdf.CellFormat(contentW, lineH, l, "", 1, "L", false, 0, "")
	}
}

// column is one cell of a table row.
type column struct {
	width float64
	text  string
	align string
	style string
	color [3]int
}

// row writes cells side by side, each wrapped in its width; the row is as
// tall as its tallest cell and moves to a new page as a whole.
func (w *writer) row(size float64, cols ...column) {
	split := make([][]string, len(cols))
	height := lineH
	for i, c := range cols {
		w.font(c.style, size)
		split[i] = w.split(w.tr(c.text), c.width-1)
		if len(split[i]) == 0 {
			split[i] = []string{""}
		}
		if h := float64(len(split[i])) * lineH; h > height {
			height = h
		}
	}
	w.ensure(height + 1)
	x, y := margin, w.pdf.GetY()
	for i, c := range cols {
		w.font(c.style, size)
		w.color(c.color)
		for j, l := range split[i] {
			w.pdf.SetXY(x, y+float64(j)*lineH)
			w.pdf.CellFormat(c.width, lineH, l, "", 0, c.align, false, 0, "")
		}
		x += c.width
	}
	w.pdf.SetXY(margin, y+height+1)
}

func (w *writer) amountRows(items []application.DocAmount) {
	for _, a := range items {
		w.row(10, column{width: contentW - 45, text: a.Label, align: "L", color: ink}, column{width: 45, text: a.Amount, align: "R", color: ink})
		if a.Detail != "" {
			w.row(8.5, column{width: contentW - 45, text: a.Detail, align: "L", color: muted})
		}
	}
}

// ---- page furniture -----------------------------------------------------------

func (w *writer) watermark() {
	if !w.doc.Draft {
		return
	}
	w.pdf.TransformBegin()
	w.pdf.SetAlpha(0.12, "Normal")
	w.pdf.TransformRotate(35, pageW/2, pageH/2)
	w.font("B", 110)
	w.pdf.SetTextColor(mark[0], mark[1], mark[2])
	label := "DRAF"
	tw := w.pdf.GetStringWidth(label)
	w.pdf.Text(pageW/2-tw/2, pageH/2+15, label)
	w.pdf.SetAlpha(1, "Normal")
	w.pdf.TransformEnd()
}

func (w *writer) footer() {
	w.pdf.SetY(pageH - 14)
	w.hairline()
	w.pdf.Ln(1.5)
	w.font("", 8)
	w.color(muted)
	third := contentW / 3
	w.pdf.CellFormat(third, 4, w.tr(w.doc.FooterRef), "", 0, "L", false, 0, "")
	w.pdf.CellFormat(third, 4, w.tr(fmt.Sprintf("Halaman %d dari {nb}", w.pdf.PageNo())), "", 0, "C", false, 0, "")
	w.font("", 7)
	w.pdf.CellFormat(third, 4, w.tr("Dibuat dengan ElProof"), "", 0, "R", false, 0, "")
}

// ---- page 1: the whole agreement ------------------------------------------------

func (w *writer) letterhead() {
	lh := w.doc.Letterhead
	top := w.pdf.GetY()
	textX := margin
	if len(lh.Logo) > 0 && lh.LogoType != "" {
		info := w.pdf.RegisterImageOptionsReader("logo", fpdf.ImageOptions{ImageType: lh.LogoType}, bytes.NewReader(lh.Logo))
		if info != nil && w.pdf.Ok() {
			h := 16.0
			iw := h * info.Width() / info.Height()
			if iw > 40 {
				iw, h = 40, 40*info.Height()/info.Width()
			}
			w.pdf.ImageOptions("logo", margin, top, iw, h, false, fpdf.ImageOptions{ImageType: lh.LogoType}, 0, "")
			textX = margin + iw + 4
		}
	}
	// The right block holds the longest line of the page head ("AMR-PNW-261120-001 · Revisi 12 · 30 September 2026"),
	// sized so it stays on one line in Plus Jakarta Sans.
	const rightW = 86.0
	leftW := pageW - margin - textX - rightW
	w.pdf.SetXY(textX, top)
	w.font("B", 13)
	w.color(ink)
	w.pdf.MultiCell(leftW, 6, w.tr(lh.Name), "", "L", false)
	w.font("", 8.5)
	w.color(muted)
	for _, line := range []string{lh.Contact, lh.Address} {
		if line != "" {
			w.pdf.SetX(textX)
			w.pdf.MultiCell(leftW, 4.2, w.tr(line), "", "L", false)
		}
	}
	leftBottom := w.pdf.GetY()

	rightX := pageW - margin - rightW
	w.pdf.SetXY(rightX, top)
	w.font("B", 18)
	w.color(ink)
	w.pdf.CellFormat(rightW, 8, w.tr(w.doc.Title), "", 2, "R", false, 0, "")
	w.font("", 8.5)
	w.color(muted)
	w.pdf.MultiCell(rightW, 4.2, w.tr(w.doc.NumberLine), "", "R", false)
	if w.doc.ValidityLine != "" {
		w.pdf.SetX(rightX)
		w.pdf.MultiCell(rightW, 4.2, w.tr(w.doc.ValidityLine), "", "R", false)
	}
	bottom := w.pdf.GetY()
	if leftBottom > bottom {
		bottom = leftBottom
	}
	if len(lh.Logo) > 0 && top+17 > bottom {
		bottom = top + 17
	}
	w.pdf.SetXY(margin, bottom+3)
	w.hairline()
	w.pdf.Ln(4)
	w.paragraph(w.doc.ForLine, "B", 11, ink)
}

func (w *writer) events() {
	if len(w.doc.Events) == 0 {
		return
	}
	w.heading("Acara")
	widths := []float64{30, 46, 50, 20, contentW - 146}
	head := []string{"Acara", "Tanggal", "Venue", "Tata letak", "Tamu"}
	cols := make([]column, len(head))
	for i, h := range head {
		cols[i] = column{width: widths[i], text: h, align: "L", style: "B", color: muted}
	}
	w.row(8.5, cols...)
	for _, e := range w.doc.Events {
		w.row(9.5,
			column{width: widths[0], text: e.Kind, align: "L", style: "B", color: ink},
			column{width: widths[1], text: e.Date, align: "L", color: ink},
			column{width: widths[2], text: e.Venue, align: "L", color: ink},
			column{width: widths[3], text: e.Layout, align: "L", color: ink},
			column{width: widths[4], text: e.Guests, align: "L", color: ink})
	}
}

func (w *writer) summary() {
	w.heading("Ringkasan biaya")
	w.amountRows(w.doc.Summary)
	w.ensure(12)
	w.pdf.Ln(1)
	w.pdf.SetFillColor(band[0], band[1], band[2])
	w.font("B", 12)
	w.color(ink)
	w.pdf.CellFormat(contentW-55, 9, w.tr(w.doc.ContractTot.Label), "", 0, "L", true, 0, "")
	w.pdf.CellFormat(55, 9, w.tr(w.doc.ContractTot.Amount), "", 1, "R", true, 0, "")
}

func (w *writer) terms() {
	if len(w.doc.Terms) == 0 {
		return
	}
	w.heading("Termin pembayaran")
	for _, t := range w.doc.Terms {
		w.row(10,
			column{width: 45, text: t.Label, align: "L", style: "B", color: ink},
			column{width: contentW - 90, text: t.Due, align: "L", color: muted},
			column{width: 45, text: t.Amount, align: "R", color: ink})
	}
	if w.doc.TermsNote != "" {
		w.paragraph(w.doc.TermsNote, "", 8.5, muted)
	}
	if len(w.doc.BankLines) > 0 {
		w.pdf.Ln(2)
		w.paragraph("Pembayaran ke", "B", 9, ink)
		for _, l := range w.doc.BankLines {
			w.paragraph(l, "", 9, ink)
		}
	}
}

// ---- page 2: details, client-paid vendors, conditions, signatures ---------------

func (w *writer) details() {
	w.heading("Rincian")
	for _, p := range w.doc.Packages {
		w.ensure(16)
		w.row(10.5, column{width: contentW - 45, text: p.Title, align: "L", style: "B", color: ink},
			column{width: 45, text: p.Amount, align: "R", style: "B", color: ink})
		if p.Covers != "" {
			w.row(8.5, column{width: contentW, text: p.Covers, align: "L", color: muted})
		}
		for _, g := range p.Groups {
			if g.Heading != "" {
				w.row(9, column{width: contentW, text: g.Heading, align: "L", style: "B", color: muted})
			}
			for _, item := range g.Items {
				w.row(9.5, column{width: 5, text: "·", align: "L", color: ink}, column{width: contentW - 5, text: item, align: "L", color: ink})
			}
		}
		if p.Details != "" {
			w.paragraph(p.Details, "", 9, muted)
		}
		if p.Extra != nil {
			w.amountRows([]application.DocAmount{*p.Extra})
		}
		w.pdf.Ln(2)
	}
	if len(w.doc.Items) > 0 {
		w.row(10, column{width: contentW, text: "Item tambahan", align: "L", style: "B", color: ink})
		w.amountRows(w.doc.Items)
		w.pdf.Ln(1)
	}
	if len(w.doc.Bonuses) > 0 {
		w.row(10, column{width: contentW, text: "Bonus", align: "L", style: "B", color: ink})
		for _, b := range w.doc.Bonuses {
			w.row(9.5, column{width: contentW - 45, text: b, align: "L", color: ink}, column{width: 45, text: "Gratis", align: "R", color: muted})
		}
		w.pdf.Ln(1)
	}
	if len(w.doc.Discounts) > 0 {
		w.row(10, column{width: contentW, text: "Potongan", align: "L", style: "B", color: ink})
		w.amountRows(w.doc.Discounts)
		w.pdf.Ln(1)
	}
	w.ensure(float64(len(w.doc.Totals))*6 + 4)
	w.hairline()
	w.pdf.Ln(1.5)
	for i, t := range w.doc.Totals {
		style := ""
		if i == len(w.doc.Totals)-1 {
			style = "B"
		}
		w.row(10, column{width: contentW - 45, text: t.Label, align: "R", style: style, color: ink},
			column{width: 45, text: t.Amount, align: "R", style: style, color: ink})
	}
}

func (w *writer) clientPaid() {
	if len(w.doc.ClientPaid) == 0 {
		return
	}
	w.heading("Vendor yang dibayar langsung oleh klien")
	for _, c := range w.doc.ClientPaid {
		w.row(9.5, column{width: contentW - 60, text: c.Label, align: "L", color: ink}, column{width: 60, text: c.Detail, align: "R", color: muted})
	}
	w.paragraph(w.doc.ClientNote, "", 8.5, muted)
}

func (w *writer) conditions() {
	if len(w.doc.Conditions) == 0 {
		return
	}
	w.heading("Syarat & ketentuan")
	for _, c := range w.doc.Conditions {
		w.paragraph(c, "", 9, ink)
	}
}

func (w *writer) signatures() {
	w.pdf.Ln(4)
	w.paragraph(w.doc.Closing, "", 9.5, ink)
	w.ensure(42)
	w.pdf.Ln(6)
	half := contentW / 2
	y := w.pdf.GetY()
	for i, s := range []application.DocSignature{w.doc.SignLeft, w.doc.SignRight} {
		x := margin + float64(i)*half
		w.pdf.SetXY(x, y)
		w.font("", 9.5)
		w.color(muted)
		w.pdf.CellFormat(half-8, 5, w.tr(s.Heading), "", 2, "L", false, 0, "")
		w.pdf.SetDrawColor(rule[0], rule[1], rule[2])
		w.pdf.Line(x, y+24, x+half-12, y+24)
		w.pdf.SetXY(x, y+25)
		w.font("B", 10)
		w.color(ink)
		w.pdf.MultiCell(half-8, 5, w.tr(s.Name), "", "L", false)
		w.pdf.SetX(x)
		w.font("", 8.5)
		w.color(muted)
		w.pdf.MultiCell(half-8, 4.5, w.tr(s.Sub), "", "L", false)
	}
}

// Render lays out the document: page 1 sums up the whole agreement, the
// following pages hold the details, the vendors the client pays directly, the
// terms & conditions, and the signatures.
func (r *Renderer) Render(doc application.QuotationDocument) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetCompression(r.compress)
	pdf.SetCatalogSort(true)
	created := doc.CreatedAt
	if created.IsZero() {
		created = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	pdf.SetCreationDate(created.UTC())
	pdf.SetModificationDate(created.UTC())
	pdf.SetProducer("ElProof", false)
	pdf.SetCreator("ElProof", false)
	pdf.SetMargins(margin, margin, margin)
	pdf.SetAutoPageBreak(true, bottomMargin)
	pdf.AliasNbPages("{nb}")
	w := &writer{pdf: pdf, fonts: setupFonts(pdf, r.coreFont), doc: doc}
	pdf.SetTitle(w.tr(doc.Title+" "+doc.FooterRef), w.fonts.utf8)
	pdf.SetHeaderFunc(w.watermark)
	pdf.SetFooterFunc(w.footer)

	pdf.AddPage()
	w.letterhead()
	w.events()
	w.summary()
	w.terms()

	pdf.AddPage()
	w.details()
	w.clientPaid()
	w.conditions()
	w.signatures()

	if err := pdf.Error(); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
