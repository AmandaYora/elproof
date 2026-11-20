// Package projects wires the largest module in the system: the project
// lifecycle and its sub-entities (events, vendors & venues with their steps
// and payments, client payments, milestones, issues, evidence, activity log)
// — see MODULE_MAP.md.
package projects

import (
	"context"
	"database/sql"
	"net/http"
	"time"

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
	overview                 *application.OverviewService
	purge                    *application.PurgeService
	events                   *application.EventService
	lines                    *application.LineService
}

// PurgeSummary is what one purge run removed, in plain numbers (the composition
// root logs it; it never sees this module's domain types).
type PurgeSummary struct {
	// Ran is false when another instance held the purge lock.
	Ran bool
	// Rows counts the rows removed (payments, vendors & venues and what hung
	// off them); Evidence the proof rows among them; Objects the stored files
	// deleted from object storage.
	Rows, Evidence, Objects int
}

// NewModule builds the module. signingSecret is the application's one secret
// (the JWT secret); the signed-upload tokens derive their own purpose-bound key
// from it, so there is no second secret to configure.
func NewModule(db *sql.DB, storageClient *storage.Client, staff application.StaffNameResolver, signingSecret string) *Module {
	projectRepo := infrastructure.NewMySQLProjectRepository(db)
	milestoneRepo := infrastructure.NewMySQLMilestoneRepository(db)
	milestoneTemplateRepo := infrastructure.NewMySQLMilestoneTemplateRepository(db)
	supplierRepo := infrastructure.NewMySQLSupplierRepository(db)
	vendorMilestoneRepo := infrastructure.NewMySQLVendorMilestoneRepository(db)
	supplierPaymentRepo := infrastructure.NewMySQLSupplierPaymentRepository(db)
	eventRepo := infrastructure.NewMySQLEventRepository(db)
	clientPaymentRepo := infrastructure.NewMySQLClientPaymentRepository(db)
	paymentTermRepo := infrastructure.NewMySQLPaymentTermRepository(db)
	issueRepo := infrastructure.NewMySQLIssueRepository(db)
	evidenceRepo := infrastructure.NewMySQLEvidenceRepository(db)
	activityRepo := infrastructure.NewMySQLActivityRepository(db)
	dashboardRepo := infrastructure.NewMySQLDashboardRepository(db)
	idempotencyRepo := infrastructure.NewMySQLIdempotencyRepository(db)
	txManager := infrastructure.NewTxManager(db)

	activityService := application.NewActivityService(activityRepo)
	evidenceService := application.NewEvidenceService(evidenceRepo, storageClient, storage.BuildKey, activityService)
	projectService := application.NewProjectService(projectRepo, milestoneRepo, milestoneTemplateRepo, supplierRepo, vendorMilestoneRepo, issueRepo, supplierPaymentRepo, eventRepo, evidenceService, activityService, staff)
	milestoneTemplateService := application.NewMilestoneTemplateService(milestoneTemplateRepo)
	supplierService := application.NewSupplierService(supplierRepo, vendorMilestoneRepo, eventRepo, supplierPaymentRepo, activityService)
	eventService := application.NewEventService(eventRepo, supplierService, projectService, txManager, activityService)
	projectService.SetEvents(eventService)
	supplierService.SetCapacity(eventService)
	contractReader := application.NewContractReader(projectService, supplierRepo)
	supplierPaymentService := application.NewSupplierPaymentService(supplierPaymentRepo, evidenceService, activityService, contractReader)
	clientPaymentService := application.NewClientPaymentService(clientPaymentRepo, evidenceService, activityService, contractReader)
	issueService := application.NewIssueService(issueRepo, vendorMilestoneRepo, supplierRepo, activityService)
	dashboardService := application.NewDashboardService(projectService, dashboardRepo, evidenceService)
	paymentTermService := application.NewPaymentTermService(paymentTermRepo, clientPaymentRepo, projectService, activityService)
	myProjectsService := application.NewMyProjectsService(projectService, paymentTermService)
	closingService := application.NewClosingChecklistService(projectService, clientPaymentRepo, supplierPaymentRepo, supplierRepo, issueRepo)

	clientHomeService := application.NewClientHomeService(projectService, clientPaymentRepo, paymentTermService, evidenceService, issueRepo)

	// Signed uploads, atomic payment + proof, atomic project creation (Tahap 4).
	uploadService := application.NewUploadService(application.NewUploadSigner(signingSecret), storageClient, evidenceRepo, storage.BuildKey)
	proofSupport := application.NewProofSupport(txManager, uploadService, evidenceService)
	supplierPaymentService.SetProofSupport(proofSupport)
	clientPaymentService.SetAtomicSupport(proofSupport, paymentTermRepo)
	// Paket WO Tahap 4: bills of the vendors the client pays directly and the
	// proofs the couple sends for them (payer = Client, never the WO's money).
	supplierBillService := application.NewSupplierBillService(infrastructure.NewMySQLSupplierBillRepository(db), supplierPaymentRepo,
		supplierPaymentRepo, supplierRepo, evidenceService, activityService)
	supplierBillService.SetProofSupport(proofSupport)
	clientHomeService.SetSupplierBills(supplierBillService)
	projectService.SetAtomicSupport(txManager, paymentTermRepo)
	projectService.SetListMoney(infrastructure.NewMySQLListMoneyRepository(db))

	references := application.NewReferences(staff)

	// The Rincian (Tahap 3): lines, the contract cache and the extra-guest
	// charges kept in step with the events' guests in the same transaction.
	lineService := application.NewLineService(infrastructure.NewMySQLLineRepository(db), projectService, eventService, eventRepo,
		supplierRepo, paymentTermRepo, references, txManager, activityService)
	eventService.SetLines(lineService)
	projectService.SetLines(lineService)
	clientHomeService.SetLines(lineService)
	handler := presentation.NewHandler(projectService, supplierService, supplierPaymentService, eventService, clientPaymentService, issueService, evidenceService, activityService, dashboardService, closingService, paymentTermService, myProjectsService)

	// The project overview (E1), the dashboard's names and the scheduled purge of
	// what stayed deleted for longer than the retention (E13, ADR-0034).
	overviewService := application.NewOverviewService(projectService, clientPaymentRepo, paymentTermService)
	purgeService := application.NewPurgeService(infrastructure.NewMySQLPurgeRepository(db), evidenceService)

	handler.SetReferences(references)
	handler.SetLines(lineService)
	handler.SetSupplierBills(supplierBillService)
	// "Jadikan proyek" (Paket WO Tahap 5): `offers` is wired later (SetQuotations).
	handler.SetConversion(application.NewConversionService(projectService, lineService, clientPaymentService, projectRepo))
	handler.SetOverview(overviewService)
	handler.SetClientHome(clientHomeService)
	handler.SetUploads(uploadService)
	handler.SetIdempotency(application.NewIdempotencyService(idempotencyRepo))
	handler.SetCalendar(application.NewCalendarService(infrastructure.NewMySQLCalendarRepository(db)))
	handler.SetFinance(application.NewFinanceService(infrastructure.NewMySQLFinanceRepository(db)))
	// The Client Portal hero photo (ADR-0047).
	handler.SetCovers(application.NewCoverService(projectService, projectRepo, txManager, storageClient, storage.BuildKey, activityService))

	return &Module{
		references:               references,
		handler:                  handler,
		milestoneTemplateHandler: presentation.NewMilestoneTemplateHandler(milestoneTemplateService),
		contracts:                contracts.New(projectService, supplierService, milestoneTemplateService),
		projectService:           projectService,
		clientHome:               clientHomeService,
		overview:                 overviewService,
		purge:                    purgeService,
		events:                   eventService,
		lines:                    lineService,
	}
}

