// Package vendors wires the vendors module: vendor categories + vendor
// directory, both tenant-scoped, plus (ADR-0016) the Venue directory — its
// own table, but still this module, since Venue and Vendor are conceptually
// siblings. Vendor CRUD itself needs no orchestration — see PLAN.md Fase 3 —
// but "Lihat Project" (a vendor's engagement history across projects)
// resolves through `projects.Contracts`, since `project_vendors` is owned by
// `projects`, not this module.
package vendors

import (
	"database/sql"
	"net/http"

	projectscontracts "elproof/internal/modules/projects/contracts"
	"elproof/internal/modules/vendors/application"
	vendorscontracts "elproof/internal/modules/vendors/contracts"
	"elproof/internal/modules/vendors/infrastructure"
	"elproof/internal/modules/vendors/presentation"
	"elproof/internal/shared/storage"
)

type Module struct {
	categoryHandler *presentation.VendorCategoryHandler
	vendorHandler   *presentation.VendorHandler
	venueHandler    *presentation.VenueHandler
	contracts       vendorscontracts.Contracts
}

func NewModule(db *sql.DB, projects projectscontracts.Contracts, storageClient *storage.Client) *Module {
	categoryRepo := infrastructure.NewMySQLVendorCategoryRepository(db)
	vendorRepo := infrastructure.NewMySQLVendorRepository(db)
	venueRepo := infrastructure.NewMySQLVenueRepository(db)

	categoryService := application.NewVendorCategoryService(categoryRepo)
	vendorService := application.NewVendorService(vendorRepo, categoryRepo, storageClient, storage.BuildKey)
	venueService := application.NewVenueService(venueRepo, storageClient, storage.BuildKey)

	return &Module{
		categoryHandler: presentation.NewVendorCategoryHandler(categoryService),
		vendorHandler:   presentation.NewVendorHandler(vendorService, projects),
		venueHandler:    presentation.NewVenueHandler(venueService),
		contracts:       vendorscontracts.New(categoryService, venueService),
	}
}

// Contracts exposes vendors' cross-module surface — used by `platform` to
// seed a new tenant's default vendor categories on registration, and (ADR-
// 0016) by `projects` to resolve a project's attached venue_id into display
// data. See the two-phase wiring note in main.go: both `platform` and
// `projects` are built before `vendors`, so this can't be either module's
// constructor argument.
func (m *Module) Contracts() vendorscontracts.Contracts {
	return m.contracts
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/vendor-categories", authed(http.HandlerFunc(m.categoryHandler.Collection)))
	mux.Handle("/api/v1/vendor-categories/", authed(http.HandlerFunc(m.categoryHandler.Item)))
	mux.Handle("/api/v1/vendors", authed(http.HandlerFunc(m.vendorHandler.Collection)))
	mux.Handle("/api/v1/vendors/template", authed(http.HandlerFunc(m.vendorHandler.Template)))
	mux.Handle("/api/v1/vendors/import", authed(http.HandlerFunc(m.vendorHandler.Import)))
	mux.Handle("/api/v1/vendors/summary", authed(http.HandlerFunc(m.vendorHandler.Summary)))
	mux.Handle("/api/v1/vendors/", authed(http.HandlerFunc(m.vendorHandler.Item)))
	mux.Handle("/api/v1/venues", authed(http.HandlerFunc(m.venueHandler.Collection)))
	mux.Handle("/api/v1/venues/template", authed(http.HandlerFunc(m.venueHandler.Template)))
	mux.Handle("/api/v1/venues/import", authed(http.HandlerFunc(m.venueHandler.Import)))
	mux.Handle("/api/v1/venues/", authed(http.HandlerFunc(m.venueHandler.Item)))
}
