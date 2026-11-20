package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	offerscontracts "elproof/internal/modules/offers/contracts"
	"elproof/internal/modules/projects/application"
	"elproof/internal/shared/response"
)

// "Jadikan proyek" (PLAN §5.4, DESAIN §6.5): the routes under
// /api/v1/projects/from-quotation/{quotationId}, and "Kesepakatan awal"
// (GET /api/v1/projects/{id}/agreement, DESAIN §6.10).

// conversionEventBody is the exact date of one event of the accepted
// revision, by position.
type conversionEventBody struct {
	EventDate string `json:"eventDate"`
}

type downPaymentBody struct {
	Amount          int64       `json:"amount"`
	PaymentDate     string      `json:"paymentDate"`
	Method          string      `json:"method"`
	ReferenceNumber string      `json:"referenceNumber"`
	Notes           string      `json:"notes"`
	Evidence        []proofBody `json:"evidence"`
}

// conversionBody is POST /projects/from-quotation/{id} (and .../preview).
type conversionBody struct {
	Name          string                `json:"name"`
	BrideName     string                `json:"brideName"`
	GroomName     string                `json:"groomName"`
	Events        []conversionEventBody `json:"events"`
	PrepStartDate string                `json:"prepStartDate"`
	PICStaffID    int64                 `json:"picStaffId"`
	// PICSalesStaffID null or absent = the quotation's PIC.
	PICSalesStaffID *int64           `json:"picSalesStaffId"`
	Description     string           `json:"description"`
	DownPayment     *downPaymentBody `json:"downPayment"`
}

func (b conversionBody) input() application.ConversionInput {
	bad := map[string][]string{}
	date := func(raw, key string) time.Time {
		if raw == "" {
			return time.Time{}
		}
		d, err := parseDate(raw)
		if err != nil {
			bad[key] = []string{"Gunakan format YYYY-MM-DD"}
			return time.Time{}
		}
		return d
	}
	in := application.ConversionInput{
		Name: b.Name, BrideName: b.BrideName, GroomName: b.GroomName,
		PrepStartDate: date(b.PrepStartDate, "prepStartDate"),
		PICStaffID:    b.PICStaffID, PICSalesStaffID: b.PICSalesStaffID, Description: b.Description,
	}
	for i, e := range b.Events {
		in.EventDates = append(in.EventDates, date(e.EventDate, "events."+strconv.Itoa(i)+".eventDate"))
	}
	if d := b.DownPayment; d != nil {
		in.DownPayment = &application.DownPaymentInput{
			Amount: d.Amount, PaymentDate: date(d.PaymentDate, "downPayment.paymentDate"), Method: d.Method,
			ReferenceNumber: d.ReferenceNumber, Notes: d.Notes, Evidence: toProofItems(d.Evidence),
		}
	}
	in.FieldErrors = bad
	return in
}

// conversionResponse is the project made from the quotation, with its events
// (as POST /projects answers), plus whether this call made it and the down
// payment recorded with it.
type conversionResponse struct {
	projectCreatedResponse
	Created       bool   `json:"created"`
	DownPaymentID *int64 `json:"downPaymentId"`
}

