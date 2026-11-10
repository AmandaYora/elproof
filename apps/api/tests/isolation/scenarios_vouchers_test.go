//go:build integration

package isolation

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"elproof/internal/modules/billing/application"
	"elproof/internal/modules/billing/domain"
	"elproof/internal/modules/billing/infrastructure"
	"elproof/internal/shared/apperror"
)

// T-20261110-17 (ADR-0041): vouchers belong to one App (optionally one plan),
// a WO tenant and an external App preview and use them, a code of another App
// looks unknown, the price never reaches Rp0, and quota / once-per-customer
// hold under concurrent charges.

func (s *stack) voucher(admin string, body map[string]any) reply {
	s.t.Helper()
	return s.do(admin, "POST", "/api/v1/vouchers", body)
}

func voucherBody(code, app string, kv ...any) map[string]any {
	m := map[string]any{"code": code, "appId": app, "discountType": "percent", "discountValue": 10, "validFrom": "", "validUntil": ""}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i].(string)] = kv[i+1]
	}
	return m
}

func reasonIs(t *testing.T, what string, r reply, reason string) {
	t.Helper()
	if r.status != http.StatusUnprocessableEntity || !strings.Contains(string(r.body), `"`+reason+`"`) {
		t.Errorf("%s: want 422 %s, got %s", what, reason, r)
	}
}

func (s *stack) planIDByName(name string) (int64, int64) {
	s.t.Helper()
	var id, price int64
	if err := s.db.QueryRow(`SELECT id, price FROM subscription_plans WHERE name = ?`, name).Scan(&id, &price); err != nil {
		s.t.Fatalf("plan %s: %v", name, err)
	}
	return id, price
}

