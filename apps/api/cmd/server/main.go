package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"elproof/internal/adminseed"
	"elproof/internal/loginslides"
	"elproof/internal/migrator"
	"elproof/internal/modules/access"
	accesscontracts "elproof/internal/modules/access/contracts"
	"elproof/internal/modules/audit"
	"elproof/internal/modules/billing"
	"elproof/internal/modules/clients"
	"elproof/internal/modules/identity"
	"elproof/internal/modules/payment"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/modules/platform"
	"elproof/internal/modules/projects"
	"elproof/internal/modules/staff"
	staffcontracts "elproof/internal/modules/staff/contracts"
	"elproof/internal/modules/vendors"
	"elproof/internal/shared/config"
	"elproof/internal/shared/database"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
	"elproof/internal/shared/storage"
)

// staffNameResolver adapts staffcontracts.Contracts to
// projects/application.StaffNameResolver's narrow, primitive-typed
// interface. Kept here (not in either module) to avoid an import cycle:
// staff/application already imports projects/contracts (for its own
// hard-delete impact-lookup bridge), so projects/application importing
// staff/contracts back would close the loop
// (projects/application -> staff/contracts -> staff/application ->
// projects/contracts -> projects/application) — see
// application.StaffNameResolver's own doc comment.
type staffNameResolver struct {
	contracts staffcontracts.Contracts
}

func (r staffNameResolver) GetName(ctx context.Context, tenantID, staffID int64) (string, bool, error) {
	summary, err := r.contracts.GetSummary(ctx, tenantID, staffID)
	if err != nil {
		return "", false, err
	}
	if summary == nil {
		return "", false, nil
	}
	return summary.Name, true, nil
}

func (r staffNameResolver) GetPhone(ctx context.Context, tenantID, staffID int64) (string, bool, error) {
	summary, err := r.contracts.GetSummary(ctx, tenantID, staffID)
	if err != nil {
		return "", false, err
	}
	if summary == nil {
		return "", false, nil
	}
	return summary.Phone, true, nil
}

func (r staffNameResolver) StaffNames(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error) {
	return r.contracts.StaffNames(ctx, tenantID, ids)
}

// staffRoles / clientRoles adapt the access module's contract to the narrow
// role ports staff and clients declare (they never import access).
type staffRoles struct{ access accesscontracts.Contracts }

func (r staffRoles) AssignableStaffRole(ctx context.Context, tenantID, roleID int64) (staff.RoleAccess, error) {
	ref, err := r.access.RoleForAssignment(ctx, tenantID, roleID, accesscontracts.KindStaff)
	return staff.RoleAccess{Name: ref.Name, Permissions: ref.Permissions, AllProjects: ref.AllProjects}, err
}

func (r staffRoles) RoleNames(ctx context.Context, tenantID int64) (map[int64]string, error) {
	return r.access.RoleNames(ctx, tenantID)
}

type clientRoles struct{ access accesscontracts.Contracts }

func (r clientRoles) DefaultClientRole(ctx context.Context, tenantID int64, family bool) (int64, error) {
	key := accesscontracts.TemplateBride
	if family {
		key = accesscontracts.TemplateFamily
	}
	ref, err := r.access.TemplateRole(ctx, tenantID, key)
	return ref.ID, err
}

func (r clientRoles) AssignableClientRole(ctx context.Context, tenantID, roleID int64) (string, error) {
	ref, err := r.access.RoleForAssignment(ctx, tenantID, roleID, accesscontracts.KindClient)
	return ref.Name, err
}

func (r clientRoles) RoleNames(ctx context.Context, tenantID int64) (map[int64]string, error) {
	return r.access.RoleNames(ctx, tenantID)
}

// main dispatches on an optional subcommand so the one compiled binary this
// project ships as its deploy image (see infra/docker/Dockerfile) is
// everything a production host needs — no separate migrate CLI, seed
// binary, or curl/wget in the container for Docker's HEALTHCHECK. With no
// argument it serves, exactly as before this dispatch existed.
//
//	./api                       serve the API (default, unchanged behavior)
//	./api migrate up|down        apply/roll back one embedded SQL migration step
//	./api migrate force <version> clear a "dirty" migration state without running SQL
//	./api seed                   reset to the minimal clean-slate dataset
//	./api healthcheck             exit 0/1 for Docker's HEALTHCHECK (self GET /api/v1/health)
//	./api upload-login-slides <dir>  compress+upload the login page's fixed
//	                                  marketing photos to object storage
func main() {
	cfg := config.Load()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			runMigrate(cfg, os.Args[2:])
			return
		case "seed":
			runSeed(cfg)
			return
		case "healthcheck":
			runHealthcheck(cfg)
			return
		case "upload-login-slides":
			runUploadLoginSlides(cfg, os.Args[2:])
			return
		}
	}

	serve(cfg)
}

