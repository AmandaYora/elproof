//go:build integration

package isolation

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

// tenant is one signed-up workspace and its Owner session.
type tenant struct {
	name    string
	marker  string // unique text in every name this tenant creates
	token   string
	staffID int64
	tenant  int64
}

// world is what the suite builds once: workspace A, workspace B with a rich
// set of resources, and B's client login.
type world struct {
	a, b tenant

	// Workspace B's resources (ids) — the targets of every attack.
	Project, Project2, Sacrifice int64
	Category, Vendor, Venue      int64
	// Supplier is the directory vendor on Project, paid by the WO with its name
	// hidden from the client; VenueSupplier is the venue its Event brought in.
	Supplier, VendorMilestone          int64
	VenueSupplier, Event, Kind         int64
	ProjectMilestone                   int64
	Client, ClientPayment, SupplierPay int64
	VenuePay, Issue                    int64
	EvidenceInternal, EvidenceVisible  int64
	Role, Staff2, Template             int64
	Tx                                 int64
	ClientRole                         int64
	clientToken                        string
	bContract                          int64
	// Workspace A's own resources, so its lists are not empty.
	aProject, aVendor, aVenue, aKind int64
}

const bContractValue = 987654321

func (s *stack) signUp(name string) tenant {
	s.t.Helper()
	marker := fmt.Sprintf("RAHASIA-%s-%06d", name, rand.Intn(1_000_000))
	email := fmt.Sprintf("owner.%s.%d@example.com", name, time.Now().UnixNano())
	sub := fmt.Sprintf("g-sub-%s-%d", name, time.Now().UnixNano())
	cred := s.google.token(email, "Owner "+marker, sub)
	s.mustOK("signup "+name, s.do("", "POST", "/api/v1/public/signup/workspace", map[string]any{
		"credential": cred, "businessName": "WO " + marker, "acceptTerms": true,
	}))
	login := s.mustOK("login "+name, s.do("", "POST", "/api/v1/auth/google", map[string]any{"credential": cred}))
	tok, _ := login.dig("data", "accessToken").(string)
	if tok == "" {
		s.t.Fatalf("no access token for %s: %s", name, login)
	}
	staffID, _ := strconv.ParseInt(fmt.Sprint(login.dig("data", "principalId")), 10, 64)
	return tenant{name: name, marker: marker, token: tok, staffID: staffID, tenant: login.id("data", "tenantId")}
}

func day(offset int) string { return time.Now().AddDate(0, 0, offset).Format("2006-01-02") }

// kind is the id of one of the workspace's event kinds ("Akad", "Resepsi", ...).
func (s *stack) kind(t tenant, name string) int64 {
	s.t.Helper()
	for _, k := range rows(s.mustOK("event kinds", s.do(t.token, "GET", "/api/v1/event-kinds", nil)), "data") {
		if k["name"] == name {
			return int64(k["id"].(float64))
		}
	}
	s.t.Fatalf("workspace %s has no event kind %q", t.name, name)
	return 0
}

// eventOf is one event of a project body: a single session of `guests`, at the
// directory venue `venue` (0: no venue yet).
func eventOf(kind int64, date string, guests int, venue int64) map[string]any {
	e := map[string]any{
		"eventKindId": kind, "eventDate": date, "layout": "Standing",
		"sessions": []map[string]any{{"label": "Sesi 1", "guests": guests}},
	}
	if venue != 0 {
		e["venueId"] = venue
	}
	return e
}

// projectBody is a new project (POST /projects without an offer) with the
// given events; with none, one Akad `eventIn` days away without a venue. A
// project carries no package or contract value of its own (Paket Tahap 3): its
// contract comes from its Rincian, see setContract.
func (s *stack) projectBody(t tenant, name string, eventIn int, events ...map[string]any) map[string]any {
	s.t.Helper()
	if len(events) == 0 {
		events = []map[string]any{eventOf(s.kind(t, "Akad"), day(eventIn), 100, 0)}
	}
	return map[string]any{
		"name": name, "brideName": "Bride " + t.marker, "groomName": "Groom " + t.marker,
		"prepStartDate": day(-30), "status": "Preparation",
		"picStaffId": t.staffID, "description": "Catatan " + t.marker, "events": events,
	}
}

// linesVersion is the project's current Rincian version, the If-Match of every
// line write.
func (s *stack) linesVersion(t tenant, projectID int64) string {
	s.t.Helper()
	v, _ := s.mustOK("lines", s.do(t.token, "GET", s.projectsPath(projectID, "/lines"), nil)).dig("data", "linesVersion").(string)
	if v == "" {
		s.t.Fatalf("project %d has no linesVersion", projectID)
	}
	return v
}

