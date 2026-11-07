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
	Project, Project2, Sacrifice      int64
	Category, Vendor, Venue           int64
	Engagement, VendorMilestone       int64
	ProjectMilestone                  int64
	Client, ClientPayment, VendorPay  int64
	VenuePay, Issue                   int64
	EvidenceInternal, EvidenceVisible int64
	Role, Staff2, Template            int64
	Tx                                int64
	ClientRole                        int64
	clientToken                       string
	bContract                         int64
	// Workspace A's own resources, so its lists are not empty.
	aProject, aVendor, aVenue int64
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

func (s *stack) createProject(t tenant, name string, contract int64) int64 {
	s.t.Helper()
	r := s.mustOK("project "+name, s.do(t.token, "POST", "/api/v1/projects", map[string]any{
		"name": name, "brideName": "Bride " + t.marker, "groomName": "Groom " + t.marker,
		"eventDate": day(120), "venue": "Gedung " + t.marker, "prepStartDate": day(1),
		"packageName": "Paket " + t.marker, "contractValue": contract, "status": "Preparation",
		"picStaffId": t.staffID, "description": "Catatan " + t.marker,
	}))
	return r.id("data", "id")
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
	w.aProject = s.createProject(a, "Proyek A "+a.marker, 111111111)
	w.aVendor = s.mustOK("vendor A", s.do(a.token, "POST", "/api/v1/vendors", vendorBody(s, a, "Vendor A "+a.marker, 0))).id("data", "id")
	w.aVenue = s.mustOK("venue A", s.do(a.token, "POST", "/api/v1/venues", venueBody(a, "Venue A "+a.marker))).id("data", "id")

	// ---- workspace B ----
	w.Project = s.createProject(b, "Proyek B "+b.marker, bContractValue)
	w.Project2 = s.createProject(b, "Proyek B2 "+b.marker, 222222222)
	w.Sacrifice = s.createProject(b, "Proyek B3 "+b.marker, 333333333)

	w.Category = s.mustOK("category", s.do(b.token, "POST", "/api/v1/vendor-categories", map[string]any{
		"name": "Kategori " + b.marker, "description": "Rahasia " + b.marker,
	})).id("data", "id")
	w.Vendor = s.mustOK("vendor", s.do(b.token, "POST", "/api/v1/vendors", vendorBody(s, b, "Vendor B "+b.marker, w.Category))).id("data", "id")
	w.Venue = s.mustOK("venue", s.do(b.token, "POST", "/api/v1/venues", venueBody(b, "Venue B "+b.marker))).id("data", "id")

	w.Engagement = s.mustOK("engagement", s.do(b.token, "POST", fmt.Sprintf("/api/v1/projects/%d/vendors", w.Project), map[string]any{
		"vendorId": w.Vendor, "categoryId": w.Category, "scope": "Dekorasi " + b.marker, "contractValue": 5000000,
		"pricingTier": "Akad", "engagementStatus": "Booked", "eventDate": day(120), "dpAmount": 1000000, "dueDate": day(60),
		"picStaffId": b.staffID, "notes": "Rahasia " + b.marker,
	})).id("data", "id")
	w.VendorMilestone = s.mustOK("vendor milestone", s.do(b.token, "POST", fmt.Sprintf("/api/v1/projects/%d/vendors/%d/milestones", w.Project, w.Engagement), map[string]any{
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
	w.VendorPay = s.mustOK("vendor payment", s.do(b.token, "POST", fmt.Sprintf("/api/v1/projects/%d/payments", w.Project), map[string]any{
		"projectVendorId": w.Engagement, "type": "DP", "amount": 1000000, "paymentDate": day(-2), "method": "Transfer",
		"referenceNumber": "VREF-" + b.marker, "notes": "Rahasia " + b.marker,
	})).id("data", "id")
	w.VenuePay = s.mustOK("venue payment", s.do(b.token, "POST", fmt.Sprintf("/api/v1/projects/%d/venue-payments", w.Project), map[string]any{
		"type": "DP", "amount": 777777, "paymentDate": day(-2), "method": "Transfer", "referenceNumber": "VNREF-" + b.marker, "notes": "Rahasia " + b.marker,
	})).id("data", "id")
	w.Issue = s.mustOK("issue", s.do(b.token, "POST", fmt.Sprintf("/api/v1/projects/%d/issues", w.Project), map[string]any{
		"projectVendorId": w.Engagement, "title": "Kendala " + b.marker, "description": "Rahasia " + b.marker, "impact": "Medium",
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
		"address": "Alamat " + t.marker, "city": "Kota Bandung", "rentalPrice": 10000000, "charge": 500000, "capacity": 300,
		"facilities": "AC", "socialMedia": "@v", "notes": "Rahasia " + t.marker,
	}
}
