package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"elproof/internal/adminseed"
	"elproof/internal/migrator"
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

// main dispatches on an optional subcommand so the one compiled binary this
// project ships as its deploy image (see infra/docker/Dockerfile) is
// everything a production host needs — no separate migrate CLI, seed
// binary, or curl/wget in the container for Docker's HEALTHCHECK. With no
// argument it serves, exactly as before this dispatch existed.
//
//	./api                 serve the API (default, unchanged behavior)
//	./api migrate up|down  apply/roll back one embedded SQL migration step
//	./api seed             reset to the minimal clean-slate dataset
//	./api healthcheck      exit 0/1 for Docker's HEALTHCHECK (self GET /api/v1/health)
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
		}
	}

	serve(cfg)
}

func runMigrate(cfg config.Config, args []string) {
	if len(args) != 1 || (args[0] != "up" && args[0] != "down") {
		log.Fatal("usage: api migrate up|down")
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