// setContract gives a project its contract value the way the WO does: a
// Rincian of one Item at that price. A new workspace is not PKP, so the
// contract total is exactly `contract`. 0 leaves the Rincian empty.
func (s *stack) setContract(t tenant, projectID, contract int64) {
	s.t.Helper()
	if contract <= 0 {
		return
	}
	s.mustOK("rincian", s.doWith(t.token, "PUT", s.projectsPath(projectID, "/lines"), map[string]any{
		"lines": []map[string]any{{"kind": "Item", "description": "Paket " + t.marker, "quantity": 1, "unit": "paket", "unitPrice": contract}},
	}, map[string]string{"If-Match": quoted(s.linesVersion(t, projectID))}))
}

// createProject creates a project and gives it a Rincian worth `contract`.
func (s *stack) createProject(t tenant, name string, contract int64) int64 {
	s.t.Helper()
	id := s.mustOK("project "+name, s.do(t.token, "POST", "/api/v1/projects", s.projectBody(t, name, 120))).id("data", "id")
	s.setContract(t, id, contract)
	return id
}

// projectAt creates a project whose main event is `eventIn` days away, at the
// directory venue `venue` when it is not 0 (the venue then joins the project's
// suppliers, paid by the WO, without an agreed cost), with a Rincian worth
// `contract`.
func (s *stack) projectAt(t tenant, name string, contract int64, eventIn int, venue int64) int64 {
	s.t.Helper()
	body := s.projectBody(t, name, eventIn, eventOf(s.kind(t, "Akad"), day(eventIn), 100, venue))
	id := s.mustOK("project "+name, s.do(t.token, "POST", "/api/v1/projects", body)).id("data", "id")
	s.setContract(t, id, contract)
	return id
}

// engage adds a directory vendor to a project, paid by the WO with its name
// hidden from the client; agreedCost 0 leaves the cost out.
func (s *stack) engage(t tenant, projectID, vendorID int64, scope string, agreedCost int64, status string, dueIn *int) int64 {
	s.t.Helper()
	body := map[string]any{"vendorId": vendorID, "paidBy": "WO", "scope": scope, "engagementStatus": status, "picStaffId": t.staffID}
	if agreedCost > 0 {
		body["agreedCost"] = agreedCost
	}
	if dueIn != nil {
		body["dueDate"] = day(*dueIn)
	}
	return s.mustOK("supplier "+scope, s.do(t.token, "POST", s.projectsPath(projectID, "/suppliers"), body)).id("data", "id")
}

// engageBy is engage with an explicit payer ("WO" or "Client") and name rule.
func (s *stack) engageBy(t tenant, projectID, vendorID int64, scope, paidBy string, showName bool) int64 {
	s.t.Helper()
	return s.mustOK("supplier "+scope, s.do(t.token, "POST", s.projectsPath(projectID, "/suppliers"), map[string]any{
		"vendorId": vendorID, "paidBy": paidBy, "showName": showName, "scope": scope, "engagementStatus": "Booked", "picStaffId": t.staffID,
	})).id("data", "id")
}

// venueSupplier is the project's supplier row for the directory venue `venue`
// (0 when the project has none).
func (s *stack) venueSupplier(t tenant, projectID, venue int64) int64 {
	s.t.Helper()
	for _, m := range rows(s.mustOK("suppliers", s.do(t.token, "GET", s.projectsPath(projectID, "/suppliers"), nil)), "data") {
		if m["kind"] == "venue" && m["supplierId"] == float64(venue) {
			return int64(m["id"].(float64))
		}
	}
	return 0
}

// supplierPay records a payment to one of the project's suppliers; the payer is
// the supplier's.
func (s *stack) supplierPay(t tenant, projectID, supplierID int64, typ string, amount int64, ref string) int64 {
	s.t.Helper()
	return s.mustOK("supplier payment "+ref, s.do(t.token, "POST", s.projectsPath(projectID, "/supplier-payments"), map[string]any{
		"projectSupplierId": supplierID, "type": typ, "amount": amount, "paymentDate": day(-2), "method": "Transfer",
		"referenceNumber": ref, "notes": "Rahasia " + t.marker,
	})).id("data", "id")
}

// pdf is a tiny but real-looking file for evidence uploads.
var pdf = base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n1 0 obj<<>>endobj\ntrailer<<>>\n%%EOF"))

func (s *stack) upload(t tenant, projectID int64, name string, visible bool) int64 {
	s.t.Helper()
	r := s.mustOK("evidence "+name, s.do(t.token, "POST", fmt.Sprintf("/api/v1/projects/%d/evidence", projectID), map[string]any{
		"name": name, "type": "Document", "fileName": "dokumen.pdf", "mimeType": "application/pdf", "base64Data": pdf,
		"documentDate": day(-1), "description": "Rahasia " + t.marker, "relatedKind": "general", "isClientVisible": visible,
	}))
	return r.id("data", "id")
}

