package presentation

import (
	"context"
	"encoding/json"
	"net/http"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/clock"
	"elproof/internal/shared/etag"
	"elproof/internal/shared/response"
)

// The Rincian of a project (PLAN §3.1, §3.2): GET/PUT /projects/{id}/lines,
// POST .../lines/preview (writes nothing), POST .../lines (one line),
// PATCH|DELETE .../lines/{lineId}, POST .../lines/{lineId}/restore and
// POST .../lines/reorder. Every write but restore carries the lines' own
// version (linesVersion) as If-Match (ADR-0036): 428 without it, 409 with
// data.current (the Rincian as it is now) when it is stale. A client principal
// may only GET, behind portal.vendor.view, with amounts behind
// portal.finance.view (PortalPolicy.Line).

// SetLines gives the handler the Rincian service.
func (h *Handler) SetLines(s *application.LineService) { h.lines = s }

func (h *Handler) routeLines(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, rest []string) {
	switch {
	case len(rest) == 0 && r.Method == http.MethodGet:
		h.listLines(w, r, claims, projectID)
	case len(rest) == 0 && r.Method == http.MethodPut:
		h.replaceLines(w, r, claims, projectID)
	case len(rest) == 0 && r.Method == http.MethodPost:
		h.addLine(w, r, claims, projectID)
	case len(rest) == 1 && rest[0] == "preview" && r.Method == http.MethodPost:
		h.previewLines(w, r, claims, projectID)
	case len(rest) == 1 && rest[0] == "reorder" && r.Method == http.MethodPost:
		h.reorderLines(w, r, claims, projectID)
	case len(rest) == 1 && r.Method == http.MethodPatch:
		h.patchLine(w, r, claims, projectID, rest[0])
	case len(rest) == 1 && r.Method == http.MethodDelete:
		h.deleteLine(w, r, claims, projectID, rest[0])
	case len(rest) == 2 && rest[1] == "restore" && r.Method == http.MethodPost:
		h.restoreLine(w, r, claims, projectID, rest[0])
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

// --- responses ---

type linePackageRuleResponse struct {
	PackageID    int64   `json:"packageId"`
	Name         string  `json:"name"`
	Price        *int64  `json:"price,omitempty"`
	EventKindIDs []int64 `json:"eventKindIds"`
	// The extra-guest rule (all three null: the package does not charge extra
	// guests). extraGuestStepPrice is left out where amounts are hidden.
	IncludedGuests      *int   `json:"includedGuests"`
	ExtraGuestStep      *int   `json:"extraGuestStep"`
	ExtraGuestStepPrice *int64 `json:"extraGuestStepPrice,omitempty"`
}

type lineExtraGuestsResponse struct {
	EventID        *int64 `json:"eventId"`
	EventKindName  string `json:"eventKindName"`
	Guests         int    `json:"guests"`
	IncludedGuests int    `json:"includedGuests"`
	Over           int    `json:"over"`
	Step           int    `json:"step"`
	StepPrice      *int64 `json:"stepPrice,omitempty"`
	Steps          int    `json:"steps"`
	NoGuests       bool   `json:"noGuests"`
}

// lineResponse is one line. id is 0 for a line a preview has not stored yet
// (key echoes the draft's); parentKey is the parent's key when the parent has
// none stored. amount and unitPrice are left out where the reader may not see
// them (the portal without portal.finance.view, and any component there).
type lineResponse struct {
	ID          int64                    `json:"id"`
	Key         string                   `json:"key,omitempty"`
	Kind        string                   `json:"kind"`
	ParentID    *int64                   `json:"parentId"`
	ParentKey   string                   `json:"parentKey,omitempty"`
	PackageRule *linePackageRuleResponse `json:"packageRule,omitempty"`
	Description string                   `json:"description"`
	CategoryID  *int64                   `json:"categoryId"`
	EventID     *int64                   `json:"eventId"`
	Quantity    *int64                   `json:"quantity"`
	Unit        string                   `json:"unit"`
	UnitPrice   *int64                   `json:"unitPrice,omitempty"`
	Amount      *int64                   `json:"amount,omitempty"`
	IsBonus     bool                     `json:"isBonus"`
	SupplierRef *int64                   `json:"supplierRef,omitempty"`
	SortOrder   int                      `json:"sortOrder"`
	ExtraGuests *lineExtraGuestsResponse `json:"extraGuests,omitempty"`
	UpdatedAt   string                   `json:"updatedAt,omitempty"`
}

type lineTotalsResponse struct {
	Gross    int64 `json:"gross"`
	Discount int64 `json:"discount"`
	Subtotal int64 `json:"subtotal"`
	Tax      int64 `json:"tax"`
	Total    int64 `json:"total"`
}

type lineMarginResponse struct {
	Subtotal     int64 `json:"subtotal"`
	SupplierCost int64 `json:"supplierCost"`
	// MissingCost counts the WO-paid vendors and venues without an agreed cost:
	// the margin is incomplete ("Margin belum lengkap (n vendor)") while > 0.
	MissingCost int   `json:"missingCost"`
	Incomplete  bool  `json:"incomplete"`
	Margin      int64 `json:"margin"`
}

// linesResponse is the Rincian. For staff: every field, margin only with
// finance.margin.view. For the portal: lines (redacted) and isPkp, totals only
// with portal.finance.view, never the version, the schedule check or the margin.
type linesResponse struct {
	ProjectID     int64               `json:"projectId"`
	Lines         []lineResponse      `json:"lines"`
	Totals        *lineTotalsResponse `json:"totals,omitempty"`
	IsPKP         bool                `json:"isPkp"`
	LinesVersion  string              `json:"linesVersion,omitempty"`
	TermsTotal    *int64              `json:"termsTotal,omitempty"`
	HasTerms      *bool               `json:"hasTerms,omitempty"`
	TermsMismatch *bool               `json:"termsMismatch,omitempty"`
	Margin        *lineMarginResponse `json:"margin,omitempty"`
}

// ETag is the lines' version: the If-Match of the next line write.
func (l linesResponse) ETag() string { return quotedVersion(l.LinesVersion) }

func toLineResponse(v application.LineView, all []application.LineView, showAmount bool) lineResponse {
	l := v.ProjectLine
	out := lineResponse{
		ID: l.ID, Key: v.Key, Kind: string(l.Kind), ParentID: l.ParentLineID, Description: l.Description,
		CategoryID: l.CategoryID, EventID: l.EventID, Quantity: l.Quantity, Unit: l.Unit, IsBonus: l.IsBonus,
		SupplierRef: l.SupplierRef, SortOrder: l.SortOrder,
	}
	if !l.UpdatedAt.IsZero() {
		out.UpdatedAt = etag.Format(l.UpdatedAt)
	}
	if p := v.ParentIndex(); p >= 0 && p < len(all) {
		parent := all[p]
		if parent.ID != 0 {
			id := parent.ID
			out.ParentID = &id
		} else {
			out.ParentKey = parent.Key
		}
	}
	if showAmount {
		amount := l.Amount
		out.Amount = &amount
		out.UnitPrice = l.UnitPrice
	}
	if r := l.PackageRule; r != nil {
		rule := &linePackageRuleResponse{
			PackageID: r.PackageID, Name: r.Name, EventKindIDs: append([]int64{}, r.EventKindIDs...),
			IncludedGuests: r.IncludedGuests, ExtraGuestStep: r.ExtraGuestStep,
		}
		if showAmount {
			price := r.Price
			rule.Price, rule.ExtraGuestStepPrice = &price, r.ExtraGuestStepPrice
		}
		out.PackageRule = rule
	}
	if e := v.ExtraGuests; e != nil {
		ex := &lineExtraGuestsResponse{
			EventID: e.EventID, EventKindName: e.EventKindName, Guests: e.Guests, IncludedGuests: e.IncludedGuests,
			Over: e.Over, Step: e.Step, Steps: e.Steps, NoGuests: e.NoGuests,
		}
		if showAmount {
			price := e.StepPrice
			ex.StepPrice = &price
		}
		out.ExtraGuests = ex
	}
	return out
}

func toLineTotals(v *application.LinesView) *lineTotalsResponse {
	t := v.Totals
	return &lineTotalsResponse{Gross: t.Gross, Discount: t.Discount, Subtotal: t.Subtotal, Tax: t.Tax, Total: t.Total}
}

// toLinesResponse is the staff answer; canMargin is finance.margin.view.
func toLinesResponse(v *application.LinesView, canMargin bool) linesResponse {
	out := linesResponse{
		ProjectID: v.ProjectID, Lines: make([]lineResponse, 0, len(v.Lines)), Totals: toLineTotals(v), IsPKP: v.IsPKP,
		LinesVersion: etag.Format(v.LinesVersion),
	}
	termsTotal, hasTerms, mismatch := v.TermsTotal, v.HasTerms, v.TermsMismatch
	out.TermsTotal, out.HasTerms, out.TermsMismatch = &termsTotal, &hasTerms, &mismatch
	for _, l := range v.Lines {
		out.Lines = append(out.Lines, toLineResponse(l, v.Lines, true))
	}
	if canMargin {
		m := v.Margin
		out.Margin = &lineMarginResponse{
			Subtotal: m.Subtotal, SupplierCost: m.SupplierCost, MissingCost: m.MissingCost,
			Incomplete: m.MissingCost > 0, Margin: m.Margin,
		}
	}
	return out
}

// toPortalLinesResponse is the "Layanan › Paket & rincian" of the Client
// Portal: every line through PortalPolicy.Line (no supplier, no component
// price, no amount at all without portal.finance.view).
func toPortalLinesResponse(v *application.LinesView, policy domain.PortalPolicy) linesResponse {
	out := linesResponse{ProjectID: v.ProjectID, Lines: make([]lineResponse, 0, len(v.Lines)), IsPKP: v.IsPKP}
	for _, l := range v.Lines {
		pl, ok := policy.Line(l.ProjectLine)
		if !ok {
			continue
		}
		l.ProjectLine = pl.Line
		item := toLineResponse(l, v.Lines, pl.ShowAmount)
		item.UpdatedAt = ""
		out.Lines = append(out.Lines, item)
	}
	if policy.Finance() {
		out.Totals = toLineTotals(v)
	}
	return out
}

func (h *Handler) linesAnswer(claims staffClaims, v *application.LinesView) linesResponse {
	return toLinesResponse(v, claims.canSeeMargin())
}

// renderLines is the stale answer's data.current.
func (h *Handler) renderLines(claims staffClaims) func(current any) (any, error) {
	return func(current any) (any, error) {
		if v, ok := current.(*application.LinesView); ok && v != nil {
			return h.linesAnswer(claims, v), nil
		}
		return nil, nil
	}
}

// --- requests ---

// lineDraftBody is one line as an editor sends it. A new line has no id (it
// may carry a key its components point at with parentKey); a component of a
// stored package sends parentId. kind is Package | Component | Item | Discount
// (ExtraGuests lines are the server's). A new Package sends packageId; its
// components come from the catalog when none are sent. An Item sends quantity
// and unitPrice (isBonus forces the price to 0); a Discount sends amount.
type lineDraftBody struct {
	ID          int64  `json:"id"`
	Key         string `json:"key"`
	ParentID    int64  `json:"parentId"`
	ParentKey   string `json:"parentKey"`
	Kind        string `json:"kind"`
	PackageID   int64  `json:"packageId"`
	Description string `json:"description"`
	CategoryID  *int64 `json:"categoryId"`
	EventID     *int64 `json:"eventId"`
	Quantity    *int64 `json:"quantity"`
	Unit        string `json:"unit"`
	UnitPrice   *int64 `json:"unitPrice"`
	Amount      int64  `json:"amount"`
	IsBonus     bool   `json:"isBonus"`
	SupplierRef *int64 `json:"supplierRef"`
}

func (b lineDraftBody) draft() application.LineDraft {
	return application.LineDraft{
		ID: b.ID, Key: b.Key, ParentID: b.ParentID, ParentKey: b.ParentKey, Kind: domain.LineKind(b.Kind),
		PackageID: b.PackageID, Description: b.Description, CategoryID: b.CategoryID, EventID: b.EventID,
		Quantity: b.Quantity, Unit: b.Unit, UnitPrice: b.UnitPrice, Amount: b.Amount, IsBonus: b.IsBonus,
		SupplierRef: b.SupplierRef,
	}
}

type linesBody struct {
	Lines []lineDraftBody `json:"lines"`
}

func (b linesBody) drafts() []application.LineDraft {
	out := make([]application.LineDraft, 0, len(b.Lines))
	for _, l := range b.Lines {
		out = append(out, l.draft())
	}
	return out
}

// projectExists is the existence check a missing If-Match asks first.
func (h *Handler) projectExists(tenantID, projectID int64) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		_, err := h.projects.Get(ctx, tenantID, projectID)
		return err
	}
}

