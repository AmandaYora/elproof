//go:build integration

package isolation

import (
	"fmt"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"testing"
)

// T-20261110-16 (ADR-0040): a subscription plan belongs to exactly one App.
// ElProof's own catalog for WO tenants is the plans of its internal App
// "platform-billing"; an external App sees and buys only its own plans; a WO
// tenant is never subscribed to another App's plan, by itself or by an admin.

const elproofApp = "platform-billing"

// externalApp registers an external App in the Console and signs it in.
func (s *stack) externalApp(admin, name string) (appID, token string) {
	s.t.Helper()
	created := s.mustOK("register app "+name, s.do(admin, "POST", "/api/v1/payment/apps", map[string]any{"name": name, "callbackUrl": "https://example.com/hook"}))
	appID, secret := str(created, "data", "appId"), str(created, "data", "secret")
	tok := s.mustOK("app token "+name, s.do("", "POST", "/api/v1/auth/app/token", map[string]any{"appId": appID, "secret": secret}))
	return appID, str(tok, "data", "accessToken")
}

func (s *stack) createPlan(admin, appID, name string, price int64) int64 {
	s.t.Helper()
	r := s.mustOK("create plan "+name, s.do(admin, "POST", "/api/v1/plans", map[string]any{
		"appId": appID, "name": name, "durationMonths": 12, "price": price, "features": []string{"Fitur " + name},
	}))
	return r.id("data", "id")
}

// planNames lists the plan names in a response's data array, sorted.
func planNames(r reply) []string {
	items, _ := r.dig("data").([]any)
	names := make([]string, 0, len(items))
	for _, it := range items {
		if m, ok := it.(map[string]any); ok {
			names = append(names, fmt.Sprint(m["name"]))
		}
	}
	sort.Strings(names)
	return names
}

func wantNames(t *testing.T, what string, r reply, want ...string) {
	t.Helper()
	if r.status != http.StatusOK {
		t.Errorf("%s: %s", what, r)
		return
	}
	sort.Strings(want)
	if got := planNames(r); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("%s: plans %v, want %v", what, got, want)
	}
}

