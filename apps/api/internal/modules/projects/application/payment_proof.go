package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
)

// Transactor runs work inside one database transaction. It is local to the
// projects module: a transaction never spans modules (ADR-0033), and everything
// the payment flows touch (payments, evidence, activity, the project row) is
// owned by this module.
type Transactor interface {
	// InTx commits when fn returns nil and rolls back on an error. A nested call
	// joins the open transaction.
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
	// LockProject serialises writers of one project's money for the rest of the
	// transaction (a no-op outside one).
	LockProject(ctx context.Context, projectID int64) error
}

// noTransactor runs fn directly: for tests and for a service wired without a
// database transaction.
type noTransactor struct{}

func (noTransactor) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
func (noTransactor) LockProject(context.Context, int64) error { return nil }

// ProofSupport is what the three payment services share to record a payment
// together with its proof in one transaction.
type ProofSupport struct {
	tx       Transactor
	uploads  *UploadService
	evidence *EvidenceService
}

func NewProofSupport(tx Transactor, uploads *UploadService, evidence *EvidenceService) *ProofSupport {
	if tx == nil {
		tx = noTransactor{}
	}
	return &ProofSupport{tx: tx, uploads: uploads, evidence: evidence}
}

// resolve verifies the claimed uploads (nothing is written yet).
func (p *ProofSupport) resolve(ctx context.Context, tenantID, projectID int64, items []ProofItem, allowed []domain.EvidenceType) ([]ResolvedProof, error) {
	if len(items) == 0 {
		return nil, nil
	}
	return p.uploads.Resolve(ctx, tenantID, projectID, items, allowed)
}

// proofTarget is the payment a set of proof files is attached to.
type proofTarget struct {
	ProjectID    int64
	ActorStaffID int64
	Kind         domain.EvidenceRelatedKind
	RelatedID    int64
	Date         time.Time
	Reference    string
}

// defaultProofName is what the two-call flow named a proof when the person did
// not: "Bukti Transfer - <ref>" or "Invoice - <ref>".
func defaultProofName(t domain.EvidenceType, reference string) string {
	base := "Bukti Transfer"
	if t == domain.EvidenceInvoice {
		base = "Invoice"
	}
	if reference != "" {
		base += " - " + reference
	}
	if r := []rune(base); len(r) > 255 {
		base = string(r[:255])
	}
	return base
}

// attach inserts one evidence row per resolved upload, on the transaction in
// ctx. A token already claimed by another row (the unique index on the stored
// path) is reported on its own item; all such problems are collected, then the
// caller's transaction is rolled back by returning the error.
func (p *ProofSupport) attach(ctx context.Context, t proofTarget, proofs []ResolvedProof) ([]domain.Evidence, error) {
	if len(proofs) == 0 {
		return nil, nil
	}
	fe := newFieldErrors()
	out := make([]domain.Evidence, 0, len(proofs))
	for i, pr := range proofs {
		name := pr.Name
		if name == "" {
			name = defaultProofName(pr.Type, t.Reference)
		}
		date := t.Date
		e := domain.Evidence{
			ProjectID: t.ProjectID, Name: name, Type: pr.Type, StoragePath: pr.Claims.Key, FileName: pr.Claims.FileName,
			DocumentDate: &date, UploadedAt: time.Now(), UploadedByStaffID: t.ActorStaffID,
			RelatedKind: t.Kind, RelatedID: t.RelatedID,
		}
		if err := p.evidence.repo.Create(ctx, &e); err != nil {
			if err == domain.ErrStoragePathTaken {
				item, token := proofKeys(i)
				fe.add("Unggahan ini sudah dipakai oleh pembayaran lain. Unggah ulang berkasnya.", item, token)
				continue
			}
			return nil, err
		}
		p.evidence.activity.Record(ctx, &t.ProjectID, domain.ActivityEvidenceUploaded, t.ActorStaffID, "evidence", formatID(e.ID), e.Name,
			"Evidence diunggah: "+e.Name)
		out = append(out, e)
	}
	if !fe.empty() {
		return nil, fe.err()
	}
	return out, nil
}

// evidenceTypes is shorthand for the allowed-type lists.
var (
	clientProofTypes   = []domain.EvidenceType{domain.EvidenceTransferProof}
	outgoingProofTypes = []domain.EvidenceType{domain.EvidenceInvoice, domain.EvidenceTransferProof}
)

func lineOf(t domain.PaymentType, amount int64) domain.PaymentLine {
	return domain.PaymentLine{Type: t, Amount: amount}
}

// inLockedTx runs fn in one transaction that holds the project's write lock, so
// a delete or a restore is checked against the same payments every other money
// write of the project sees. Without transaction support (unit tests, a service
// wired without a database) it simply runs fn.
func inLockedTx(ctx context.Context, proof *ProofSupport, projectID int64, fn func(ctx context.Context) error) error {
	var tx Transactor = noTransactor{}
	if proof != nil && proof.tx != nil {
		tx = proof.tx
	}
	return tx.InTx(ctx, func(ctx context.Context) error {
		if err := tx.LockProject(ctx, projectID); err != nil {
			return err
		}
		return fn(ctx)
	})
}
