package application

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/logger"
)

// TenantBrandingResolver is the narrow, primitive-typed shape DocumentService
// needs from `platform` to embed a tenant's real branding (business name,
// brand color, logo bytes) on a generated Kwitansi/Tagihan PDF (PLAN.md) --
// deliberately primitive-only (not platformcontracts.BrandingInfo), same
// import-cycle-avoidance reasoning as StaffNameResolver above:
// platform/application already imports projects/contracts (Timeline Default
// template seeding, tenant registration), so projects/application importing
// platform/contracts back would close the loop
// (projects/application -> platform/contracts -> platform/application ->
// projects/contracts -> projects/application). main.go bridges the real
// platformcontracts.Contracts into this shape with a small adapter, the same
// pattern as staffNameResolver. Unlike ClientCleaner/VenueResolver, `platform`
// is already built before `projects` in main.go, so this is a direct
// constructor argument, no two-phase setter needed (same as StaffNameResolver).
type TenantBrandingResolver interface {
	ResolveBranding(ctx context.Context, tenantID int64) (businessName, brandColorPreset string, logo []byte, err error)
}

type GenerateInvoiceInput struct {
	Amount  int64
	DueDate *time.Time
	Note    string
}

// DocumentService generates Kwitansi (receipt) and Tagihan (bill) PDFs
// on-demand from live client_payments/project data -- there is no stored
// document entity anywhere in this feature (PLAN.md's "agile" approach). Kept
// as its own service (not folded into ClientPaymentService, already dense)
// since it's a fundamentally different concern: rendering, not ledger
// mutation.
type DocumentService struct {
	clientPayments ClientPaymentRepository
	projects       ProjectRepository
	activity       *ActivityService
	branding       TenantBrandingResolver
}

func NewDocumentService(clientPayments ClientPaymentRepository, projects ProjectRepository, activity *ActivityService, branding TenantBrandingResolver) *DocumentService {
	return &DocumentService{clientPayments: clientPayments, projects: projects, activity: activity, branding: branding}
}

// resolveBranding is best-effort: a branding lookup failure (or a tenant with
// no branding resolver wired at all, defensively) must never block cutting a
// receipt/bill -- it just renders without a business name/logo, same
// tolerance the checklist requires for "tenant belum punya logo".
func (s *DocumentService) resolveBranding(ctx context.Context, tenantID int64) (businessName string, logo []byte) {
	if s.branding == nil {
		return "", nil
	}
	businessName, _, logo, err := s.branding.ResolveBranding(ctx, tenantID)
	if err != nil {
		logger.Error("failed to resolve tenant branding for document generation: %v", err)
		return "", nil
	}
	return businessName, logo
}

// GenerateReceipt renders a Kwitansi (or, for a Refund row, a "Bukti
// Pengembalian Dana") for exactly one existing client_payments row -- no new
// data is created, this is purely a read+render. Returns the PDF bytes and a
// suggested filename.
func (s *DocumentService) GenerateReceipt(ctx context.Context, tenantID, projectID, paymentID int64) ([]byte, string, error) {
	project, err := s.projects.FindByID(ctx, tenantID, projectID)
	if err != nil {
		return nil, "", err
	}
	if project == nil {
		return nil, "", apperror.NotFound("Project tidak ditemukan")
	}
	payment, err := s.clientPayments.FindByID(ctx, projectID, paymentID)
	if err != nil {
		return nil, "", err
	}
	if payment == nil {
		return nil, "", apperror.NotFound("Pembayaran tidak ditemukan")
	}

	businessName, logo := s.resolveBranding(ctx, tenantID)
	clientName := clientDisplayName(*project)

	isRefund := payment.Type == domain.PaymentRefund
	title := "KWITANSI"
	verb := "Telah diterima dari"
	fileName := fmt.Sprintf("Kwitansi-%d.pdf", paymentID)
	if isRefund {
		title = "BUKTI PENGEMBALIAN DANA"
		verb = "Telah dikembalikan kepada"
		fileName = fmt.Sprintf("Bukti-Pengembalian-Dana-%d.pdf", paymentID)
	}

	pdf, err := buildDocumentPDF(documentContent{
		BusinessName: businessName,
		Logo:         logo,
		Title:        title,
		DocNumber:    fmt.Sprintf("KWT/%d/%d", projectID, paymentID),
		ProjectName:  project.Name,
		ClientName:   clientName,
		Rows: []documentRow{
			{Label: "Jenis Pembayaran", Value: string(payment.Type)},
			{Label: "Tanggal", Value: payment.PaymentDate.Format("2 January 2006")},
			{Label: "Metode", Value: payment.Method},
			{Label: "No. Referensi", Value: payment.ReferenceNumber},
			{Label: "Jumlah", Value: formatRupiah(payment.Amount)},
		},
		BodyText: fmt.Sprintf("%s %s sejumlah %s.", verb, clientName, formatRupiah(payment.Amount)),
	})
	if err != nil {
		return nil, "", err
	}
	return pdf, fileName, nil
}

