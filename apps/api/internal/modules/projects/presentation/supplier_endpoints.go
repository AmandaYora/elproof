package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/clock"
	"elproof/internal/shared/response"
)

// Vendor & venue of a project (PLAN §3.1):
//
//	GET    /projects/{id}/suppliers                          list (staff; client: Layanan)
//	POST   /projects/{id}/suppliers                          add a directory vendor
//	PATCH  /projects/{id}/suppliers/{sid}                    partial update (If-Match)
//	DELETE /projects/{id}/suppliers/{sid}                    soft delete (24 h restore)
//	POST   /projects/{id}/suppliers/{sid}/restore
//	POST   /projects/{id}/suppliers/{sid}/refresh-capacity   venue: copy the directory's capacities
//	POST   /projects/{id}/suppliers/{sid}/milestones         add a step
//	PATCH  /projects/{id}/suppliers/{sid}/milestones/{mid}   partial update of a step (If-Match)
//
// Venues come in through events (POST/PUT .../events), never here.
func (h *Handler) routeSuppliers(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, rest []string) {
	switch {
	case len(rest) == 0 && r.Method == http.MethodGet:
		h.listSuppliers(w, r, claims, projectID)
	case len(rest) == 0 && r.Method == http.MethodPost:
		h.createSupplier(w, r, claims, projectID)
	case len(rest) == 1 && r.Method == http.MethodPatch:
		h.updateSupplier(w, r, claims, projectID, rest[0])
	case len(rest) == 1 && r.Method == http.MethodDelete:
		h.deleteSupplier(w, r, claims, projectID, rest[0])
	case len(rest) == 2 && rest[1] == "restore" && r.Method == http.MethodPost:
		h.restoreSupplier(w, r, claims, projectID, rest[0])
	case len(rest) == 2 && rest[1] == "refresh-capacity" && r.Method == http.MethodPost:
		h.refreshSupplierCapacity(w, r, claims, projectID, rest[0])
	case len(rest) == 2 && rest[1] == "milestones" && r.Method == http.MethodPost:
		h.createVendorMilestone(w, r, claims, projectID, rest[0])
	case len(rest) == 3 && rest[1] == "milestones" && r.Method == http.MethodPatch:
		h.updateVendorMilestone(w, r, claims, projectID, rest[0], rest[2])
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

// canSeeMargin: the agreed cost, what the WO paid its suppliers and those
// payments are for finance.margin.view only (PLAN §4.3).
func (c staffClaims) canSeeMargin() bool { return !c.isClient && c.can("finance.margin.view") }

func (h *Handler) listSuppliers(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	list, err := h.suppliers.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	rows, err := h.supplierRows(r.Context(), claims, projectID, list)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", rows)
}

// supplierPaid is what each supplier has been paid, by payer (DP + Termin +
// Pelunasan - Refund, counting only what CountsAsPaid), and the ids of its
// live payments.
type supplierPaid struct {
	wo, client map[int64]int64
	ids        map[int64][]int64
}

func (h *Handler) supplierPaidAmounts(ctx context.Context, projectID int64) (supplierPaid, error) {
	out := supplierPaid{wo: map[int64]int64{}, client: map[int64]int64{}, ids: map[int64][]int64{}}
	payments, err := h.supplierPayments.List(ctx, projectID)
	if err != nil {
		return out, err
	}
	lines := map[domain.Payer]map[int64][]domain.PaymentLine{domain.PayerWO: {}, domain.PayerClient: {}}
	for _, p := range payments {
		out.ids[p.ProjectSupplierID] = append(out.ids[p.ProjectSupplierID], p.ID)
		// A client's proof the WO has not marked right is not paid yet.
		if byPayer := lines[p.Payer]; byPayer != nil && p.CountsAsPaid() {
			byPayer[p.ProjectSupplierID] = append(byPayer[p.ProjectSupplierID], domain.PaymentLine{Type: p.Type, Amount: p.Amount})
		}
	}
	for id, l := range lines[domain.PayerWO] {
		out.wo[id] = domain.AppliedToContract(l)
	}
	for id, l := range lines[domain.PayerClient] {
		out.client[id] = domain.AppliedToContract(l)
	}
	return out, nil
}

// supplierRows builds the list rows (names, PIC, steps with their proof, paid
// amounts, lock message, events at a venue) for the given suppliers of a
// project from one query per source. A client gets the portal's view of each
// (PortalPolicy): the service instead of a hidden vendor's name, no money, no
// internal notes, only the files it may see.
func (h *Handler) supplierRows(ctx context.Context, claims staffClaims, projectID int64, list []domain.ProjectSupplier) ([]supplierResponse, error) {
	view, err := h.portalViewFor(ctx, claims, projectID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(list))
	categoryIDs, picIDs := make([]int64, 0, len(list)), make([]int64, 0, len(list))
	for i, s := range list {
		ids[i] = s.ID
		categoryIDs, picIDs = append(categoryIDs, s.CategoryID), append(picIDs, s.PICStaffID)
	}
	steps, err := h.suppliers.MilestonesBySupplier(ctx, ids)
	if err != nil {
		return nil, err
	}
	evidences, err := h.evidence.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	ix := indexEvidence(evidences, view)
	categoryNames := h.refs.CategoryNames(ctx, claims.tenantID, categoryIDs)
	var (
		paid        supplierPaid
		attachments map[int64]domain.SupplierAttachments
		usedBy      = map[int64]int{}
		picNames    map[int64]string
	)
	if !claims.isClient {
		if paid, err = h.supplierPaidAmounts(ctx, projectID); err != nil {
			return nil, err
		}
		if attachments, err = h.suppliers.Attachments(ctx, projectID); err != nil {
			return nil, err
		}
		events, err := h.events.List(ctx, claims.tenantID, projectID)
		if err != nil {
			return nil, err
		}
		for _, e := range events {
			if e.VenueSupplierID != nil {
				usedBy[*e.VenueSupplierID]++
			}
		}
		picNames = h.projects.StaffNames(ctx, claims.tenantID, picIDs)
	}
	out := make([]supplierResponse, 0, len(list))
	for _, s := range list {
		resp := toSupplierResponse(s)
		resp.CategoryName = categoryNames[s.CategoryID]
		resp.EvidenceCount = len(ix.refs(domain.RelatedProjectSupplier, s.ID))
		for _, m := range steps[s.ID] {
			mr := toVendorMilestoneResponse(m)
			mr.Evidence = ix.refs(domain.RelatedVendorMilestone, m.ID)
			resp.EvidenceCount += len(mr.Evidence)
			resp.Milestones = append(resp.Milestones, mr)
		}
		if view.forClient() {
			redactSupplierForClient(&resp, view.policy.SupplierName(s, resp.CategoryName))
			out = append(out, resp)
			continue
		}
		for _, payID := range paid.ids[s.ID] {
			resp.EvidenceCount += len(ix.refs(domain.RelatedSupplierPayment, payID))
		}
		resp.PICName = picNames[s.PICStaffID]
		resp.UsedByEvents = usedBy[s.ID]
		a := attachments[s.ID]
		resp.PaidByLocked = a.PaidByLock(s.PaidBy)
		if s.PaidBy == domain.PayerClient {
			v := paid.client[s.ID]
			resp.PaidClient = &v
		}
		if claims.canSeeMargin() {
			if s.PaidBy == domain.PayerWO {
				v := paid.wo[s.ID]
				resp.PaidWO = &v
			}
		} else {
			resp.AgreedCost = nil
		}
		out = append(out, resp)
	}
	return out, nil
}

// supplierItem is one supplier as the list shows it.
func (h *Handler) supplierItem(ctx context.Context, claims staffClaims, projectID int64, s domain.ProjectSupplier) (supplierResponse, error) {
	rows, err := h.supplierRows(ctx, claims, projectID, []domain.ProjectSupplier{s})
	if err != nil {
		return supplierResponse{}, err
	}
	return rows[0], nil
}

type supplierInputBody struct {
	VendorID         int64  `json:"vendorId"`
	PaidBy           string `json:"paidBy"`
	ShowName         bool   `json:"showName"`
	AgreedCost       *int64 `json:"agreedCost"`
	EngagementStatus string `json:"engagementStatus"`
	BookingDate      string `json:"bookingDate"`
	DueDate          string `json:"dueDate"`
	PICStaffID       int64  `json:"picStaffId"`
	Scope            string `json:"scope"`
	Notes            string `json:"notes"`
}

func parseOptionalDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := parseDate(s)
	if err != nil {
		return nil
	}
	return &t
}

// createSupplier adds a directory vendor to the project (project.edit). Its
// name and category are copied from the directory; an inactive vendor cannot
// be picked. The agreed cost needs finance.margin.view. Every problem is
// reported together.
func (h *Handler) createSupplier(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body supplierInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	ctx := r.Context()
	if body.AgreedCost != nil && !claims.canSeeMargin() {
		forbiddenPermission(w, "Peran Anda tidak dapat mengisi biaya rekanan")
		return
	}
	fields := map[string][]string{}
	ref, err := h.refs.SupplierVendor(ctx, claims.tenantID, "vendorId", body.VendorID)
	if err != nil && !collectValidation(fields, err) {
		writeAppError(w, err)
		return
	}
	if err := h.refs.Staff(ctx, claims.tenantID, "picStaffId", body.PICStaffID); err != nil && !collectValidation(fields, err) {
		writeAppError(w, err)
		return
	}
	for key, raw := range map[string]string{"bookingDate": body.BookingDate, "dueDate": body.DueDate} {
		if raw != "" && parseOptionalDate(raw) == nil {
			fields[key] = append(fields[key], "Gunakan format YYYY-MM-DD")
		}
	}
	if len(fields) > 0 {
		msg := "Data vendor belum lengkap atau tidak sesuai"
		for _, k := range []string{"vendorId", "picStaffId", "bookingDate", "dueDate"} {
			if m := fields[k]; len(m) > 0 {
				msg = m[0]
				break
			}
		}
		response.Error(w, http.StatusUnprocessableEntity, msg, fields)
		return
	}
	sup, err := h.suppliers.Create(ctx, projectID, claims.staffID, application.SupplierInput{
		VendorID: ref.ID, Name: ref.Name, CategoryID: ref.CategoryID, PaidBy: domain.Payer(body.PaidBy), ShowName: body.ShowName,
		AgreedCost: body.AgreedCost, EngagementStatus: domain.EngagementStatus(body.EngagementStatus),
		BookingDate: parseOptionalDate(body.BookingDate), DueDate: parseOptionalDate(body.DueDate),
		PICStaffID: body.PICStaffID, Scope: body.Scope, Notes: body.Notes,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	item, err := h.supplierItem(ctx, claims, projectID, *sup)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Vendor berhasil ditambahkan ke proyek", h.mutated(ctx, claims.tenantID, projectID, item))
}

// collectValidation folds a validation error's fields into fields and reports
// true; for any other error it changes nothing and reports false.
func collectValidation(fields map[string][]string, err error) bool {
	ae, ok := apperror.As(err)
	if !ok || ae.Kind != apperror.KindValidation {
		return false
	}
	for k, v := range ae.Fields {
		fields[k] = append(fields[k], v...)
	}
	return true
}

// updateSupplier is PATCH /projects/{id}/suppliers/{sid}: a partial update of
// paidBy, showName, agreedCost (nullable; finance.margin.view), engagementStatus,
// bookingDate and dueDate (nullable), picStaffId, scope and notes (nullable).
// "Dibayar oleh" is refused (422 paid_by_locked) once a payment or a file hangs
// off the supplier. See updateProject for the body, If-Match (428 without) and
// 409 rules.
func (h *Handler) updateSupplier(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, sidRaw string) {
	sid, err := parseInt64(sidRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	expected, ok := requireIfMatch(w, r, func(ctx context.Context) error {
		_, err := h.suppliers.Get(ctx, projectID, sid)
		return err
	})
	if !ok {
		return
	}
	pr, ok := newPatchReader(w, r)
	if !ok {
		return
	}
	if pr.has("agreedCost") && !claims.canSeeMargin() {
		forbiddenPermission(w, "Peran Anda tidak dapat mengubah biaya rekanan")
		return
	}
	paidBy, status := pr.text("paidBy", false), pr.text("engagementStatus", false)
	patch := application.SupplierPatch{
		ShowName: pr.boolean("showName"), AgreedCost: pr.integer("agreedCost", true),
		BookingDate: pr.date("bookingDate", true), DueDate: pr.date("dueDate", true),
		PICStaffID: pr.integer("picStaffId", false), Scope: pr.text("scope", false), Notes: pr.text("notes", true),
	}
	if paidBy.Present() {
		patch.PaidBy = application.Some(domain.Payer(paidBy.Value()))
	}
	if status.Present() {
		patch.EngagementStatus = application.Some(domain.EngagementStatus(status.Value()))
	}
	if err := pr.finish(); err != nil {
		writeAppError(w, err)
		return
	}
	if patch.PICStaffID.Present() {
		if err := h.refs.Staff(r.Context(), claims.tenantID, "picStaffId", patch.PICStaffID.Value()); err != nil {
			writeAppError(w, err)
			return
		}
	}
	sup, err := h.suppliers.Patch(r.Context(), projectID, sid, claims.staffID, patch, expected)
	if err != nil {
		writePatchError(w, err, func(cur any) (any, error) {
			return h.supplierItem(r.Context(), claims, projectID, *cur.(*domain.ProjectSupplier))
		})
		return
	}
	item, err := h.supplierItem(r.Context(), claims, projectID, *sup)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Vendor & venue berhasil diperbarui", h.mutated(r.Context(), claims.tenantID, projectID, item))
}

// deleteSupplier is a soft delete (project.edit): the supplier, its steps,
// payments, kendala and their files leave every list and total until restored.
// A venue an event still takes place at is refused (422 venue_in_use).
func (h *Handler) deleteSupplier(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, sidRaw string) {
	sid, err := parseInt64(sidRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	_, until, err := h.suppliers.Delete(r.Context(), projectID, sid, claims.staffID, clock.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Vendor & venue dihapus", h.mutated(r.Context(), claims.tenantID, projectID, toDeletedResponse(sid, until)))
}

// restoreSupplier brings a deleted supplier back with what hung off it; 404
// when it is not deleted or past the 24 hours.
func (h *Handler) restoreSupplier(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, sidRaw string) {
	sid, err := parseInt64(sidRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	sup, err := h.suppliers.Restore(r.Context(), projectID, sid, claims.staffID, clock.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	item, err := h.supplierItem(r.Context(), claims, projectID, *sup)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Vendor & venue dipulihkan", h.mutated(r.Context(), claims.tenantID, projectID, item))
}

// refreshSupplierCapacity copies the directory venue's current capacities into
// the project's venue ("Perbarui kapasitas"). The events keep what they are.
func (h *Handler) refreshSupplierCapacity(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, sidRaw string) {
	sid, err := parseInt64(sidRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	sup, err := h.suppliers.Get(r.Context(), projectID, sid)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if sup.Kind != domain.SupplierVenue {
		response.Error(w, http.StatusUnprocessableEntity, "Hanya venue yang punya kapasitas", map[string][]string{"kind": {"Hanya venue yang punya kapasitas"}})
		return
	}
	ref, err := h.refs.VenueRef(r.Context(), claims.tenantID, "supplierId", sup.SupplierID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	fresh, err := h.suppliers.RefreshCapacity(r.Context(), projectID, sid, claims.staffID, ref.CapacityStanding, ref.CapacitySeated)
	if err != nil {
		writeAppError(w, err)
		return
	}
	item, err := h.supplierItem(r.Context(), claims, projectID, *fresh)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Kapasitas venue diperbarui dari direktori", h.mutated(r.Context(), claims.tenantID, projectID, item))
}

// --- Steps of a supplier ---

type vendorMilestoneInputBody struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	TargetDate  string `json:"targetDate"`
	PICStaffID  int64  `json:"picStaffId"`
}

func (h *Handler) createVendorMilestone(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, sidRaw string) {
	sid, err := parseInt64(sidRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	var body vendorMilestoneInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	targetDate, err := parseDate(body.TargetDate)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"targetDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	if err := h.refs.Staff(r.Context(), claims.tenantID, "picStaffId", body.PICStaffID); err != nil {
		writeAppError(w, err)
		return
	}
	m, err := h.suppliers.CreateMilestone(r.Context(), projectID, sid, claims.staffID, application.VendorMilestoneInput{
		Name: body.Name, Description: body.Description, TargetDate: targetDate, PICStaffID: body.PICStaffID,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Tahapan vendor berhasil ditambahkan", h.mutated(r.Context(), claims.tenantID, projectID, toVendorMilestoneResponse(*m)))
}

// updateVendorMilestone is PATCH /projects/{id}/suppliers/{sid}/milestones/{mid}:
// a partial update (status, targetDate, completedDate (nullable), picStaffId,
// description, notes). See updateProject for the If-Match (428 without) and 409 rules.
func (h *Handler) updateVendorMilestone(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, sidRaw, milestoneIDRaw string) {
	sid, err := parseInt64(sidRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	milestoneID, err := parseInt64(milestoneIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tahapan tidak valid", nil)
		return
	}
	expected, ok := requireIfMatch(w, r, func(ctx context.Context) error {
		return h.suppliers.MilestoneExists(ctx, projectID, sid, milestoneID)
	})
	if !ok {
		return
	}
	pr, ok := newPatchReader(w, r)
	if !ok {
		return
	}
	status := pr.text("status", false)
	patch := application.VendorMilestonePatch{
		TargetDate: pr.date("targetDate", false), CompletedDate: pr.date("completedDate", true),
		PICStaffID: pr.integer("picStaffId", false), Description: pr.text("description", true), Notes: pr.text("notes", true),
	}
	if status.Present() {
		patch.Status = application.Some(domain.MilestoneStatus(status.Value()))
	}
	if err := pr.finish(); err != nil {
		writeAppError(w, err)
		return
	}
	if patch.PICStaffID.Present() {
		if err := h.refs.Staff(r.Context(), claims.tenantID, "picStaffId", patch.PICStaffID.Value()); err != nil {
			writeAppError(w, err)
			return
		}
	}
	m, err := h.suppliers.PatchMilestone(r.Context(), projectID, sid, milestoneID, claims.staffID, patch, expected, nowWIB())
	if err != nil {
		writePatchError(w, err, func(cur any) (any, error) {
			return toVendorMilestoneResponse(*cur.(*domain.VendorMilestone)), nil
		})
		return
	}
	response.OK(w, "Tahapan vendor berhasil diperbarui", h.mutated(r.Context(), claims.tenantID, projectID, toVendorMilestoneResponse(*m)))
}
