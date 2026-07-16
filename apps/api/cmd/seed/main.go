// Command seed resets elproof_db to a minimal clean-slate state: one
// Platform Console super admin account and one subscription plan — nothing
// else. Safe to re-run. See internal/adminseed for the actual logic, which
// is shared with `cmd/server`'s `seed` subcommand (the image the VPS
// deploys never has this separate binary, only cmd/server — see
// docs/DEPLOYMENT.md).
package main

import (
	"context"
	"log"

	"elproof/internal/adminseed"
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

	if err := adminseed.Run(context.Background(), db); err != nil {
		log.Fatal(err)
	}
}