// GenerateInvoice renders a Tagihan for the project's current outstanding
// balance (or an ad-hoc smaller amount, e.g. just a DP) -- see PLAN.md. When
// DueDate is set, it overwrites projects.next_due_date (there is only ever
// one "active" Tagihan at a time, no history table to keep consistent).
// Unlike GenerateReceipt, this is recorded to activity_log -- it represents a
// formal request to the client, not a re-print of existing data.
func (s *DocumentService) GenerateInvoice(ctx context.Context, tenantID, projectID, actorStaffID int64, input GenerateInvoiceInput) ([]byte, string, error) {
	if input.Amount <= 0 {
		return nil, "", apperror.Validation("Jumlah tagihan tidak valid", map[string][]string{"amount": {"Jumlah harus lebih dari 0"}})
	}
	project, err := s.projects.FindByID(ctx, tenantID, projectID)
	if err != nil {
		return nil, "", err
	}
	if project == nil {
		return nil, "", apperror.NotFound("Project tidak ditemukan")
	}
	payments, err := s.clientPayments.ListByProject(ctx, projectID)
	if err != nil {
		return nil, "", err
	}
	var totalReceived int64
	for _, p := range payments {
		if p.Type == domain.PaymentRefund {
			totalReceived -= p.Amount
		} else {
			totalReceived += p.Amount
		}
	}
	outstanding := project.ContractValue - totalReceived

	if input.DueDate != nil {
		if err := s.projects.SetNextDueDate(ctx, tenantID, projectID, input.DueDate); err != nil {
			return nil, "", err
		}
	}

	businessName, logo := s.resolveBranding(ctx, tenantID)
	clientName := clientDisplayName(*project)

	rows := []documentRow{
		{Label: "Nilai Kontrak", Value: formatRupiah(project.ContractValue)},
		{Label: "Sudah Dibayar", Value: formatRupiah(totalReceived)},
		{Label: "Sisa Tagihan", Value: formatRupiah(outstanding)},
		{Label: "Jumlah Diminta", Value: formatRupiah(input.Amount)},
	}
	if input.DueDate != nil {
		rows = append(rows, documentRow{Label: "Jatuh Tempo", Value: input.DueDate.Format("2 January 2006")})
	}

	bodyText := fmt.Sprintf("Dengan ini kami mengajukan tagihan kepada %s sejumlah %s.", clientName, formatRupiah(input.Amount))
	if input.Note != "" {
		bodyText += " Catatan: " + input.Note
	}

	now := time.Now()
	docNumber := fmt.Sprintf("TGH-%d-%s", projectID, now.Format("20060102"))
	pdf, err := buildDocumentPDF(documentContent{
		BusinessName: businessName, Logo: logo, Title: "TAGIHAN", DocNumber: docNumber,
		ProjectName: project.Name, ClientName: clientName, Rows: rows, BodyText: bodyText,
	})
	if err != nil {
		return nil, "", err
	}

	s.activity.Record(ctx, &projectID, domain.ActivityInvoiceGenerated, actorStaffID, "project", formatID(projectID), "Tagihan",
		fmt.Sprintf("Tagihan %s diterbitkan", formatRupiah(input.Amount)))

	fileName := fmt.Sprintf("Tagihan-%d-%s.pdf", projectID, now.Format("20060102"))
	return pdf, fileName, nil
}

func clientDisplayName(p domain.Project) string {
	names := make([]string, 0, 2)
	if p.BrideName != "" {
		names = append(names, p.BrideName)
	}
	if p.GroomName != "" {
		names = append(names, p.GroomName)
	}
	return strings.Join(names, " & ")
}

