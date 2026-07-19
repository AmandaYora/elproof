// Package vendors wires the vendors module: vendor categories + vendor
// directory, both tenant-scoped. Vendor CRUD itself needs no orchestration —
// see PLAN.md Fase 3 — but "Lihat Project" (a vendor's engagement history
// across projects) resolves through `projects.Contracts`, since
// `project_vendors` is owned by `projects`, not this module.
package vendors

import (
	"database/sql"
	"net/http"

	projectscontracts "elproof/internal/modules/projects/contracts"
	"elproof/internal/modules/vendors/application"
	vendorscontracts "elproof/internal/modules/vendors/contracts"
	"elproof/internal/modules/vendors/infrastructure"
	"elproof/internal/modules/vendors/presentation"
)

type Module struct {
	categoryHandler *presentation.VendorCategoryHandler
	vendorHandler   *presentation.VendorHandler
	contracts       vendorscontracts.Contracts
}

func NewModule(db *sql.DB, projects projectscontracts.Contracts) *Module {
	categoryRepo := infrastructure.NewMySQLVendorCategoryRepository(db)
	vendorRepo := infrastructure.NewMySQLVendorRepository(db)

	categoryService := application.NewVendorCategoryService(categoryRepo)
	vendorService := application.NewVendorService(vendorRepo, categoryRepo)

	return &Module{
		categoryHandler: presentation.NewVendorCategoryHandler(categoryService),
		vendorHandler:   presentation.NewVendorHandler(vendorService, projects),
		contracts:       vendorscontracts.New(categoryService),
	}
}

// Contracts exposes vendors' cross-module surface — currently used by
// `platform` to seed a new tenant's default vendor categories on
// registration. See the two-phase wiring note in main.go: `platform` is
// built before `vendors`, so this can't be a `platform.NewModule` argument.
func (m *Module) Contracts() vendorscontracts.Contracts {
	return m.contracts
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/vendor-categories", authed(http.HandlerFunc(m.categoryHandler.Collection)))
	mux.Handle("/api/v1/vendor-categories/", authed(http.HandlerFunc(m.categoryHandler.Item)))
	mux.Handle("/api/v1/vendors", authed(http.HandlerFunc(m.vendorHandler.Collection)))
	mux.Handle("/api/v1/vendors/", authed(http.HandlerFunc(m.vendorHandler.Item)))
}
