// Package offers wires the offers module (docs/plan/paket-wo/PLAN.md §2.1,
// ADR-0049): the catalog of event kinds and packages, quotations (Tahap 5), and
// the pure pricing and capacity rules quotations and projects share. It uses
// vendors' contract directly; everything that needs offers (platform,
// vendors, later projects) gets it through a setter in main.go, so no import
// cycle can form.
package offers

import (
	"database/sql"
	"net/http"

	"elproof/internal/modules/offers/application"
	offerscontracts "elproof/internal/modules/offers/contracts"
	"elproof/internal/modules/offers/infrastructure"
	"elproof/internal/modules/offers/infrastructure/pdf"
	"elproof/internal/modules/offers/presentation"
)

type Module struct {
	kindHandler      *presentation.EventKindHandler
	packageHandler   *presentation.PackageHandler
	quotationHandler *presentation.QuotationHandler
	quotations       *application.QuotationService
	renderer         application.DocumentRenderer
	contracts        offerscontracts.Contracts
}

// NewModule builds the module; directory is vendors' contract (built before
// offers).
func NewModule(db *sql.DB, directory application.VendorDirectory) *Module {
	kindRepo := infrastructure.NewMySQLEventKindRepository(db)
	packageRepo := infrastructure.NewMySQLPackageRepository(db)
	quotationRepo := infrastructure.NewMySQLQuotationRepository(db)
	kinds := application.NewEventKindService(kindRepo)
	packages := application.NewPackageService(packageRepo, kindRepo, directory)
	// The catalog's reference rules count quotations too (PLAN §9 item 6).
	packages.SetQuotationUsage(quotationRepo)
	kinds.SetQuotationUsage(quotationRepo)
	quotations := application.NewQuotationService(quotationRepo, infrastructure.NewTxManager(db), packageRepo, kindRepo, directory)
	renderer := pdf.NewRenderer()
	quotations.SetPorts(nil, nil, nil, renderer)
	return &Module{
		kindHandler:      presentation.NewEventKindHandler(kinds),
		packageHandler:   presentation.NewPackageHandler(packages),
		quotationHandler: presentation.NewQuotationHandler(quotations),
		quotations:       quotations,
		renderer:         renderer,
		contracts:        offerscontracts.New(kinds, packages),
	}
}

// SetQuotationPorts completes quotations in main.go: the workspace profile
// (platform), staff names (staff), and the object storage. Until it is
// called, creating a quotation answers "Profil usaha belum tersedia".
func (m *Module) SetQuotationPorts(profile application.WorkspaceProfile, staff application.StaffNameResolver, files application.FileStore) {
	m.quotations.SetPorts(profile, staff, files, m.renderer)
}

// Quotations is what "Jadikan proyek" (projects) needs of quotations.
func (m *Module) Quotations() offerscontracts.Quotations {
	return offerscontracts.NewQuotations(m.quotations)
}

// Contracts exposes offers' cross-module surface.
func (m *Module) Contracts() offerscontracts.Contracts { return m.contracts }

func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/event-kinds", authed(http.HandlerFunc(m.kindHandler.Collection)))
	mux.Handle("/api/v1/event-kinds/", authed(http.HandlerFunc(m.kindHandler.Item)))
	mux.Handle("/api/v1/packages", authed(http.HandlerFunc(m.packageHandler.Collection)))
	mux.Handle("/api/v1/packages/preview", authed(http.HandlerFunc(m.packageHandler.Preview)))
	mux.Handle("/api/v1/packages/", authed(http.HandlerFunc(m.packageHandler.Item)))
	mux.Handle("/api/v1/quotations", authed(http.HandlerFunc(m.quotationHandler.Collection)))
	mux.Handle("/api/v1/quotations/summary", authed(http.HandlerFunc(m.quotationHandler.Summary)))
	mux.Handle("/api/v1/quotations/", authed(http.HandlerFunc(m.quotationHandler.Item)))
}