// buildWorld signs up both workspaces and builds B's resources.
func buildWorld(t *testing.T, s *stack) *world {
	t.Helper()
	w := &world{a: s.signUp("a"), b: s.signUp("b"), bContract: bContractValue}
	a, b := w.a, w.b

	// ---- workspace A: some data of its own ----
	w.aKind = s.kind(a, "Akad")
	w.aProject = s.createProject(a, "Proyek A "+a.marker, 111111111)
	w.aVendor = s.mustOK("vendor A", s.do(a.token, "POST", "/api/v1/vendors", vendorBody(s, a, "Vendor A "+a.marker, 0))).id("data", "id")
	w.aVenue = s.mustOK("venue A", s.do(a.token, "POST", "/api/v1/venues", venueBody(a, "Venue A "+a.marker))).id("data", "id")

	// ---- workspace B ----
	w.Category = s.mustOK("category", s.do(b.token, "POST", "/api/v1/vendor-categories", map[string]any{
		"name": "Kategori " + b.marker, "description": "Rahasia " + b.marker,
	})).id("data", "id")
	w.Vendor = s.mustOK("vendor", s.do(b.token, "POST", "/api/v1/vendors", vendorBody(s, b, "Vendor B "+b.marker, w.Category))).id("data", "id")
	w.Venue = s.mustOK("venue", s.do(b.token, "POST", "/api/v1/venues", venueBody(b, "Venue B "+b.marker))).id("data", "id")
	w.Kind = s.kind(b, "Akad")

	// B's main project: its Akad takes place at B's venue, which brings the venue in.
	created := s.mustOK("project B", s.do(b.token, "POST", "/api/v1/projects",
		s.projectBody(b, "Proyek B "+b.marker, 120, eventOf(w.Kind, day(120), 100, w.Venue))))
	w.Project = created.id("data", "id")
	s.setContract(b, w.Project, bContractValue)
	if evs := rows(created, "data", "events"); len(evs) == 1 {
		w.Event = int64(evs[0]["id"].(float64))
	}
	w.Project2 = s.createProject(b, "Proyek B2 "+b.marker, 222222222)
	w.Sacrifice = s.createProject(b, "Proyek B3 "+b.marker, 333333333)
	w.VenueSupplier = s.venueSupplier(b, w.Project, w.Venue)
	if w.Event == 0 || w.VenueSupplier == 0 {
		t.Fatalf("B's project has no event (%d) or venue supplier (%d): %s", w.Event, w.VenueSupplier, created)
	}

	w.Supplier = s.mustOK("supplier", s.do(b.token, "POST", fmt.Sprintf("/api/v1/projects/%d/suppliers", w.Project), map[string]any{
		"vendorId": w.Vendor, "paidBy": "WO", "showName": false, "agreedCost": 5000000, "engagementStatus": "Booked",
		"dueDate": day(60), "picStaffId": b.staffID, "scope": "Dekorasi " + b.marker, "notes": "Rahasia " + b.marker,
	})).id("data", "id")
	w.VendorMilestone = s.mustOK("vendor milestone", s.do(b.token, "POST", fmt.Sprintf("/api/v1/projects/%d/suppliers/%d/milestones", w.Project, w.Supplier), map[string]any{
		"name": "Milestone vendor " + b.marker, "description": "Rahasia", "targetDate": day(30), "picStaffId": b.staffID,
	})).id("data", "id")

	ms := s.mustOK("milestones", s.do(b.token, "GET", fmt.Sprintf("/api/v1/projects/%d/milestones", w.Project), nil))
	if list, ok := ms.dig("data").([]any); ok && len(list) > 0 {
		if m, ok := list[0].(map[string]any); ok {
			if f, ok := m["id"].(float64); ok {
				w.ProjectMilestone = int64(f)
			}
		}
	}
	if w.ProjectMilestone == 0 {
		t.Fatalf("project has no milestone to target: %s", ms)
	}

	w.ClientPayment = s.mustOK("client payment", s.do(b.token, "POST", fmt.Sprintf("/api/v1/projects/%d/client-payments", w.Project), map[string]any{
		"type": "DP", "amount": 4444444, "paymentDate": day(-2), "method": "Transfer", "referenceNumber": "REF-" + b.marker, "notes": "Rahasia " + b.marker,
	})).id("data", "id")
	w.SupplierPay = s.supplierPay(b, w.Project, w.Supplier, "DP", 1000000, "VREF-"+b.marker)
	w.VenuePay = s.supplierPay(b, w.Project, w.VenueSupplier, "DP", 777777, "VNREF-"+b.marker)
	w.Issue = s.mustOK("issue", s.do(b.token, "POST", fmt.Sprintf("/api/v1/projects/%d/issues", w.Project), map[string]any{
		"projectSupplierId": w.Supplier, "title": "Kendala " + b.marker, "description": "Rahasia " + b.marker, "impact": "Medium",
		"foundDate": day(-1), "resolutionPlan": "Rencana " + b.marker, "picStaffId": b.staffID, "targetResolutionDate": day(10),
	})).id("data", "id")
	s.mustOK("payment terms", s.do(b.token, "PUT", fmt.Sprintf("/api/v1/projects/%d/payment-terms", w.Project), map[string]any{
		"terms": []map[string]any{{"label": "Termin " + b.marker, "amount": 1000000, "dueDate": day(30)}},
	}))
	w.EvidenceInternal = s.upload(b, w.Project, "Dokumen internal "+b.marker, false)
	w.EvidenceVisible = s.upload(b, w.Project, "Dokumen klien "+b.marker, true)

	w.Role = s.mustOK("role", s.do(b.token, "POST", "/api/v1/access/roles", map[string]any{
		"kind": "staff", "name": "Peran " + b.marker, "description": "Rahasia", "projectScope": "all", "permissions": []string{"project.edit"},
	})).id("data", "id")
	w.Template = s.mustOK("template", s.do(b.token, "POST", "/api/v1/milestone-templates", map[string]any{
		"name": "Template " + b.marker, "daysBeforeEvent": 45,
	})).id("data", "id")
	st := s.mustOK("staff", s.do(b.token, "POST", "/api/v1/staff", map[string]any{
		"name": "Staf " + b.marker, "title": "Koordinator", "roleId": w.Role, "email": fmt.Sprintf("staf.b.%d@example.com", time.Now().UnixNano()), "phone": "081234567890",
	}))
	w.Staff2 = st.id("data", "staff", "id")

	// B's client, with a real login (the invitation link sets the password).
	roles := s.mustOK("roles", s.do(b.token, "GET", "/api/v1/access/roles?kind=client", nil))
	if list, ok := roles.dig("data").([]any); ok && len(list) > 0 {
		if m, ok := list[0].(map[string]any); ok {
			if f, ok := m["id"].(float64); ok {
				w.ClientRole = int64(f)
			}
		}
	}
	cl := s.mustOK("client", s.do(b.token, "POST", "/api/v1/clients", map[string]any{
		"projectId": w.Project, "role": "Bride", "relationNote": "Rahasia " + b.marker, "name": "Klien " + b.marker,
		"phone": "081298765432", "username": fmt.Sprintf("klien.b.%d", time.Now().UnixNano()%1_000_000_000), "email": "", "accessRoleId": w.ClientRole,
	}))
	w.Client = cl.id("data", "client", "id")
	linkToken, _ := cl.dig("data", "accessLink", "token").(string)
	sess := s.mustOK("client password", s.do("", "POST", "/api/v1/auth/links/"+linkToken+"/password", map[string]any{"password": "Kata-sandi-klien-123"}))
	w.clientToken, _ = sess.dig("data", "accessToken").(string)
	if w.clientToken == "" {
		t.Fatalf("no client token: %s", sess)
	}

	// A paid subscription transaction of B (no API creates one without a gateway).
	res, err := s.db.Exec(`INSERT INTO subscription_transactions (tenant_id, type, amount, payment_method, payment_reference, status, paid_at)
		VALUES (?, 'new', 2000000, 'QRIS', ?, 'paid', NOW())`, b.tenant, "ELP-"+b.marker)
	if err != nil {
		t.Fatalf("insert transaction: %v", err)
	}
	w.Tx, _ = res.LastInsertId()
	return w
}

