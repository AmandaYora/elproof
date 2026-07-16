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
	"elproof/internal/modules/vendors/infrastructure"
	"elproof/internal/modules/vendors/presentation"
)

type Module struct {
	categoryHandler *presentation.VendorCategoryHandler
	vendorHandler   *presentation.VendorHandler
}

func NewModule(db *sql.DB, projects projectscontracts.Contracts) *Module {
	categoryRepo := infrastructure.NewMySQLVendorCategoryRepository(db)
	vendorRepo := infrastructure.NewMySQLVendorRepository(db)

	categoryService := application.NewVendorCategoryService(categoryRepo)
	vendorService := application.NewVendorService(vendorRepo, categoryRepo)

	return &Module{
		categoryHandler: presentation.NewVendorCategoryHandler(categoryService),
		vendorHandler:   presentation.NewVendorHandler(vendorService, projects),
	}
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/vendor-categories", authed(http.HandlerFunc(m.categoryHandler.Collection)))
	mux.Handle("/api/v1/vendor-categories/", authed(http.HandlerFunc(m.categoryHandler.Item)))
	mux.Handle("/api/v1/vendors", authed(http.HandlerFunc(m.vendorHandler.Collection)))
	mux.Handle("/api/v1/vendors/", authed(http.HandlerFunc(m.vendorHandler.Item)))
}
