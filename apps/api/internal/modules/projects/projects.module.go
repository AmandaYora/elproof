// Package projects wires the largest module in the system: the project
// lifecycle and its 7 sub-entities (milestones, vendor engagements, vendor
// milestones, payments, issues, evidence, activity log) — see MODULE_MAP.md.
package projects

import (
	"database/sql"
	"net/http"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/contracts"
	"elproof/internal/modules/projects/infrastructure"
	"elproof/internal/modules/projects/presentation"
	"elproof/internal/shared/storage"
)

type Module struct {
	handler   *presentation.Handler
	contracts contracts.Contracts
}

func NewModule(db *sql.DB, storageClient *storage.Client) *Module {
	projectRepo := infrastructure.NewMySQLProjectRepository(db)
	milestoneRepo := infrastructure.NewMySQLMilestoneRepository(db)
	vendorEngagementRepo := infrastructure.NewMySQLVendorEngagementRepository(db)
	vendorMilestoneRepo := infrastructure.NewMySQLVendorMilestoneRepository(db)
	paymentRepo := infrastructure.NewMySQLPaymentRepository(db)
	issueRepo := infrastructure.NewMySQLIssueRepository(db)
	evidenceRepo := infrastructure.NewMySQLEvidenceRepository(db)
	activityRepo := infrastructure.NewMySQLActivityRepository(db)
	dashboardRepo := infrastructure.NewMySQLDashboardRepository(db)

	activityService := application.NewActivityService(activityRepo)
	projectService := application.NewProjectService(projectRepo, milestoneRepo, vendorEngagementRepo, vendorMilestoneRepo, issueRepo, paymentRepo, activityService)
	vendorEngagementService := application.NewVendorEngagementService(vendorEngagementRepo, vendorMilestoneRepo, activityService)
	paymentService := application.NewPaymentService(paymentRepo, activityService)
	issueService := application.NewIssueService(issueRepo, activityService)
	evidenceService := application.NewEvidenceService(evidenceRepo, storageClient, storage.BuildKey, activityService)
	dashboardService := application.NewDashboardService(projectService, dashboardRepo)

	handler := presentation.NewHandler(projectService, vendorEngagementService, paymentService, issueService, evidenceService, activityService, dashboardService)

	return &Module{
		handler:   handler,
		contracts: contracts.New(projectService, vendorEngagementService),
	}
}

func (m *Module) Contracts() contracts.Contracts {
	return m.contracts
}

// SetClientAccessResolver completes Fase 6's client-portal scoping. It must
// be called from main.go after clients.NewModule exists — see the
// presentation.ClientAccessResolver doc comment for why this can't just be
// a constructor parameter (clients.NewModule itself needs projects.Contracts()
// already built, so the two modules can't be constructed in either order
// with a direct constructor dependency both ways).
func (m *Module) SetClientAccessResolver(resolver presentation.ClientAccessResolver) {
	m.handler.SetClientAccessResolver(resolver)
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/projects", authed(http.HandlerFunc(m.handler.Collection)))
	mux.Handle("/api/v1/projects/", authed(http.HandlerFunc(m.handler.Item)))
	mux.Handle("/api/v1/dashboard", authed(http.HandlerFunc(m.handler.Dashboard)))
}