func vendorBody(s *stack, t tenant, name string, categoryID int64) map[string]any {
	if categoryID == 0 {
		// Any seeded category of the tenant will do.
		r := s.mustOK("categories", s.do(t.token, "GET", "/api/v1/vendor-categories?all=true", nil))
		if list, ok := r.dig("data").([]any); ok && len(list) > 0 {
			if m, ok := list[0].(map[string]any); ok {
				if f, ok := m["id"].(float64); ok {
					categoryID = int64(f)
				}
			}
		}
	}
	return map[string]any{
		"name": name, "categoryId": categoryID, "picName": "PIC " + t.marker, "phone": "081211112222", "email": "",
		"socialMedia": "@x", "city": "Kota Bandung", "address": "Alamat " + t.marker, "priceAkad": 1000000, "priceAkadResepsi": 2000000, "priceResepsi": 1500000,
		"notes": "Rahasia " + t.marker,
	}
}

func venueBody(t tenant, name string) map[string]any {
	return map[string]any{
		"name": name, "picName": "PIC " + t.marker, "phonePic": "081233334444", "phoneVenue": "0221234567", "email": "",
		"address": "Alamat " + t.marker, "city": "Kota Bandung", "priceAkadResepsi": 10000000, "charge": 500000, "capacityStanding": 300,
		"facilities": "AC", "socialMedia": "@v", "notes": "Rahasia " + t.marker,
	}
}
