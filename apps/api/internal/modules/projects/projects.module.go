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
	references               *application.References
	handler                  *presentation.Handler
	milestoneTemplateHandler *presentation.MilestoneTemplateHandler
	contracts                contracts.Contracts
	projectService           *application.ProjectService
	clientHome               *application.ClientHomeService
}

// NewModule builds the module. signingSecret is the application's one secret
// (the JWT secret); the signed-upload tokens derive their own purpose-bound key
// from it, so there is no second secret to configure.
func NewModule(db *sql.DB, storageClient *storage.Client, staff application.StaffNameResolver, signingSecret string) *Module {
	projectRepo := infrastructure.NewMySQLProjectRepository(db)
	milestoneRepo := infrastructure.NewMySQLMilestoneRepository(db)
	milestoneTemplateRepo := infrastructure.NewMySQLMilestoneTemplateRepository(db)
	vendorEngagementRepo := infrastructure.NewMySQLVendorEngagementRepository(db)
	vendorMilestoneRepo := infrastructure.NewMySQLVendorMilestoneRepository(db)
	paymentRepo := infrastructure.NewMySQLPaymentRepository(db)
	clientPaymentRepo := infrastructure.NewMySQLClientPaymentRepository(db)
	paymentTermRepo := infrastructure.NewMySQLPaymentTermRepository(db)
	venuePaymentRepo := infrastructure.NewMySQLVenuePaymentRepository(db)
	issueRepo := infrastructure.NewMySQLIssueRepository(db)
	evidenceRepo := infrastructure.NewMySQLEvidenceRepository(db)
	activityRepo := infrastructure.NewMySQLActivityRepository(db)
	dashboardRepo := infrastructure.NewMySQLDashboardRepository(db)
	idempotencyRepo := infrastructure.NewMySQLIdempotencyRepository(db)
	txManager := infrastructure.NewTxManager(db)

	activityService := application.NewActivityService(activityRepo)
	evidenceService := application.NewEvidenceService(evidenceRepo, storageClient, storage.BuildKey, activityService)
	projectService := application.NewProjectService(projectRepo, milestoneRepo, milestoneTemplateRepo, vendorEngagementRepo, vendorMilestoneRepo, issueRepo, paymentRepo, venuePaymentRepo, evidenceService, activityService, staff)
	milestoneTemplateService := application.NewMilestoneTemplateService(milestoneTemplateRepo)
	vendorEngagementService := application.NewVendorEngagementService(vendorEngagementRepo, vendorMilestoneRepo, activityService)
	contractReader := application.NewContractReader(projectService, vendorEngagementRepo)
	paymentService := application.NewPaymentService(paymentRepo, evidenceService, activityService, contractReader)
	clientPaymentService := application.NewClientPaymentService(clientPaymentRepo, evidenceService, activityService, contractReader)
	venuePaymentService := application.NewVenuePaymentService(venuePaymentRepo, evidenceService, activityService, contractReader)
	issueService := application.NewIssueService(issueRepo, vendorMilestoneRepo, vendorEngagementRepo, activityService)
	dashboardService := application.NewDashboardService(projectService, dashboardRepo, evidenceService)
	paymentTermService := application.NewPaymentTermService(paymentTermRepo, clientPaymentRepo, projectService, activityService)
	myProjectsService := application.NewMyProjectsService(projectService, paymentTermService)
	closingService := application.NewClosingChecklistService(projectService, clientPaymentRepo, paymentRepo, venuePaymentRepo, vendorEngagementRepo, issueRepo)

	clientHomeService := application.NewClientHomeService(projectService, clientPaymentRepo, paymentTermService, evidenceService, issueRepo)

	// Signed uploads, atomic payment + proof, atomic project creation (Tahap 4).
	uploadService := application.NewUploadService(application.NewUploadSigner(signingSecret), storageClient, evidenceRepo, storage.BuildKey)
	proofSupport := application.NewProofSupport(txManager, uploadService, evidenceService)
	paymentService.SetProofSupport(proofSupport)
	venuePaymentService.SetProofSupport(proofSupport)
	clientPaymentService.SetAtomicSupport(proofSupport, paymentTermRepo)
	projectService.SetAtomicSupport(txManager, paymentTermRepo)

	references := application.NewReferences(staff)
	handler := presentation.NewHandler(projectService, vendorEngagementService, paymentService, clientPaymentService, venuePaymentService, issueService, evidenceService, activityService, dashboardService, closingService, paymentTermService, myProjectsService)

	handler.SetReferences(references)
	handler.SetClientHome(clientHomeService)
	handler.SetUploads(uploadService)
	handler.SetIdempotency(application.NewIdempotencyService(idempotencyRepo))

	return &Module{
		references:               references,
		handler:                  handler,
		milestoneTemplateHandler: presentation.NewMilestoneTemplateHandler(milestoneTemplateService),
		contracts:                contracts.New(projectService, vendorEngagementService, milestoneTemplateService),
		projectService:           projectService,
		clientHome:               clientHomeService,
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

// SetClientCleaner completes the same two-phase wiring as
// SetClientAccessResolver above, for ProjectService.Delete's (ADR-0013)
// best-effort client cleanup — see application.ClientCleaner's doc comment.
func (m *Module) SetClientCleaner(cleaner application.ClientCleaner) {
	m.projectService.SetClientCleaner(cleaner)
}

// SetVenueResolver completes the same two-phase wiring, for resolving a
// project's attached venue_id into display data (ADR-0016) — see
// application.VenueResolver's doc comment.
// SetTrialQuota plugs in the platform module's trial caps (main.go).
func (m *Module) SetTrialQuota(q application.TrialQuota) {
	m.projectService.SetTrialQuota(q)
}

// SetVendorDirectory completes the same two-phase wiring, so a vendor or a
// category named in a request body can be confirmed to be this workspace's.
func (m *Module) SetVendorDirectory(d application.VendorDirectory) {
	m.references.SetVendors(d)
}

// SetTenantContact completes the two-phase wiring with `platform`, whose
// business profile holds the WhatsApp number and bank account the Client
// Portal home shows (see application.TenantContactReader).
func (m *Module) SetTenantContact(c application.TenantContactReader) {
	m.clientHome.SetTenantContact(c)
}

func (m *Module) SetVenueResolver(resolver application.VenueResolver) {
	m.projectService.SetVenueResolver(resolver)
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/projects", authed(http.HandlerFunc(m.handler.Collection)))
	mux.Handle("/api/v1/projects/", authed(http.HandlerFunc(m.handler.Item)))
	mux.Handle("/api/v1/dashboard", authed(http.HandlerFunc(m.handler.Dashboard)))
	mux.Handle("/api/v1/dashboard/mine", authed(http.HandlerFunc(m.handler.MyDashboard)))
	mux.Handle("/api/v1/milestone-templates", authed(http.HandlerFunc(m.milestoneTemplateHandler.Collection)))
	mux.Handle("/api/v1/milestone-templates/", authed(http.HandlerFunc(m.milestoneTemplateHandler.Item)))
}
