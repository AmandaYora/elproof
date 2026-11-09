package main

import (
	"context"
	"log"
	"time"

	"elproof/internal/modules/clients"
	"elproof/internal/modules/identity"
	"elproof/internal/modules/projects"
	"elproof/internal/shared/config"
	"elproof/internal/shared/database"
	"elproof/internal/shared/storage"
)

// purgeAll permanently removes everything that has stayed deleted for longer than
// the retention (ADR-0034): payments and vendor engagements with their evidence
// rows and stored files (projects), and clients with their logins (clients). It
// is idempotent, and safe with several instances: each module takes a database
// lock, so a second instance simply finds the work taken and does nothing. It
// logs what it removed; an error in one module does not stop the other.
func purgeAll(ctx context.Context, projectsModule *projects.Module, clientsModule *clients.Module) {
	now := time.Now()
	if s, err := projectsModule.PurgeExpired(ctx, now); err != nil {
		log.Printf("purge: projects: %v", err)
	} else if s.Ran && s.Rows > 0 {
		log.Printf("purge: projects removed %d rows (%d evidence rows, %d stored files)", s.Rows, s.Evidence, s.Objects)
	}
	if s, err := clientsModule.PurgeExpired(ctx, now); err != nil {
		log.Printf("purge: clients: %v", err)
	} else if s.Ran && s.Removed > 0 {
		log.Printf("purge: clients removed %d", s.Removed)
	}
}

// startPurger runs purgeAll every interval for the lifetime of the process (the
// same fire-and-forget shape as the payment reconciler: no shutdown signal exists
// anywhere else in this server either). An interval of 0 or less switches it off.
func startPurger(ctx context.Context, interval time.Duration, projectsModule *projects.Module, clientsModule *clients.Module) {
	if interval <= 0 {
		return
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				purgeAll(ctx, projectsModule, clientsModule)
			}
		}
	}()
}

// runPurge is `api purge`: one purge pass, then exit. It builds only what the
// purge needs (the database, object storage, and the two modules with their
// dependencies), not the whole server.
func runPurge(cfg config.Config) {
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	storageClient, err := storage.New(storage.Config{
		Endpoint: cfg.S3Endpoint, Bucket: cfg.S3Bucket,
		AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey, UseSSL: cfg.S3UseSSL,
		Region: cfg.S3Region, Presign: cfg.S3Presign,
	})
	if err != nil {
		log.Fatalf("failed to create object storage client: %v", err)
	}
	identityModule := identity.NewModule(db, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL, identity.GoogleConfig{ClientID: cfg.GoogleClientID})
	// The projects module needs a staff resolver only for names; the purge never
	// asks for one.
	projectsModule := projects.NewModule(db, storageClient, nil, cfg.JWTSecret)
	clientsModule := clients.NewModule(db, projectsModule.Contracts(), identityModule.Contracts())
	purgeAll(context.Background(), projectsModule, clientsModule)
	log.Println("purge: done")
}
