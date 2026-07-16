// Command seed resets elproof_db to a minimal clean-slate state: one
// Platform Console super admin account and one subscription plan — nothing
// else. Safe to re-run: it truncates and reseeds every table it touches
// rather than trying to diff/upsert a growing dataset.
//
// This is administrative tooling, not request-path code — unlike
// cmd/server, it reaches into modules' application/infrastructure packages
// directly for the handful of things no module's contracts expose yet
// rather than inventing contract methods no real caller needs.
package main

import (
	"context"
	"database/sql"
	"log"
	"strconv"

	billingapp "elproof/internal/modules/billing/application"
	billinginfra "elproof/internal/modules/billing/infrastructure"
	identityapp "elproof/internal/modules/identity/application"
	identitycontracts "elproof/internal/modules/identity/contracts"
	identityinfra "elproof/internal/modules/identity/infrastructure"
	platformdomain "elproof/internal/modules/platform/domain"
	platforminfra "elproof/internal/modules/platform/infrastructure"
	"elproof/internal/shared/config"
	"elproof/internal/shared/database"
)

func main() {
	cfg := config.Load()
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	truncateAll(db)

	planRepo := billinginfra.NewMySQLPlanRepository(db)
	planService := billingapp.NewPlanService(planRepo)

	credentialRepo := identityinfra.NewMySQLCredentialRepository(db)
	hasher := identityinfra.NewBcryptHasher()
	management := identityapp.NewManagementService(credentialRepo, hasher)
	identity := identitycontracts.New(management)

	adminRepo := platforminfra.NewMySQLPlatformAdminRepository(db)

	// --- The one subscription plan ---
	plan, err := planService.Create(ctx, billingapp.PlanInput{
		Name: "Paket 1 Tahun", DurationMonths: 12, Price: 2_000_000,
		Features: []string{
			"Akses penuh aplikasi ElProof untuk seluruh tim WO Console",
			"Kelola project, vendor, dan client tanpa batas",
			"Penyimpanan dokumen & bukti transaksi project",
			"Backup data otomatis setiap hari",
			"Dukungan teknis prioritas dari tim ElProof",
		},
	})
	must(err)
	log.Printf("seeded plan: %s (id=%d)", plan.Name, plan.ID)

	// --- The one Platform Console super admin account ---
	admin := &platformdomain.PlatformAdmin{
		Name: "Super Admin", Title: "Super Admin ElProof", Role: platformdomain.RoleSuperAdmin,
		Username: "superadmin", Email: "superadmin", Phone: "-", IsActive: true,
	}
	must(adminRepo.Create(ctx, admin))
	must(identity.CreateCredential(ctx, identitycontracts.CreateCredentialInput{
		PrincipalType: identitycontracts.PrincipalPlatformAdmin, PrincipalID: formatID(admin.ID),
		Username: admin.Username, Password: "superadmin", Role: string(admin.Role), DisplayName: admin.Name,
	}))
	log.Printf("seeded platform admin: %s (id=%d)", admin.Username, admin.ID)

	log.Println("done")
}

func truncateAll(db *sql.DB) {
	stmts := []string{
		"SET FOREIGN_KEY_CHECKS=0",
		"TRUNCATE TABLE refresh_tokens",
		"TRUNCATE TABLE credentials",
		"TRUNCATE TABLE subscription_transactions",
		"TRUNCATE TABLE plan_features",
		"TRUNCATE TABLE subscription_plans",
		"TRUNCATE TABLE staff_members",
		"TRUNCATE TABLE platform_admins",
		"TRUNCATE TABLE tenants",
		"TRUNCATE TABLE vendors",
		"TRUNCATE TABLE vendor_categories",
		"TRUNCATE TABLE activity_log",
		"TRUNCATE TABLE evidence",
		"TRUNCATE TABLE vendor_issues",
		"TRUNCATE TABLE vendor_payments",
		"TRUNCATE TABLE vendor_milestones",
		"TRUNCATE TABLE project_vendors",
		"TRUNCATE TABLE project_milestones",
		"TRUNCATE TABLE projects",
		"TRUNCATE TABLE clients",
		// Fase 9 (`payment` module + platform's own pending-charge index) —
		// none of these are business ledgers, safe to wipe along with
		// everything else. payment_gateway_config's single row is
		// re-inserted below to match the table's post-migration baseline;
		// payment_apps' internal App row re-bootstraps itself automatically
		// the next time cmd/server starts (EnsureInternalApp).
		"TRUNCATE TABLE pending_subscription_charges",
		"TRUNCATE TABLE payment_webhook_events",
		"TRUNCATE TABLE payment_charge_dispatch",
		"TRUNCATE TABLE payment_apps",
		"TRUNCATE TABLE payment_gateway_config",
		"INSERT INTO payment_gateway_config (id, active_provider, is_sandbox) VALUES (1, NULL, TRUE)",
		"SET FOREIGN_KEY_CHECKS=1",
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			log.Fatalf("truncate (%s): %v", stmt, err)
		}
	}
}

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