func runMigrate(cfg config.Config, args []string) {
	if len(args) == 2 && args[0] == "force" {
		version, convErr := strconv.Atoi(args[1])
		if convErr != nil {
			log.Fatalf("usage: api migrate force <version>")
		}
		if err := migrator.Force(cfg.DatabaseURL, version); err != nil {
			log.Fatalf("migrate force %d: %v", version, err)
		}
		log.Printf("migrate force %d: done", version)
		return
	}

	if len(args) != 1 || (args[0] != "up" && args[0] != "down") {
		log.Fatal("usage: api migrate up|down|force <version>")
	}

	var err error
	if args[0] == "up" {
		err = migrator.Up(cfg.DatabaseURL)
	} else {
		err = migrator.Down(cfg.DatabaseURL)
	}
	if err != nil {
		log.Fatalf("migrate %s: %v", args[0], err)
	}
	log.Printf("migrate %s: done", args[0])
}

func runSeed(cfg config.Config) {
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := adminseed.Run(context.Background(), db); err != nil {
		log.Fatal(err)
	}
}

// runUploadLoginSlides is a one-off, re-runnable operator command (not part
// of serve()'s request path) — see internal/loginslides for what it does.
func runUploadLoginSlides(cfg config.Config, args []string) {
	if len(args) != 1 {
		log.Fatal("usage: api upload-login-slides <source-dir>")
	}

	storageClient, err := storage.New(storage.Config{
		Endpoint: cfg.S3Endpoint, Bucket: cfg.S3Bucket,
		AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey, UseSSL: cfg.S3UseSSL,
	})
	if err != nil {
		log.Fatalf("failed to create object storage client: %v", err)
	}

	if err := loginslides.Run(context.Background(), storageClient, args[0]); err != nil {
		log.Fatalf("upload-login-slides: %v", err)
	}
	log.Println("upload-login-slides: done")
}

func runHealthcheck(cfg config.Config) {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%s/api/v1/health", cfg.AppPort))
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
	resp.Body.Close()
	os.Exit(0)
}