func TestVouchers(t *testing.T) {
	s := startStack(t)
	admin := s.platformAdmin()
	jwsID, jwsTok := s.externalApp(admin, "JWS Voucher")
	elpPlan, elpPrice := s.planIDByName("Paket 1 Tahun")
	elp2 := s.createPlan(admin, elproofApp, "ElProof Bulanan", 250_000)
	jwsPlan := s.createPlan(admin, jwsID, "JWS Gold", 1_500_000)
	wo := s.signUp("voucher")
	wo2 := s.signUp("voucher2")

	check := func(tok string, plan int64, code string) reply {
		return s.do(tok, "POST", "/api/v1/subscriptions/voucher-check", map[string]any{"planId": plan, "voucherCode": code})
	}
	extCheck := func(plan int64, code string) reply {
		return s.do(jwsTok, "POST", "/api/v1/external/subscriptions/voucher-check", map[string]any{"planId": plan, "voucherCode": code})
	}

	t.Run("kriteria 1: form voucher", func(t *testing.T) {
		wantStatus(t, "percent 100", s.voucher(admin, voucherBody("PENUH", elproofApp, "discountValue", 100)), http.StatusUnprocessableEntity)
		r := s.voucher(admin, voucherBody("NOL", elproofApp, "planId", elp2, "discountType", "amount", "discountValue", 250_000))
		if r.status != http.StatusUnprocessableEntity || !strings.Contains(string(r.body), "discountValue") {
			t.Errorf("amount equal to the plan price must be refused on discountValue: %s", r)
		}
		r = s.voucher(admin, voucherBody("SALAHPAKET", elproofApp, "planId", jwsPlan))
		if r.status != http.StatusUnprocessableEntity || !strings.Contains(string(r.body), "planId") {
			t.Errorf("a plan of another App must be refused on planId: %s", r)
		}
		wantStatus(t, "unknown App", s.voucher(admin, voucherBody("TANPAAPP", "app_tidak_ada")), http.StatusUnprocessableEntity)
		r = s.mustOK("create HEMAT10", s.voucher(admin, voucherBody("hemat10", elproofApp, "maxDiscount", 100_000, "quota", 50)))
		if str(r, "data", "code") != "HEMAT10" || str(r, "data", "state") != "active" {
			t.Errorf("created voucher: %s", r)
		}
		wantStatus(t, "duplicate code in another case", s.voucher(admin, voucherBody("Hemat10", elproofApp)), http.StatusUnprocessableEntity)
		list := s.mustOK("list", s.do(admin, "GET", "/api/v1/vouchers?search=hemat&appId="+elproofApp, nil))
		if total, _ := list.dig("meta", "total").(float64); total != 1 {
			t.Errorf("list search: %s", list)
		}
		wantStatus(t, "WO staff cannot list vouchers", s.do(wo.token, "GET", "/api/v1/vouchers", nil), http.StatusForbidden)
		wantStatus(t, "App cannot list vouchers", s.do(jwsTok, "GET", "/api/v1/vouchers", nil), http.StatusForbidden)
	})

	t.Run("kriteria 2: tenant melihat harga dan alasan penolakan", func(t *testing.T) {
		r := s.mustOK("check HEMAT10", check(wo.token, elpPlan, " hemat10 "))
		disc := elpPrice / 10
		if disc > 100_000 {
			disc = 100_000
		}
		if r.dig("data", "originalAmount") != float64(elpPrice) || r.dig("data", "discountAmount") != float64(disc) || r.dig("data", "finalAmount") != float64(elpPrice-disc) {
			t.Errorf("quote: %s", r)
		}
		s.mustOK("expired", s.voucher(admin, voucherBody("LAMA", elproofApp, "validUntil", day(-2))))
		s.mustOK("future", s.voucher(admin, voucherBody("NANTI", elproofApp, "validFrom", day(3))))
		s.mustOK("one plan", s.voucher(admin, voucherBody("BULANAN", elproofApp, "planId", elp2)))
		off := s.mustOK("off", s.voucher(admin, voucherBody("MATI", elproofApp)))
		s.mustOK("deactivate", s.do(admin, "PATCH", fmt.Sprintf("/api/v1/vouchers/%d", off.id("data", "id")), map[string]any{"isActive": false}))
		s.mustOK("jws voucher", s.voucher(admin, voucherBody("JWS20", jwsID, "discountValue", 20, "oncePerCustomer", true)))
		reasonIs(t, "wrong code", check(wo.token, elpPlan, "TIDAKADA"), "voucher_not_found")
		reasonIs(t, "expired", check(wo.token, elpPlan, "LAMA"), "voucher_expired")
		reasonIs(t, "not started", check(wo.token, elpPlan, "NANTI"), "voucher_not_started")
		reasonIs(t, "inactive", check(wo.token, elpPlan, "MATI"), "voucher_inactive")
		reasonIs(t, "other plan", check(wo.token, elpPlan, "BULANAN"), "voucher_wrong_plan")
		reasonIs(t, "a JWS code from a WO", check(wo.token, elpPlan, "JWS20"), "voucher_not_found")
		wantStatus(t, "a WO checks a JWS plan", check(wo.token, jwsPlan, "HEMAT10"), http.StatusUnprocessableEntity)
		if r := check(wo.token, elpPlan, "LAMA"); !strings.Contains(string(r.body), "kedaluwarsa") {
			t.Errorf("message must be Indonesian and say why: %s", r)
		}
	})

	t.Run("kriteria 3: aplikasi lain lewat API", func(t *testing.T) {
		r := s.mustOK("ext check", extCheck(jwsPlan, "jws20"))
		if r.dig("data", "finalAmount") != float64(1_200_000) {
			t.Errorf("ext quote: %s", r)
		}
		r = extCheck(jwsPlan, "HEMAT10")
		if r.status != http.StatusUnprocessableEntity || !strings.Contains(string(r.body), `"reason":"voucher_not_found"`) {
			t.Errorf("ElProof code used by JWS: %s", r)
		}
		wantStatus(t, "JWS checks an ElProof plan", extCheck(elpPlan, "JWS20"), http.StatusUnprocessableEntity)
		wantStatus(t, "WO token on the external route", s.do(wo.token, "POST", "/api/v1/external/subscriptions/voucher-check", map[string]any{"planId": jwsPlan, "voucherCode": "JWS20"}), http.StatusForbidden)
	})

	t.Run("kriteria 5: kuota dan sekali per pelanggan", func(t *testing.T) {
		one := s.mustOK("quota 1", s.voucher(admin, voucherBody("SATU", elproofApp, "quota", 1))).id("data", "id")
		once := s.mustOK("once", s.voucher(admin, voucherBody("SEKALI", elproofApp, "oncePerCustomer", true))).id("data", "id")
		ins := func(voucherID int64, code, ref string, tenantID int64, status string) {
			if _, err := s.db.Exec(`INSERT INTO subscription_voucher_redemptions (voucher_id, voucher_code, order_ref, customer_type, customer_ref, plan_id, original_amount, discount_amount, final_amount, status)
				VALUES (?, ?, ?, 'tenant', ?, ?, 100, 10, 90, ?)`, voucherID, code, ref, fmt.Sprint(tenantID), elpPlan, status); err != nil {
				t.Fatal(err)
			}
		}
		ins(one, "SATU", "INV-Q1", wo2.tenant, "used")
		reasonIs(t, "quota used up", check(wo.token, elpPlan, "SATU"), "voucher_exhausted")
		ins(once, "SEKALI", "INV-O1", wo.tenant, "reserved")
		reasonIs(t, "once per customer, pending use", check(wo.token, elpPlan, "SEKALI"), "voucher_already_used")
		s.mustOK("other tenant may still use it", check(wo2.token, elpPlan, "SEKALI"))
		if _, err := s.db.Exec(`UPDATE subscription_voucher_redemptions SET status = 'released' WHERE order_ref = 'INV-O1'`); err != nil {
			t.Fatal(err)
		}
		s.mustOK("a released use no longer counts", check(wo.token, elpPlan, "SEKALI"))

		// Kriteria 5: the admin sees who used it, for what and when.
		ins(once, "SEKALI", "INV-O2", wo.tenant, "used")
		r := s.mustOK("redemptions", s.do(admin, "GET", fmt.Sprintf("/api/v1/vouchers/%d/redemptions", once), nil))
		body := string(r.body)
		for _, want := range []string{`"orderRef":"INV-O2"`, `"customerName":"WO ` + wo.marker + `"`, `"discountAmount":10`, `"status":"used"`, `"createdAt"`} {
			if !strings.Contains(body, want) {
				t.Errorf("redemptions lack %s: %s", want, r)
			}
		}
		v := s.mustOK("voucher counts", s.do(admin, "GET", fmt.Sprintf("/api/v1/vouchers/%d", once), nil))
		if v.dig("data", "usedCount") != float64(1) {
			t.Errorf("usedCount: %s", v)
		}
	})

	t.Run("tagihan yang gagal tidak memakan voucher", func(t *testing.T) {
		// No gateway is configured here, so both charges fail after the voucher was reserved.
		r := s.do(wo.token, "POST", "/api/v1/subscriptions/pay", map[string]any{"planId": elpPlan, "voucherCode": "HEMAT10"})
		if r.status < 400 {
			t.Fatalf("pay without a gateway should fail here: %s", r)
		}
		reasonIs(t, "pay with an expired code", s.do(wo.token, "POST", "/api/v1/subscriptions/pay", map[string]any{"planId": elpPlan, "voucherCode": "LAMA"}), "voucher_expired")
		r = s.do(jwsTok, "POST", "/api/v1/external/subscriptions/charges", map[string]any{"orderRef": "ORD-V-1", "planId": jwsPlan, "voucherCode": "JWS20"})
		if r.status < 400 {
			t.Fatalf("charge without a gateway should fail here: %s", r)
		}
		if n := s.count(`SELECT COUNT(*) FROM subscription_voucher_redemptions WHERE voucher_code IN ('HEMAT10', 'JWS20')`); n != 0 {
			t.Errorf("a failed charge left %d voucher uses", n)
		}
		if n := s.count(`SELECT COUNT(*) FROM subscription_app_charges WHERE order_ref = 'ORD-V-1'`); n != 0 {
			t.Errorf("a failed charge left its intent row")
		}
	})

	t.Run("If-Match basi", func(t *testing.T) {
		r := s.mustOK("create", s.voucher(admin, voucherBody("VERSI", elproofApp)))
		id, version := r.id("data", "id"), str(r, "data", "updatedAt")
		path := fmt.Sprintf("/api/v1/vouchers/%d", id)
		s.mustOK("edit", s.doWith(admin, "PATCH", path, voucherBody("VERSI", elproofApp, "discountValue", 15), map[string]string{"If-Match": `"` + version + `"`}))
		r = s.doWith(admin, "PATCH", path, voucherBody("VERSI", elproofApp, "discountValue", 20), map[string]string{"If-Match": `"` + version + `"`})
		if r.status != http.StatusConflict || r.dig("data", "discountValue") != float64(15) {
			t.Errorf("stale edit: %s", r)
		}
	})

	// Concurrency, on the real repository: the row lock makes quota and
	// once-per-customer exact under simultaneous charges.
	t.Run("balapan kuota dan sekali per pelanggan", func(t *testing.T) {
		ctx := context.Background()
		repo := infrastructure.NewMySQLVoucherRepository(s.db)
		svc := application.NewVoucherService(repo, infrastructure.NewMySQLPlanRepository(s.db))
		plan, err := infrastructure.NewMySQLPlanRepository(s.db).FindByID(ctx, elpPlan)
		if err != nil || plan == nil {
			t.Fatalf("plan: %v", err)
		}
		s.mustOK("race quota", s.voucher(admin, voucherBody("BALAP", elproofApp, "quota", 3)))
		s.mustOK("race once", s.voucher(admin, voucherBody("BALAP1", elproofApp, "oncePerCustomer", true)))

		race := func(code string, customer func(i int) domain.Customer) (ok []string, reasons map[string]int) {
			var mu sync.Mutex
			var wg sync.WaitGroup
			reasons = map[string]int{}
			for i := 0; i < 20; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					ref := fmt.Sprintf("RACE-%s-%d", code, i)
					_, err := svc.Reserve(ctx, code, plan, customer(i), ref)
					mu.Lock()
					defer mu.Unlock()
					if err == nil {
						ok = append(ok, ref)
						return
					}
					ae, _ := apperror.As(err)
					if ae == nil {
						reasons["error: "+err.Error()]++
						return
					}
					reasons[ae.Code]++
				}(i)
			}
			wg.Wait()
			return ok, reasons
		}
		ok, reasons := race("BALAP", func(i int) domain.Customer {
			return domain.Customer{Type: domain.CustomerTenant, Ref: fmt.Sprint(1000 + i)}
		})
		if len(ok) != 3 || reasons[application.VoucherExhausted] != 17 {
			t.Fatalf("quota 3 under 20 concurrent charges: ok=%d reasons=%v", len(ok), reasons)
		}
		ok1, reasons1 := race("BALAP1", func(int) domain.Customer { return domain.Customer{Type: domain.CustomerApp, Ref: jwsID} })
		if len(ok1) != 1 || reasons1[application.VoucherAlreadyUsed] != 19 {
			t.Fatalf("once per customer under 20 concurrent charges: ok=%d reasons=%v", len(ok1), reasons1)
		}

		// A cancelled charge gives its unit back; a paid one keeps it.
		if err := svc.Settle(ctx, ok[0], false); err != nil {
			t.Fatal(err)
		}
		if err := svc.Settle(ctx, ok[1], true); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Reserve(ctx, "BALAP", plan, domain.Customer{Type: domain.CustomerTenant, Ref: "2000"}, "RACE-AFTER"); err != nil {
			t.Fatalf("a released unit must be usable again: %v", err)
		}
		if err := svc.Settle(ctx, ok[1], false); err != nil { // already used: left alone
			t.Fatal(err)
		}
		if n := s.count(`SELECT COUNT(*) FROM subscription_voucher_redemptions WHERE order_ref = ? AND status = 'used'`, ok[1]); n != 1 {
			t.Error("settling a used reservation again must not release it")
		}

		// The stale sweep releases an old reservation without a charge, and keeps one whose charge is pending.
		old := time.Now().Add(-2 * time.Hour)
		if _, err := s.db.Exec(`UPDATE subscription_voucher_redemptions SET created_at = ? WHERE order_ref IN (?, ?)`, old, ok[2], "RACE-AFTER"); err != nil {
			t.Fatal(err)
		}
		if _, err := s.db.Exec(`INSERT INTO subscription_transactions (tenant_id, type, amount, original_amount, payment_method, payment_reference, status)
			VALUES (?, 'new', 90, 100, 'QRIS', 'RACE-AFTER', 'pending')`, wo.tenant); err != nil {
			t.Fatal(err)
		}
		n, err := svc.ReleaseStale(ctx)
		if err != nil || n != 1 {
			t.Fatalf("ReleaseStale released %d (%v), want 1", n, err)
		}
		if s.count(`SELECT COUNT(*) FROM subscription_voucher_redemptions WHERE order_ref = ? AND status = 'released'`, ok[2]) != 1 ||
			s.count(`SELECT COUNT(*) FROM subscription_voucher_redemptions WHERE order_ref = 'RACE-AFTER' AND status = 'reserved'`) != 1 {
			t.Error("the sweep released the wrong reservation")
		}
	})
}