func TestPlansPerApp(t *testing.T) {
	s := startStack(t)
	admin := s.platformAdmin() // reseeds: one ElProof plan "Paket 1 Tahun"
	jwsID, jwsTok := s.externalApp(admin, "JWS Uji")
	kasirID, kasirTok := s.externalApp(admin, "Kasir Uji")

	elproof2 := s.createPlan(admin, elproofApp, "ElProof Bulanan", 250_000)
	jwsPlan := s.createPlan(admin, jwsID, "JWS Gold", 2_000_000)
	kasirPlan := s.createPlan(admin, kasirID, "Kasir Basic", 99_000)

	// Kriteria 1: a plan must name exactly one known App.
	t.Run("form wajib satu aplikasi", func(t *testing.T) {
		base := map[string]any{"name": "Tanpa App", "durationMonths": 1, "price": 1, "features": []string{"a"}}
		wantStatus(t, "create without appId", s.do(admin, "POST", "/api/v1/plans", base), http.StatusUnprocessableEntity)
		withApp := func(app string) map[string]any {
			m := map[string]any{"appId": app}
			for k, v := range base {
				m[k] = v
			}
			return m
		}
		r := s.do(admin, "POST", "/api/v1/plans", withApp("app_tidak_ada"))
		wantStatus(t, "create with unknown appId", r, http.StatusUnprocessableEntity)
		if !strings.Contains(string(r.body), "appId") {
			t.Errorf("422 must point at appId: %s", r)
		}
		wantStatus(t, "create with ElProof in another case", s.do(admin, "POST", "/api/v1/plans", withApp("Platform-Billing")), http.StatusUnprocessableEntity)
		wantStatus(t, "create with an App id in another case", s.do(admin, "POST", "/api/v1/plans", withApp(strings.ToUpper(jwsID))), http.StatusUnprocessableEntity)
		wantStatus(t, "edit to unknown appId", s.do(admin, "PATCH", fmt.Sprintf("/api/v1/plans/%d", kasirPlan), withApp("app_tidak_ada")), http.StatusUnprocessableEntity)
		if n := s.count(`SELECT COUNT(*) FROM subscription_plans WHERE name = 'Tanpa App'`); n != 0 {
			t.Errorf("a rejected plan was stored (%d rows)", n)
		}
		if n := s.count(`SELECT COUNT(*) FROM subscription_plans WHERE id = ? AND app_id = ?`, kasirPlan, kasirID); n != 1 {
			t.Error("a rejected edit changed the plan's App")
		}
	})

	// Kriteria 2/3: the Console reads every App's plans, or one App's.
	t.Run("konsol saring per aplikasi", func(t *testing.T) {
		wantNames(t, "admin, all", s.do(admin, "GET", "/api/v1/plans?all=true", nil), "Paket 1 Tahun", "ElProof Bulanan", "JWS Gold", "Kasir Basic")
		wantNames(t, "admin, appId=jws", s.do(admin, "GET", "/api/v1/plans?all=true&appId="+jwsID, nil), "JWS Gold")
		wantNames(t, "admin, appId=elproof", s.do(admin, "GET", "/api/v1/plans?all=true&appId="+elproofApp, nil), "Paket 1 Tahun", "ElProof Bulanan")
		page := s.do(admin, "GET", "/api/v1/plans?appId="+kasirID, nil)
		wantNames(t, "admin, paginated appId=kasir", page, "Kasir Basic")
		if total, _ := page.dig("meta", "total").(float64); total != 1 {
			t.Errorf("paginated total = %v, want 1", page.dig("meta", "total"))
		}
	})

	// Kriteria 4: who sees which catalog.
	wo := s.signUp("paket")
	t.Run("tenant WO hanya katalog ElProof", func(t *testing.T) {
		wantNames(t, "tenant, all", s.do(wo.token, "GET", "/api/v1/plans?all=true", nil), "Paket 1 Tahun", "ElProof Bulanan")
		wantNames(t, "tenant, asks for jws", s.do(wo.token, "GET", "/api/v1/plans?all=true&appId="+jwsID, nil), "Paket 1 Tahun", "ElProof Bulanan")
		wantNames(t, "tenant, paginated", s.do(wo.token, "GET", "/api/v1/plans?appId="+jwsID, nil), "Paket 1 Tahun", "ElProof Bulanan")
		wantNames(t, "public homepage", s.do("", "GET", "/api/v1/public/plans", nil), "Paket 1 Tahun", "ElProof Bulanan")
	})
	t.Run("aplikasi hanya paketnya sendiri", func(t *testing.T) {
		wantNames(t, "jws catalog", s.do(jwsTok, "GET", "/api/v1/external/subscription-plans", nil), "JWS Gold")
		wantNames(t, "kasir catalog", s.do(kasirTok, "GET", "/api/v1/external/subscription-plans", nil), "Kasir Basic")
		for _, plan := range []int64{kasirPlan, elproof2} {
			r := s.do(jwsTok, "POST", "/api/v1/external/subscriptions/charges", map[string]any{"orderRef": fmt.Sprintf("ORD-X-%d", plan), "planId": plan})
			wantStatus(t, fmt.Sprintf("jws buys plan %d of another App", plan), r, http.StatusUnprocessableEntity)
		}
		if n := s.count(`SELECT COUNT(*) FROM subscription_app_charges WHERE app_id = ?`, jwsID); n != 0 {
			t.Errorf("a refused purchase left %d ledger rows", n)
		}
	})
	t.Run("langganan tenant menolak paket aplikasi lain", func(t *testing.T) {
		wantStatus(t, "tenant pays a JWS plan", s.do(wo.token, "POST", "/api/v1/subscriptions/pay", map[string]any{"planId": jwsPlan}), http.StatusUnprocessableEntity)
		activate := func(plan int64) reply {
			return s.do(admin, "POST", fmt.Sprintf("/api/v1/tenants/%d/activate-subscription", wo.tenant), map[string]any{
				"planId": plan, "reason": "internal", "proofReference": "UJI-PAKET-1", "note": "uji paket per aplikasi",
			})
		}
		r := activate(jwsPlan)
		wantStatus(t, "admin activates a JWS plan for a WO", r, http.StatusUnprocessableEntity)
		if !strings.Contains(string(r.body), "planId") {
			t.Errorf("422 must point at planId: %s", r)
		}
		if n := s.count(`SELECT COUNT(*) FROM tenants WHERE id = ? AND plan_id = ?`, wo.tenant, jwsPlan); n != 0 {
			t.Error("the WO tenant was put on a JWS plan")
		}
		wantStatus(t, "admin activates an ElProof plan", activate(elproof2), http.StatusOK)
		if n := s.count(`SELECT COUNT(*) FROM tenants WHERE id = ? AND plan_id = ?`, wo.tenant, elproof2); n != 1 {
			t.Error("activation with an ElProof plan did not apply")
		}
	})
	t.Run("rute pemetaan lama hilang", func(t *testing.T) {
		if r := s.do(admin, "GET", "/api/v1/app-plans?appId="+jwsID, nil); r.status != http.StatusNotFound {
			t.Errorf("/app-plans still answers: %s", r)
		}
	})
}