// formatRupiah renders a plain "Rp X.XXX.XXX" grouping -- the same grouping
// formatCurrency (frontend) already uses, kept independent here since PDF
// generation is backend-only and has no reason to depend on frontend code.
func formatRupiah(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	digits := fmt.Sprintf("%d", n)
	var grouped []byte
	for i, c := range []byte(digits) {
		if i > 0 && (len(digits)-i)%3 == 0 {
			grouped = append(grouped, '.')
		}
		grouped = append(grouped, c)
	}
	result := "Rp " + string(grouped)
	if neg {
		result = "-" + result
	}
	return result
}

type documentRow struct {
	Label string
	Value string
}

type documentContent struct {
	BusinessName string
	Logo         []byte
	Title        string
	DocNumber    string
	ProjectName  string
	ClientName   string
	Rows         []documentRow
	BodyText     string
}

// detectImageExtension sniffs magic bytes rather than trusting a stored
// mimeType (TenantBrandingResolver only carries raw bytes) -- maroto/gofpdf
// only embeds PNG/JPEG, so an unrecognized format (e.g. a WebP logo, allowed
// at upload time) is treated the same as "no logo" rather than erroring the
// whole document.
func detectImageExtension(data []byte) (extension.Type, bool) {
	if len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}) {
		return extension.Png, true
	}
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return extension.Jpg, true
	}
	return "", false
}

func buildDocumentPDF(content documentContent) ([]byte, error) {
	cfg := config.NewBuilder().
		WithLeftMargin(15).
		WithTopMargin(15).
		WithRightMargin(15).
		WithBottomMargin(15).
		Build()
	m := maroto.New(cfg)

	if ext, ok := detectImageExtension(content.Logo); ok {
		m.AddRow(16,
			image.NewFromBytesCol(2, content.Logo, ext, props.Rect{Center: true, Percent: 80}),
			col.New(10).Add(text.New(content.BusinessName, props.Text{Size: 13, Style: fontstyle.Bold, Top: 5})),
		)
	} else {
		m.AddRow(16, col.New(12).Add(text.New(content.BusinessName, props.Text{Size: 13, Style: fontstyle.Bold, Align: align.Center, Top: 5})))
	}
	m.AddRows(line.NewRow(2))

	m.AddRows(text.NewRow(10, content.Title, props.Text{Style: fontstyle.Bold, Size: 15, Align: align.Center, Top: 4}))
	m.AddRows(text.NewRow(6, "No. Dokumen: "+content.DocNumber, props.Text{Size: 9, Align: align.Center}))

	m.AddRow(8,
		text.NewCol(3, "Project", props.Text{Size: 9, Style: fontstyle.Bold}),
		text.NewCol(9, content.ProjectName, props.Text{Size: 9}),
	)
	m.AddRow(8,
		text.NewCol(3, "Client", props.Text{Size: 9, Style: fontstyle.Bold}),
		text.NewCol(9, content.ClientName, props.Text{Size: 9}),
	)
	m.AddRow(8,
		text.NewCol(3, "Tanggal Cetak", props.Text{Size: 9, Style: fontstyle.Bold}),
		text.NewCol(9, time.Now().Format("2 January 2006"), props.Text{Size: 9}),
	)
	m.AddRows(line.NewRow(4))

	for _, row := range content.Rows {
		m.AddRow(8,
			text.NewCol(4, row.Label, props.Text{Size: 9, Style: fontstyle.Bold}),
			text.NewCol(8, row.Value, props.Text{Size: 9}),
		)
	}
	m.AddRows(line.NewRow(4))

	m.AddRows(text.NewRow(16, content.BodyText, props.Text{Size: 10, Top: 4}))

	m.AddRows(text.NewRow(10, "Dokumen ini dibuat otomatis oleh sistem ElProof pada "+time.Now().Format("2 January 2006 15:04")+" WIB.",
		props.Text{Size: 7, Align: align.Center, Color: &props.Color{Red: 130, Green: 130, Blue: 130}}))

	document, err := m.Generate()
	if err != nil {
		return nil, apperror.Internal("Gagal membuat dokumen PDF")
	}
	return document.GetBytes(), nil
}