// --- handlers ---

// listLines (GET .../lines): staff see everything they may; a client sees
// the portal's redacted Rincian.
func (h *Handler) listLines(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	v, err := h.lines.View(r.Context(), claims.tenantID, projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if claims.isClient {
		response.OK(w, "ok", toPortalLinesResponse(v, domain.NewPortalPolicy(claims.can)))
		return
	}
	response.OK(w, "ok", h.linesAnswer(claims, v))
}

// previewLines (POST .../lines/preview) computes the drafts with the same
// functions a save uses and writes nothing.
func (h *Handler) previewLines(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body linesBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeBadBody(w)
		return
	}
	v, err := h.lines.Preview(r.Context(), claims.tenantID, projectID, body.drafts())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", h.linesAnswer(claims, v))
}

// replaceLines (PUT .../lines) saves the whole Rincian as the editor holds it.
func (h *Handler) replaceLines(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	expected, ok := requireIfMatch(w, r, h.projectExists(claims.tenantID, projectID))
	if !ok {
		return
	}
	var body linesBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeBadBody(w)
		return
	}
	v, err := h.lines.Replace(r.Context(), claims.tenantID, projectID, claims.staffID, body.drafts(), expected)
	if err != nil {
		writePatchError(w, err, h.renderLines(claims))
		return
	}
	response.OK(w, "Rincian disimpan", h.mutated(r.Context(), claims.tenantID, projectID, h.linesAnswer(claims, v)))
}

