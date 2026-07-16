package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"elproof/internal/modules/billing"
	"elproof/internal/modules/clients"
	"elproof/internal/modules/identity"
	"elproof/internal/modules/payment"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/modules/platform"
	"elproof/internal/modules/projects"
	"elproof/internal/modules/staff"
	"elproof/internal/modules/vendors"
	"elproof/internal/shared/config"
	"elproof/internal/shared/database"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
	"elproof/internal/shared/storage"
)

func main() {
	cfg := config.Load()

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		response.OK(w, "ok", nil)
	})

	identityModule := identity.NewModule(db, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	identityModule.RegisterRoutes(mux)

	authed := middleware.RequireAuth(cfg.JWTSecret)

	mux.Handle("/api/v1/auth/me", authed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := middleware.FromContext(r.Context())
		response.OK(w, "ok", claims)
	})))

	storageClient, err := storage.New(storage.Config{
		Endpoint: cfg.S3Endpoint, Bucket: cfg.S3Bucket,
		AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey, UseSSL: cfg.S3UseSSL,
	})
	if err != nil {
		log.Fatalf("failed to create object storage client: %v", err)
	}

	// payment is built first — it depends on no other module (see
	// MODULE_PAYMENT.md §3), so any App internal (e.g. platform) can receive
	// its Client at their own construction time.
	paymentModule, err := payment.NewModule(db, cfg.PaymentEncryptionKey)
	if err != nil {
		log.Fatalf("failed to init payment module: %v", err)
	}

	staffModule := staff.NewModule(db)
	billingModule := billing.NewModule(db)
	platformModule := platform.NewModule(db, staffModule.Contracts(), identityModule.Contracts(), billingModule.Contracts(), paymentModule.Client())
	// projects is built before vendors — vendors' "Lihat Project" resolves a
	// vendor's cross-project engagement history through projects.Contracts()
	// (project_vendors is owned by projects, not vendors).
	projectsModule := projects.NewModule(db, storageClient)
	vendorsModule := vendors.NewModule(db, projectsModule.Contracts())
	clientsModule := clients.NewModule(db, projectsModule.Contracts(), identityModule.Contracts())
	// Two-phase wiring: projects needs clients' contract (Fase 6 client-portal
	// scoping) but clients.NewModule already needs projects.Contracts() to
	// exist first, so this can't be a constructor argument either direction.
	projectsModule.SetClientAccessResolver(clientsModule.Contracts())
	// Same bridging pattern: payment can't import platform (or any other App
	// internal) to know its webhook consumer, so main.go registers it here,
	// after both modules are built — see payment.module.go's Dispatcher doc.
	paymentModule.Dispatcher().RegisterConsumer(paymentcontracts.InternalAppBilling, platformModule)

	paymentModule.RegisterRoutes(mux, authed)
	billingModule.RegisterRoutes(mux, authed)
	platformModule.RegisterRoutes(mux, authed)
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
	log.Fatal(http.ListenAndServe(":"+cfg.AppPort, handler))
}

// spaFileServer serves static assets from root, falling back to
// root/index.html for any path that doesn't match a real file — required so
// React Router's client-side routes resolve on a hard refresh/direct link.
// Registered on "/", it would otherwise also swallow any unmatched
// /api/v1/... request (e.g. demo-login when disabled) into a fake 200 HTML
// response instead of a real 404, so those are rejected before falling back.
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
