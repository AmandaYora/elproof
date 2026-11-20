package application

import (
	"context"
	"io"
	"time"

	"elproof/internal/modules/offers/domain"
	"elproof/internal/shared/pagination"
)

// QuotationListFilter narrows GET /quotations. Tab is "" (every status),
// a stored status, or Expired (a Draft past its validity, computed against
// Today). ExpiringSoon keeps Drafts valid through Today..Today+7.
// OnlyPICStaffID is the `assigned` scope (a person sees the quotations they
// handle); PICStaffID is the screen's PIC filter. A group of alternatives
// matches when any of its members does.
type QuotationListFilter struct {
	Search         string
	Tab            domain.QuotationStatus
	ExpiringSoon   bool
	PICStaffID     *int64
	OnlyPICStaffID *int64
	Today          time.Time
}

// LineRow is a line to write with the position of its parent among the rows
// being written (-1 = a top-level line).
type LineRow struct {
	Line   domain.QuotationLine
	Parent int
}

// QuotationRepository stores quotations and everything in them. Every call is
// scoped to a tenant: a row of another tenant reads as missing (ADR-0004).
// Writes that need to be atomic run inside Transactor.InTx; Lock takes the
// quotation row's write lock for the rest of that transaction.
type QuotationRepository interface {
	// NextSequence moves the tenant's counter for a WIB day one step and
	// returns it (SELECT … FOR UPDATE inside the caller's transaction).
	NextSequence(ctx context.Context, tenantID int64, docDate time.Time) (int, error)
	Create(ctx context.Context, q *domain.Quotation) error
	// FindByID is the quotation (nil when missing, deleted, or another
	// tenant's).
	FindByID(ctx context.Context, tenantID, id int64) (*domain.Quotation, error)
	// Lock is FindByID with the row locked FOR UPDATE.
	Lock(ctx context.Context, tenantID, id int64) (*domain.Quotation, error)
	// SaveHeader writes couple names, contact, PIC, validity, and notes, and
	// moves the header's version.
	SaveHeader(ctx context.Context, q *domain.Quotation) error
	// SaveState writes status, validity, and every acceptance, rejection,
	// replacement, and conversion field, and moves the header's version.
	SaveState(ctx context.Context, q *domain.Quotation) error
	// SaveTermsText writes the terms & conditions text and moves the terms'
	// version.
	SaveTermsText(ctx context.Context, tenantID, id int64, text string) error
	// SetCaches writes the list's caches (contract value, earliest event) and
	// marks the quotation changed. bumpLines/bumpTerms also move those versions.
	SetCaches(ctx context.Context, tenantID, id int64, total int64, mainEventDate *time.Time, bumpLines, bumpTerms bool) error
	SoftDelete(ctx context.Context, tenantID, id int64) error
	ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, filter QuotationListFilter, sort pagination.Sort) ([]domain.Quotation, int64, error)
	ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort)
	// StatusCounts is how many groups match the filter per tab (its Tab and
	// ExpiringSoon are ignored).
	StatusCounts(ctx context.Context, tenantID int64, filter QuotationListFilter) (map[domain.QuotationStatus]int, error)
	// Alternatives is every live alternative of the given group roots.
	Alternatives(ctx context.Context, tenantID int64, rootIDs []int64) ([]domain.Quotation, error)
	// FindByProject is the quotation a project was made from (nil when none).
	FindByProject(ctx context.Context, tenantID, projectID int64) (*domain.Quotation, error)

	// LoadContent is the events (with sessions), suppliers, lines, and terms.
	LoadContent(ctx context.Context, tenantID, quotationID int64) (*domain.WorkingCopy, error)
	InsertEvent(ctx context.Context, tenantID int64, e *domain.QuotationEvent) error
	// UpdateEvent writes every field and replaces the sessions.
	UpdateEvent(ctx context.Context, tenantID int64, e *domain.QuotationEvent) error
	DeleteEvent(ctx context.Context, tenantID, quotationID, eventID int64) error
	SetEventOrder(ctx context.Context, tenantID, quotationID int64, ids []int64) error
	InsertSupplier(ctx context.Context, tenantID int64, s *domain.QuotationSupplier) error
	UpdateSupplier(ctx context.Context, tenantID int64, s *domain.QuotationSupplier) error
	DeleteSupplier(ctx context.Context, tenantID, quotationID, supplierID int64) error
	// ReplaceLines removes every line and writes rows in order.
	ReplaceLines(ctx context.Context, tenantID, quotationID int64, rows []LineRow) error
	// SetLineAmounts writes computed amounts (extra guests) by line id.
	SetLineAmounts(ctx context.Context, tenantID, quotationID int64, amounts map[int64]int64) error
	ReplaceTerms(ctx context.Context, tenantID, quotationID int64, terms []domain.QuotationTerm) error

	// FindRevisionByHash is the revision with that content (nil when none).
	FindRevisionByHash(ctx context.Context, tenantID, quotationID int64, hash string) (*domain.QuotationRevision, error)
	FindRevision(ctx context.Context, tenantID, quotationID int64, n int) (*domain.QuotationRevision, error)
	FindRevisionByID(ctx context.Context, tenantID, revisionID int64) (*domain.QuotationRevision, error)
	ListRevisions(ctx context.Context, tenantID, quotationID int64) ([]domain.QuotationRevision, error)
	// InsertRevision stores the revision and sets the quotation's
	// revision_count to its number.
	InsertRevision(ctx context.Context, tenantID int64, r *domain.QuotationRevision) error

	AddActivity(ctx context.Context, tenantID int64, a domain.Activity) error
	ListActivity(ctx context.Context, tenantID, quotationID int64) ([]domain.Activity, error)

	// QuotationUsage answers the catalog's and the directory's reference checks.
	QuotationUsage
}

