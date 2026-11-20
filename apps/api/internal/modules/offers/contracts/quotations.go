package contracts

import (
	"context"
	"time"

	"elproof/internal/modules/offers/application"
	"elproof/internal/modules/offers/domain"
)

// Quotations is what "Jadikan proyek" (PLAN §5.4, owned by projects) needs of
// offers. Every call is scoped to the tenant from the caller's session: a
// quotation of another workspace answers 404 like a missing one (ADR-0004).
type Quotations interface {
	// AcceptedQuotation is an accepted quotation with the content of the
	// revision it was accepted on. A quotation that is not Accepted answers a
	// 422 with code quotation_not_accepted (a Superseded one can never be
	// converted); a missing one 404. One already converted still answers
	// (ConvertedProjectID set), so the conversion endpoint can stay
	// idempotent.
	AcceptedQuotation(ctx context.Context, tenantID, quotationID int64) (AcceptedQuotation, error)
	// MarkConverted records that projectID was made from the quotation; the
	// quotation becomes read-only. Idempotent: calling it again with the same
	// project is a no-op, with another project id it repairs the cache.
	MarkConverted(ctx context.Context, tenantID, quotationID, projectID int64) error
	// MarkProjectDeleted records that a project made from a quotation was
	// deleted permanently ("Proyek sudah dihapus"); the quotation stays
	// converted. A project made without a quotation is a no-op.
	MarkProjectDeleted(ctx context.Context, tenantID, projectID int64) error
}

// The frozen content of a revision, re-exported so projects never imports
// offers/domain (ADR-0049). Positions (VenueIndex, ParentIndex, EventIndex,
// SupplierIndex, DueEventIndex) point into the same content's slices.
type (
	QuotationContent  = domain.Content
	SnapshotEvent     = domain.SnapshotEvent
	SnapshotSession   = domain.SnapshotSession
	SnapshotSupplier  = domain.SnapshotSupplier
	SnapshotLine      = domain.SnapshotLine
	SnapshotTerm      = domain.SnapshotTerm
	SupplierKind      = domain.SupplierKind
	Payer             = domain.Payer
	DueBasis          = domain.DueBasis
	QuotationIdentity = domain.WorkspaceIdentity
)

// Values of SupplierKind, Payer, and DueBasis.
const (
	SupplierVendor  = domain.SupplierVendor
	SupplierVenue   = domain.SupplierVenue
	PayerWO         = domain.PayerWO
	PayerClient     = domain.PayerClient
	DueOnAcceptance = domain.DueOnAcceptance
	DueOnEvent      = domain.DueOnEvent
)

// AcceptedQuotation is an accepted quotation as a project copies it.
type AcceptedQuotation struct {
	ID              int64
	Number          string
	CoupleNames     string
	ContactWhatsApp string
	PICStaffID      int64
	IsPKP           bool
	// AcceptedOn is the WIB day of acceptance (the base of "saat diterima" terms).
	AcceptedOn         *time.Time
	AcceptedChannel    string
	ConvertedProjectID *int64
	Revision           AcceptedRevision
	// Content is the accepted revision's content: events with sessions,
	// suppliers (paid by, agreed cost, show name, capacities), lines (package
	// rules included), terms with their computed amounts, terms text, totals.
	Content QuotationContent
}

// AcceptedRevision identifies the revision a project keeps as "Kesepakatan
// awal".
type AcceptedRevision struct {
	ID            int64
	N             int
	ContractTotal int64
	CreatedAt     time.Time
}

type quotationsImpl struct {
	svc *application.QuotationService
}

// NewQuotations wraps the quotation service as the cross-module contract.
func NewQuotations(svc *application.QuotationService) Quotations { return &quotationsImpl{svc: svc} }

func (c *quotationsImpl) AcceptedQuotation(ctx context.Context, tenantID, quotationID int64) (AcceptedQuotation, error) {
	q, rev, err := c.svc.AcceptedQuotation(ctx, tenantID, quotationID)
	if err != nil {
		return AcceptedQuotation{}, err
	}
	out := AcceptedQuotation{
		ID: q.ID, Number: q.Number, CoupleNames: rev.Snapshot.Content.CoupleNames, ContactWhatsApp: rev.Snapshot.Content.ContactWhatsApp,
		PICStaffID: rev.Snapshot.Content.PICStaffID, IsPKP: rev.Snapshot.Content.IsPKP, AcceptedOn: q.AcceptedOn,
		ConvertedProjectID: q.ConvertedProjectID,
		Revision:           AcceptedRevision{ID: rev.ID, N: rev.N, ContractTotal: rev.ContractTotal, CreatedAt: rev.CreatedAt},
		Content:            rev.Snapshot.Content,
	}
	if q.AcceptedChannel != nil {
		out.AcceptedChannel = string(*q.AcceptedChannel)
	}
	return out, nil
}

func (c *quotationsImpl) MarkConverted(ctx context.Context, tenantID, quotationID, projectID int64) error {
	return c.svc.MarkConverted(ctx, tenantID, quotationID, projectID)
}

func (c *quotationsImpl) MarkProjectDeleted(ctx context.Context, tenantID, projectID int64) error {
	return c.svc.MarkProjectDeleted(ctx, tenantID, projectID)
}