// addLine (POST .../lines) adds one line.
func (h *Handler) addLine(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	expected, ok := requireIfMatch(w, r, h.projectExists(claims.tenantID, projectID))
	if !ok {
		return
	}
	var body lineDraftBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeBadBody(w)
		return
	}
	v, err := h.lines.Add(r.Context(), claims.tenantID, projectID, claims.staffID, body.draft(), expected)
	if err != nil {
		writePatchError(w, err, h.renderLines(claims))
		return
	}
	response.Created(w, "Baris rincian ditambahkan", h.mutated(r.Context(), claims.tenantID, projectID, h.linesAnswer(claims, v)))
}

// patchLine (PATCH .../lines/{lineId}) changes the fields sent of one line.
func (h *Handler) patchLine(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, idRaw string) {
	lineID, err := parseInt64(idRaw)
	if err != nil || lineID <= 0 {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	expected, ok := requireIfMatch(w, r, h.projectExists(claims.tenantID, projectID))
	if !ok {
		return
	}
	p, ok := newPatchReader(w, r)
	if !ok {
		return
	}
	patch := application.LinePatch{
		Description: p.text("description", false),
		Unit:        p.text("unit", true),
		CategoryID:  p.integer("categoryId", true),
		EventID:     p.integer("eventId", true),
		Quantity:    p.integer("quantity", false),
		UnitPrice:   p.integer("unitPrice", false),
		Amount:      p.integer("amount", false),
		SupplierRef: p.integer("supplierRef", true),
		IsBonus:     p.boolean("isBonus"),
	}
	if err := p.finish(); err != nil {
		writeAppError(w, err)
		return
	}
	v, err := h.lines.Patch(r.Context(), claims.tenantID, projectID, lineID, claims.staffID, patch, expected)
	if err != nil {
		writePatchError(w, err, h.renderLines(claims))
		return
	}
	response.OK(w, "Baris rincian diubah", h.mutated(r.Context(), claims.tenantID, projectID, h.linesAnswer(claims, v)))
}

// reorderLines (POST .../lines/reorder, body {ids}) puts every line except
// the extra-guest charges in the order sent.
func (h *Handler) reorderLines(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	expected, ok := requireIfMatch(w, r, h.projectExists(claims.tenantID, projectID))
	if !ok {
		return
	}
	var body struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeBadBody(w)
		return
	}
	v, err := h.lines.Reorder(r.Context(), claims.tenantID, projectID, claims.staffID, body.IDs, expected)
	if err != nil {
		writePatchError(w, err, h.renderLines(claims))
		return
	}
	response.OK(w, "Urutan rincian disimpan", h.mutated(r.Context(), claims.tenantID, projectID, h.linesAnswer(claims, v)))
}