// TestPlansPerAppMigration walks 000073 down and up again: the old mapping
// comes back from app_id, and a mapping turns into app_id.
func TestPlansPerAppMigration(t *testing.T) {
	s := startStack(t)
	run := func(arg string) {
		cmd := exec.Command(s.bin, "migrate", arg)
		cmd.Dir, cmd.Env = s.dir, s.env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("migrate %s: %v\n%s", arg, err, out)
		}
	}
	version := func() int {
		var v int
		if err := s.db.QueryRow(`SELECT version FROM schema_migrations`).Scan(&v); err != nil {
			t.Fatalf("schema_migrations: %v", err)
		}
		return v
	}
	columnExists := func() bool {
		return s.count(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'subscription_plans' AND column_name = 'app_id'`) == 1
	}
	tableExists := func() bool {
		return s.count(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'subscription_plan_apps'`) == 1
	}
	sqlExec := func(q string, args ...any) {
		if _, err := s.db.Exec(q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}

	if !columnExists() || tableExists() {
		t.Fatalf("after up: app_id column=%v, subscription_plan_apps table=%v", columnExists(), tableExists())
	}
	sqlExec(`INSERT INTO subscription_plans (app_id, name, duration_months, price) VALUES ('platform-billing', 'Mig ElProof', 1, 1), ('app_mig_a', 'Mig A', 1, 2)`)

	for version() > 72 {
		run("down")
	}
	if columnExists() || !tableExists() {
		t.Fatalf("after down: app_id column=%v, subscription_plan_apps table=%v", columnExists(), tableExists())
	}
	if n := s.count(`SELECT COUNT(*) FROM subscription_plan_apps spa JOIN subscription_plans p ON p.id = spa.plan_id WHERE p.name = 'Mig A' AND spa.app_id = 'app_mig_a'`); n != 1 {
		t.Errorf("down: the App plan must come back as one mapping row, found %d", n)
	}
	if n := s.count(`SELECT COUNT(*) FROM subscription_plan_apps spa JOIN subscription_plans p ON p.id = spa.plan_id WHERE p.name = 'Mig ElProof'`); n != 0 {
		t.Errorf("down: an ElProof plan must be unmapped (public), found %d mapping rows", n)
	}

	// Old-model data: an unmapped plan, a plan mapped to one App, and one
	// mapped to two Apps (development data) — each must land on one App.
	sqlExec(`INSERT INTO subscription_plans (name, duration_months, price) VALUES ('Mig Publik', 1, 3), ('Mig Dua', 1, 4)`)
	sqlExec(`INSERT INTO subscription_plan_apps (plan_id, app_id) SELECT id, 'app_mig_z' FROM subscription_plans WHERE name = 'Mig Dua'`)
	sqlExec(`INSERT INTO subscription_plan_apps (plan_id, app_id) SELECT id, 'app_mig_b' FROM subscription_plans WHERE name = 'Mig Dua'`)

	run("up")
	if !columnExists() || tableExists() {
		t.Fatalf("after up again: app_id column=%v, subscription_plan_apps table=%v", columnExists(), tableExists())
	}
	for name, app := range map[string]string{"Mig ElProof": elproofApp, "Mig A": "app_mig_a", "Mig Publik": elproofApp, "Mig Dua": "app_mig_b"} {
		if n := s.count(`SELECT COUNT(*) FROM subscription_plans WHERE name = ? AND app_id = ?`, name, app); n != 1 {
			t.Errorf("up: plan %q must belong to %q", name, app)
		}
	}
	// Exact comparison, like Go: a case variant of an App id is another value.
	if n := s.count(`SELECT COUNT(*) FROM subscription_plans WHERE app_id = ?`, "PLATFORM-BILLING"); n != 0 {
		t.Errorf("app_id must compare case-sensitively, %d rows matched PLATFORM-BILLING", n)
	}
	// No default: an INSERT that forgets the App is refused.
	if _, err := s.db.Exec(`INSERT INTO subscription_plans (name, duration_months, price) VALUES ('Tanpa App', 1, 1)`); err == nil {
		t.Error("a plan without app_id must be refused by the schema")
	}
}