func serve(cfg config.Config) {
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		response.OK(w, "ok", nil)
	})

	googleJWKS := ""
	if cfg.GoogleJWKSURL != "" {
		// Only end-to-end tests point this elsewhere (a local fake Google);
		// a non-local URL is refused so a misconfigured server can never
		// trust someone else's signing keys.
		if cfg.AppEnv == "production" || !isLocalURL(cfg.GoogleJWKSURL) {
			log.Fatalf("GOOGLE_JWKS_URL is only for local tests; remove it from this environment")
		}
		googleJWKS = cfg.GoogleJWKSURL
	}
	identityModule := identity.NewModule(db, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL,
		identity.GoogleConfig{ClientID: cfg.GoogleClientID, JWKSURL: googleJWKS})
	identityModule.RegisterRoutes(mux)

	// Public, non-secret settings the web app needs before sign-in.
	mux.HandleFunc("/api/v1/public/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.Error(w, http.StatusMethodNotAllowed, "Metode tidak diizinkan", nil)
			return
		}
		response.OK(w, "ok", map[string]string{"googleClientId": cfg.GoogleClientID})
	})

	// Every authenticated route: verify the JWT, hold `client` principals to
	// the Client Portal's own allowlist (client_scope.go), then enforce the
	// tenant's subscription state (tenant_scope.go). The gate's resolver is
	// the platform module, bound below once it exists.
	requireAuth := middleware.RequireAuth(cfg.JWTSecret)
	// Tokens stop working as soon as their account is deactivated, demoted
	// or loses two-factor, not only when they expire.
	liveSession := &middleware.RequireLiveSession{}
	liveSession.SetValidator(identityModule.SessionValidator())
	// A support token also needs its session, the Owner's consent and the
	// admin to still be valid (bound once the platform module exists).
	liveSupport := &middleware.RequireLiveSupportSession{}
	// Staff and client permissions come from their workspace role (ADR-0026);
	// resolved once per request, the resolver is the access module below.
	loadGrants := &middleware.LoadGrants{}
	restrictClients := middleware.RestrictPrincipal("client", clientPortalAllowed)
	restrictAdmins := middleware.RestrictPrincipalBy("platform_admin", platformAdminPolicy)
	tenantGate := middleware.NewTenantAccessGate(staffAllowedWhenRestricted)
	auditModule := audit.NewModule(db)
	// Every state-changing request by ElProof's own team is recorded,
	// including refused attempts (audit sits before the role check).
	auditAdmins := middleware.AuditTrail(auditModule.Recorder(), "platform_admin")
	authed := func(next http.Handler) http.Handler {
		return requireAuth(liveSession.Middleware(liveSupport.Middleware(middleware.RestrictImpersonation(auditAdmins(restrictAdmins(restrictClients(tenantGate.Middleware(loadGrants.Middleware(next)))))))))
	}

	mux.Handle("/api/v1/auth/me", authed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := middleware.FromContext(r.Context())
		response.OK(w, "ok", claims)
	})))

	storageClient, err := storage.New(storage.Config{
		Endpoint: cfg.S3Endpoint, Bucket: cfg.S3Bucket,
		AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey, UseSSL: cfg.S3UseSSL,
		Region: cfg.S3Region, Presign: cfg.S3Presign,
	})
	if err != nil {
		log.Fatalf("failed to create object storage client: %v", err)
	}

	// payment depends on identity (Fase 10, one-way, same shape as
	// vendors -> projects — see knowledge/MODULE_MAP.md) to mint bearer
	// tokens for external Apps; it still depends on no other module for its
	// own gateway/App-registry logic, so any App internal (e.g. platform)
	// can receive its Client at their own construction time.
	paymentModule, err := payment.NewModule(db, cfg.PaymentEncryptionKey, identityModule.Contracts(), cfg.AppTokenTTL)
	if err != nil {
		log.Fatalf("failed to init payment module: %v", err)
	}

	accessModule := access.NewModule(db)
	accessModule.SetMemberCounter(identityModule.Contracts())
	loadGrants.SetResolver(accessModule)

	staffModule := staff.NewModule(db, identityModule.Contracts())
	staffModule.SetRoles(staffRoles{accessModule.Contracts()})
	// billing now takes two dependencies from payment (A5, same direct-
	// contracts-injection pattern as platform receiving paymentModule.Client()
	// below): AppRegistry for its own external/console App-token endpoints,
	// Client to create subscription charges. Urutan konstruksi tidak berubah
	// (A6) — payment is still built before billing.
	billingModule := billing.NewModule(db, paymentModule.AppRegistry(), paymentModule.Client())
	platformModule := platform.NewModule(db, staffModule.Contracts(), identityModule.Contracts(), billingModule.Contracts(), paymentModule.Client(), storageClient)
	liveSupport.SetValidator(platformModule.SupportSessionValidator())
	// Two-phase wiring (identity is built first): a tenant frozen by the
	// platform team can neither log in nor refresh, and every request is
	// checked against the tenant's subscription lifecycle.
	identityModule.SetTenantGuard(platformModule)
	identityModule.SetTenantNames(platformModule)
	tenantGate.SetResolver(platformModule)
	// Pre-auth, Host-header-resolved tenant branding (ADR-0015) — registered
	// unwrapped, same style as identityModule.RegisterRoutes(mux) above.
	platformModule.SetAuditRecorder(auditModule.Recorder())
	platformModule.RegisterPublicRoutes(mux)
	if err := platformModule.LoadSubscriptionPolicy(context.Background()); err != nil {
		log.Printf("subscription policy: using defaults (%v)", err)
	}
	// projects is built before vendors — vendors' "Lihat Project" resolves a
	// vendor's cross-project engagement history through projects.Contracts()
	// (project_vendors is owned by projects, not vendors).
	projectsModule := projects.NewModule(db, storageClient, staffNameResolver{contracts: staffModule.Contracts()}, cfg.JWTSecret)
	// Two-phase wiring: platform needs projects' contract (seed a newly
	// registered tenant's Timeline Default template, PLAN.md) but
	// platformModule is built above, before projectsModule exists — same
	// bridging pattern as SetVendors below.
	platformModule.SetProjects(projectsModule.Contracts())
	// Two-phase wiring: staff needs projects' contract (hard-delete "impact"
	// lookup, PLAN.md) but staffModule is built above, before projectsModule
	// exists — same bridging pattern as SetProjects above.
	staffModule.SetProjectReferenceLookup(projectsModule.Contracts())
	vendorsModule := vendors.NewModule(db, projectsModule.Contracts(), storageClient)
	// Two-phase wiring: platform needs vendors' contract (seed default vendor
	// categories on tenant registration) but platformModule is built above,
	// before vendorsModule exists — same bridging pattern as the
	// SetClientAccessResolver call below.
	platformModule.SetVendors(vendorsModule.Contracts())
	// Same bridge, the new reciprocal direction (ADR-0016): projects needs to
	// resolve a project's attached venue_id into venue details for its own
	// Project Detail tab and Client Portal's Venue tab, but projectsModule is
	// built above, before vendorsModule exists.
	projectsModule.SetVenueResolver(vendorsModule.Contracts())
	projectsModule.SetVendorDirectory(vendorsModule.Contracts())
	// The Client Portal home shows the WO's WhatsApp number and bank account,
	// which live in platform's business profile.
	projectsModule.SetTenantContact(platformModule)
	clientsModule := clients.NewModule(db, projectsModule.Contracts(), identityModule.Contracts())
	// Two-phase wiring: projects needs clients' contract (Fase 6 client-portal
	// scoping) but clients.NewModule already needs projects.Contracts() to
	// exist first, so this can't be a constructor argument either direction.
	projectsModule.SetClientAccessResolver(clientsModule.Contracts())
	// Same bridge, second interface (ADR-0013's hard-delete client cleanup) —
	// clientsModule.Contracts() satisfies both ClientAccessResolver and
	// ClientCleaner, so this is the same object as the call above.
	projectsModule.SetClientCleaner(clientsModule.Contracts())
	// Free-trial caps (PLAN.md decision K6): staff, projects and clients each
	// count their own rows; platform says whether the tenant is on trial
	// and what the cap is. The onboarding checklist reads client usage.
	staffModule.SetTrialQuota(platformModule)
	projectsModule.SetTrialQuota(platformModule)
	clientsModule.SetTrialQuota(platformModule)
	platformModule.SetClients(clientsModule.Contracts())
	platformModule.SetAccess(accessModule.Contracts())
	clientsModule.SetRoles(clientRoles{accessModule.Contracts()})
	// Same bridging pattern: payment can't import platform (or any other App
	// internal) to know its webhook consumer, so main.go registers it here,
	// after both modules are built — see payment.module.go's Dispatcher doc.
	paymentModule.Dispatcher().RegisterConsumer(paymentcontracts.InternalAppBilling, platformModule)
	// Started only after every App internal's consumer is registered above —
	// a sweep tick may need to dispatch to one right away. Runs for the
	// lifetime of the process; no separate shutdown signal exists anywhere
	// else in this server either (see server.ListenAndServe() below), so this
	// goroutine simply ends when the process does.
	paymentModule.StartReconciler(context.Background(), cfg.PaymentReconcileInterval)
	// Sejajar dengan reconciler payment di atas, bukan menggantikannya (A16) —
	// billing punya reconciler sendiri untuk subscription_app_charges, yang
	// memanggil gateway secara independen (A32).
	billingModule.SetSubscriptionChargeMaxAge(cfg.SubscriptionChargeMaxAge)
	billingModule.StartSubscriptionReconciler(context.Background(), cfg.SubscriptionReconcileInterval)
	// Keeps tenants.subscription_status (list filters, dashboards) in step
	// with the date-derived lifecycle; access never depends on it.
	platformModule.StartSubscriptionStatusSweeper(context.Background(), time.Hour)

	identityModule.RegisterAuthedRoutes(mux, authed)
	// A WO's changes to its own roles, and to its own business profile (including
	// the bank account clients are told to pay into), are audited like the
	// platform team's: who, when, from where. Bodies are never logged.
	auditStaff := middleware.AuditTrail(auditModule.Recorder(), "staff")
	accessModule.RegisterRoutes(mux, func(h http.Handler) http.Handler { return authed(auditStaff(h)) })
	auditModule.RegisterRoutes(mux, authed)
	paymentModule.RegisterRoutes(mux, authed)
	billingModule.RegisterRoutes(mux, authed)
	platformModule.RegisterRoutes(mux, func(h http.Handler) http.Handler { return authed(auditStaff(h)) })
	staffModule.RegisterRoutes(mux, authed)
	vendorsModule.RegisterRoutes(mux, authed)
	projectsModule.RegisterRoutes(mux, authed)
	clientsModule.RegisterRoutes(mux, authed)

	// Serves the built frontend (see infra/docker/Dockerfile, which copies
	// apps/web's build output here) — a no-op locally, where the frontend
	// runs on its own Vite dev server (`npm run dev:web`) instead.
	mux.Handle("/", spaFileServer("./public"))

	handler := middleware.CORS(cfg.AppEnv == "development")(mux)

	log.Printf("%s api listening on :%s (%s)", cfg.AppName, cfg.AppPort, cfg.AppEnv)
	server := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: handler,
		// WriteTimeout is looser than ReadTimeout so it doesn't clip legitimate
		// large evidence-upload responses.
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}

// spaFileServer serves static assets from root, falling back to
// root/index.html for any path that doesn't match a real file — required so
// React Router's client-side routes resolve on a hard refresh/direct link.
// Registered on "/", it would otherwise also swallow any unmatched
// /api/v1/... request into a fake 200 HTML response instead of a real 404,
// so those are rejected before falling back.
func spaFileServer(root string) http.Handler {
	fileServer := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
			return
		}
		fullPath := filepath.Join(root, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(fullPath); err != nil || info.IsDir() {
			http.ServeFile(w, r, filepath.Join(root, "index.html"))
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// isLocalURL reports whether raw points at this machine (loopback only).
func isLocalURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}