// deleteLine (DELETE .../lines/{lineId}) is a soft delete: a package takes
// its components and extra-guest charge with it (never a vendor, a venue or a
// payment). The answer is the Rincian plus deleted {id, restorableUntil}.
func (h *Handler) deleteLine(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, idRaw string) {
	lineID, err := parseInt64(idRaw)
	if err != nil || lineID <= 0 {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	expected, ok := requireIfMatch(w, r, h.projectExists(claims.tenantID, projectID))
	if !ok {
		return
	}
	v, until, err := h.lines.Delete(r.Context(), claims.tenantID, projectID, lineID, claims.staffID, expected, clock.Now())
	if err != nil {
		writePatchError(w, err, h.renderLines(claims))
		return
	}
	response.OK(w, "Baris rincian dihapus", h.mutated(r.Context(), claims.tenantID, projectID, h.linesAnswer(claims, v),
		extra{key: "deleted", value: toDeletedResponse(lineID, until)}))
}

// restoreLine (POST .../lines/{lineId}/restore) brings a deleted line back
// within the retention ("Urungkan"); 404 when it is not deleted or too late.
func (h *Handler) restoreLine(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, idRaw string) {
	lineID, err := parseInt64(idRaw)
	if err != nil || lineID <= 0 {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	v, err := h.lines.Restore(r.Context(), claims.tenantID, projectID, lineID, claims.staffID, clock.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Baris rincian dipulihkan", h.mutated(r.Context(), claims.tenantID, projectID, h.linesAnswer(claims, v)))
}