// routeConversion serves /api/v1/projects/from-quotation/{quotationId}[/...].
// Staff only, project.create; the down payment and its uploads also need
// finance.record.
func (h *Handler) routeConversion(w http.ResponseWriter, r *http.Request, rest []string) {
	claims, ok := requireStaff(w, r)
	if !ok {
		return
	}
	if len(rest) == 0 {
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
		return
	}
	quotationID, err := strconv.ParseInt(rest[0], 10, 64)
	if err != nil || quotationID <= 0 {
		response.Error(w, http.StatusBadRequest, "ID penawaran tidak valid", nil)
		return
	}
	if !claims.can("project.create") {
		forbiddenPermission(w, "Peran Anda tidak dapat membuat proyek baru")
		return
	}
	if h.conversion == nil {
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
		return
	}
	rest = rest[1:]
	switch {
	case len(rest) == 0 && r.Method == http.MethodPost:
		h.idempotent(w, r, claims, func(w http.ResponseWriter, r *http.Request) {
			h.convertQuotation(w, r, claims, quotationID)
		})
	case len(rest) == 1 && rest[0] == "preview" && r.Method == http.MethodPost:
		h.previewConversion(w, r, claims, quotationID)
	case len(rest) == 1 && rest[0] == "uploads" && r.Method == http.MethodPost:
		h.createConversionUpload(w, r, claims, quotationID)
	case len(rest) == 2 && rest[0] == "uploads" && r.Method == http.MethodPut:
		h.putConversionUpload(w, r, claims, quotationID, rest[1])
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

func decodeConversion(w http.ResponseWriter, r *http.Request, claims staffClaims) (application.ConversionInput, bool) {
	var body conversionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return application.ConversionInput{}, false
	}
	if body.DownPayment != nil && !claims.can("finance.record") {
		forbiddenPermission(w, "Peran Anda tidak dapat mencatat pembayaran klien")
		return application.ConversionInput{}, false
	}
	return body.input(), true
}

// convertQuotation is POST /projects/from-quotation/{id}: 201 with the new
// project, or 200 with the project the quotation already became.
func (h *Handler) convertQuotation(w http.ResponseWriter, r *http.Request, claims staffClaims, quotationID int64) {
	input, ok := decodeConversion(w, r, claims)
	if !ok {
		return
	}
	res, err := h.conversion.Convert(r.Context(), claims.tenantID, claims.actor(), quotationID, input, nowWIB())
	if err != nil {
		writeAppError(w, err)
		return
	}
	p := res.Project
	resp := conversionResponse{projectCreatedResponse: projectCreatedResponse{projectResponse: toProjectResponse(*p), Events: []eventResponse{}}, Created: res.Created}
	resp.PICName = h.projects.ResolvePICName(r.Context(), claims.tenantID, p.PICStaffID)
	if res.DownPayment != nil && res.DownPayment.Payment != nil {
		id := res.DownPayment.Payment.ID
		resp.DownPaymentID = &id
	}
	events, err := h.events.List(r.Context(), claims.tenantID, p.ID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if resp.Events, err = h.eventRows(r.Context(), claims, p.ID, events); err != nil {
		writeAppError(w, err)
		return
	}
	h.describeProjects(r.Context(), claims.tenantID, []*projectResponse{&resp.projectResponse})
	if !res.Created {
		response.OK(w, "Penawaran ini sudah dijadikan proyek", resp)
		return
	}
	response.Created(w, "Proyek dibuat dari penawaran", resp)
}

// conversionPreviewResponse is "Periksa sebelum membuat".
type conversionPreviewResponse struct {
	QuotationID     int64  `json:"quotationId"`
	QuotationNumber string `json:"quotationNumber"`
	RevisionN       int    `json:"revisionN"`
	// EventDate is the main event's (the earliest exact date).
	EventDate     string                   `json:"eventDate"`
	ContractTotal int64                    `json:"contractTotal"`
	IsPKP         bool                     `json:"isPkp"`
	Terms         []conversionTermResponse `json:"terms"`
	Counts        conversionCounts         `json:"counts"`
	// ExistingProjectID: the quotation is a project already (POST answers it).
	ExistingProjectID *int64 `json:"existingProjectId"`
}

type conversionTermResponse struct {
	Label   string `json:"label"`
	Amount  int64  `json:"amount"`
	DueDate string `json:"dueDate"`
}

type conversionCounts struct {
	Events       int  `json:"events"`
	Suppliers    int  `json:"suppliers"`
	Lines        int  `json:"lines"`
	HasTermsText bool `json:"hasTermsText"`
}

// previewConversion is POST /projects/from-quotation/{id}/preview: the same
// body and the same checks (422 with the same keys, capacity included) as the
// conversion, writing nothing.
func (h *Handler) previewConversion(w http.ResponseWriter, r *http.Request, claims staffClaims, quotationID int64) {
	input, ok := decodeConversion(w, r, claims)
	if !ok {
		return
	}
	pv, err := h.conversion.Preview(r.Context(), claims.tenantID, claims.actor(), quotationID, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	resp := conversionPreviewResponse{
		QuotationID: pv.Quotation.ID, QuotationNumber: pv.Quotation.Number, RevisionN: pv.Quotation.Revision.N,
		EventDate: pv.EventDate.Format(dateLayout), ContractTotal: pv.ContractTotal, IsPKP: pv.Quotation.IsPKP,
		Terms:             make([]conversionTermResponse, 0, len(pv.Terms)),
		Counts:            conversionCounts{Events: pv.Events, Suppliers: pv.Suppliers, Lines: pv.Lines, HasTermsText: pv.HasTermsText},
		ExistingProjectID: pv.ExistingProjectID,
	}
	for _, t := range pv.Terms {
		resp.Terms = append(resp.Terms, conversionTermResponse{Label: t.Label, Amount: t.Amount, DueDate: t.DueDate.Format(dateLayout)})
	}
	response.OK(w, "ok", resp)
}

// createConversionUpload is POST /projects/from-quotation/{id}/uploads: a
// signed upload for the down payment's proof, before the project exists (the
// token is bound to the workspace, not to a project). Same body and answer as
// POST /projects/{id}/uploads.
func (h *Handler) createConversionUpload(w http.ResponseWriter, r *http.Request, claims staffClaims, quotationID int64) {
	if !claims.can("finance.record") {
		forbiddenPermission(w, "Peran Anda tidak dapat mencatat pembayaran klien")
		return
	}
	if err := h.conversion.CheckQuotation(r.Context(), claims.tenantID, claims.actor(), quotationID); err != nil {
		writeAppError(w, err)
		return
	}
	var body uploadRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	grant, err := h.uploads.Grant(r.Context(), claims.tenantID, 0, application.UploadInput{
		FileName: body.FileName, MimeType: body.MimeType, Size: body.Size,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	if !grant.Direct {
		grant.URL = "/api/v1/projects/from-quotation/" + strconv.FormatInt(quotationID, 10) + "/uploads/" + grant.Token
	}
	response.Created(w, "Unggahan disiapkan", uploadGrantResponse{
		UploadToken: grant.Token, Method: http.MethodPut, URL: grant.URL,
		Headers:   map[string]string{"Content-Type": grant.MimeType},
		ExpiresAt: grant.ExpiresAt.In(wib).Format(time.RFC3339), Direct: grant.Direct,
	})
}

// putConversionUpload is PUT /projects/from-quotation/{id}/uploads/{token}:
// the bytes, when object storage is not reachable from the browser.
func (h *Handler) putConversionUpload(w http.ResponseWriter, r *http.Request, claims staffClaims, quotationID int64, token string) {
	if !claims.can("finance.record") {
		forbiddenPermission(w, "Peran Anda tidak dapat mencatat pembayaran klien")
		return
	}
	if err := h.conversion.CheckQuotation(r.Context(), claims.tenantID, claims.actor(), quotationID); err != nil {
		writeAppError(w, err)
		return
	}
	h.putUpload(w, r, claims, 0, token)
}

// agreementResponse is "Kesepakatan awal": the frozen revision the project
// was made from (read-only), for comparison with the current Rincian.
type agreementResponse struct {
	QuotationID       int64   `json:"quotationId"`
	QuotationNumber   string  `json:"quotationNumber"`
	RevisionID        int64   `json:"revisionId"`
	RevisionN         int     `json:"revisionN"`
	RevisionCreatedAt string  `json:"revisionCreatedAt"`
	AcceptedOn        *string `json:"acceptedOn"`
	AcceptedChannel   string  `json:"acceptedChannel"`
	ContractTotal     int64   `json:"contractTotal"`
	// Content is the revision's content as `offers` froze it (events with
	// sessions, suppliers, lines with positions, terms with their computed
	// amounts, terms text, totals). agreedCost is null without
	// finance.margin.view.
	Content offerscontracts.QuotationContent `json:"content"`
}

// getAgreement is GET /projects/{id}/agreement (staff with access to the
// project; never a client). 404 for a project made without a quotation.
func (h *Handler) getAgreement(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	if claims.isClient || h.conversion == nil {
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
		return
	}
	q, err := h.conversion.Agreement(r.Context(), claims.tenantID, projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	content := q.Content
	content.Suppliers = append([]offerscontracts.SnapshotSupplier(nil), q.Content.Suppliers...)
	if !claims.can("finance.margin.view") {
		for i := range content.Suppliers {
			content.Suppliers[i].AgreedCost = nil
		}
	}
	resp := agreementResponse{
		QuotationID: q.ID, QuotationNumber: q.Number, RevisionID: q.Revision.ID, RevisionN: q.Revision.N,
		RevisionCreatedAt: q.Revision.CreatedAt.UTC().Format(time.RFC3339), AcceptedChannel: q.AcceptedChannel,
		ContractTotal: q.Revision.ContractTotal, Content: content,
	}
	if q.AcceptedOn != nil {
		d := q.AcceptedOn.Format(dateLayout)
		resp.AcceptedOn = &d
	}
	response.OK(w, "ok", resp)
}