// PurgeExpired permanently removes what has been deleted for longer than the
// retention: payments and vendors & venues, their evidence rows and stored
// files. Idempotent and safe with several instances (a database lock lets one
// work at a time). It is what `api purge` and the server's ticker run.
func (m *Module) PurgeExpired(ctx context.Context, now time.Time) (PurgeSummary, error) {
	report, ran, err := m.purge.Run(ctx, now)
	return PurgeSummary{Ran: ran, Rows: report.Total(), Evidence: report.Evidence, Objects: report.Objects}, err
}

// SetClientDirectory completes the two-phase wiring with `clients` for the
// overview's client list (see application.ClientDirectory).
func (m *Module) SetClientDirectory(d application.ClientDirectory) {
	m.overview.SetClients(d)
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

// SetEventCatalog completes the two-phase wiring with `offers`: the event kinds
// a project event copies and the one capacity rule (PLAN §4.1).
func (m *Module) SetEventCatalog(c application.EventCatalog) {
	m.events.SetCatalog(c)
}

// SetOffers completes the two-phase wiring with `offers` for the Rincian: a
// package as a line copies it and the pricing rules (ADR-0049).
func (m *Module) SetOffers(o application.OffersPort) {
	m.lines.SetOffers(o)
}

// SetQuotations completes the two-phase wiring with `offers` for "Jadikan
// proyek" (PLAN §5.4): the accepted revision a project is made from, and the
// marks MarkConverted / MarkProjectDeleted.
func (m *Module) SetQuotations(q application.QuotationSource) {
	m.projectService.SetQuotations(q)
}

// SetWorkspaceProfile completes the two-phase wiring with `platform`: whether
// the workspace is PKP, copied onto a project when it is created.
func (m *Module) SetWorkspaceProfile(w application.WorkspaceProfileSource) {
	m.projectService.SetWorkspaceProfile(w)
}

// SetEventVenues completes the two-phase wiring with `vendors`: the directory
// venue an event takes place at, with its capacities.
func (m *Module) SetEventVenues(v application.EventVenues) {
	m.events.SetVenues(v)
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/projects", authed(http.HandlerFunc(m.handler.Collection)))
	mux.Handle("/api/v1/projects/", authed(http.HandlerFunc(m.handler.Item)))
	mux.Handle("/api/v1/dashboard", authed(http.HandlerFunc(m.handler.Dashboard)))
	mux.Handle("/api/v1/dashboard/mine", authed(http.HandlerFunc(m.handler.MyDashboard)))
	mux.Handle("/api/v1/calendar", authed(http.HandlerFunc(m.handler.Calendar)))
	// Keuangan: money across the caller's projects (finance.record).
	mux.Handle("/api/v1/finance/summary", authed(http.HandlerFunc(m.handler.FinanceSummary)))
	mux.Handle("/api/v1/finance/receivables", authed(http.HandlerFunc(m.handler.FinanceReceivables)))
	mux.Handle("/api/v1/finance/payables", authed(http.HandlerFunc(m.handler.FinancePayables)))
	mux.Handle("/api/v1/finance/verifications", authed(http.HandlerFunc(m.handler.FinanceVerifications)))
	mux.Handle("/api/v1/milestone-templates", authed(http.HandlerFunc(m.milestoneTemplateHandler.Collection)))
	mux.Handle("/api/v1/milestone-templates/", authed(http.HandlerFunc(m.milestoneTemplateHandler.Item)))
}