// QuotationUsage is what the catalog and the directory ask before a
// permanent delete: does a quotation of the tenant still refer to it.
type QuotationUsage interface {
	PackageReferenced(ctx context.Context, tenantID, packageID int64) (bool, error)
	VendorReferenced(ctx context.Context, tenantID, vendorID int64) (bool, error)
	VenueReferenced(ctx context.Context, tenantID, venueID int64) (bool, error)
	CategoryReferenced(ctx context.Context, tenantID, categoryID int64) (bool, error)
	// EventKindQuotationCounts is how many quotations have an event of each kind.
	EventKindQuotationCounts(ctx context.Context, tenantID int64) (map[int64]int, error)
}

// Transactor runs a function inside one database transaction of this module.
type Transactor interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// QuotationProfile is what a quotation needs of the workspace (platform's
// business profile and Penawaran settings): the document code of its number,
// PKP, the default validity and terms text, and the PDF letterhead.
type QuotationProfile struct {
	DocumentCode        string
	IsPKP               bool
	DefaultValidityDays int
	DefaultTermsText    string
	Identity            domain.WorkspaceIdentity
}

// WorkspaceProfile reads the profile of a workspace; the adaptor in main.go
// fills it from platform (ADR-0049: offers does not import platform).
type WorkspaceProfile interface {
	QuotationProfile(ctx context.Context, tenantID int64) (QuotationProfile, error)
}

// StaffNameResolver gives staff names of a workspace (offers never imports
// staff/contracts; ADR-0049). GetName answers found=false for an id that is
// not a member of the tenant.
type StaffNameResolver interface {
	GetName(ctx context.Context, tenantID, staffID int64) (string, bool, error)
	StaffNames(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error)
}

// FileStore is the object storage (ADR-0006): the WO's logo and the evidence
// of acceptance.
type FileStore interface {
	Save(ctx context.Context, key string, data []byte, contentType string) (string, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
}

// DocumentRenderer turns a prepared document into PDF bytes
// (infrastructure/pdf; no domain logic there).
type DocumentRenderer interface {
	Render(doc QuotationDocument) ([]byte, error)
}
