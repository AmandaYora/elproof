package presentation

import (
	"context"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
)

// proofBody is one file a payment request attaches: the token POST
// /projects/{id}/uploads returned, what kind of document it is, and an optional
// display name (default "Bukti Transfer - <reference>" / "Invoice - <reference>").
type proofBody struct {
	UploadToken string `json:"uploadToken"`
	Type        string `json:"type"`
	Name        string `json:"name"`
}

func toProofItems(list []proofBody) []application.ProofItem {
	out := make([]application.ProofItem, 0, len(list))
	for _, p := range list {
		out = append(out, application.ProofItem{UploadToken: p.UploadToken, Type: domain.EvidenceType(p.Type), Name: p.Name})
	}
	return out
}

// evidenceKey identifies the item a proof file hangs off.
type evidenceKey struct {
	kind domain.EvidenceRelatedKind
	id   int64
}

// evidenceIndex groups a project's evidence rows by the item they are attached
// to, so a list response can embed each item's files from ONE query.
type evidenceIndex struct {
	visible map[evidenceKey][]evidenceRefResponse
	// any is true for every item with at least one file, whether or not the
	// caller may see it (evidenceComplete is a fact about the item, not the viewer).
	any map[evidenceKey]bool
}

// portalView is who may see which file of a project. For a client principal it
// is the portal policy for their permissions (PLAN §4.4); for staff without
// finance.margin.view (staff=true) it only hides the files of payments the WO
// makes to its suppliers. nil stands for staff who see everything.
type portalView struct {
	staff  bool
	policy domain.PortalPolicy
	owners domain.EvidenceOwners
}

// canSee is the per-file rule.
// forClient: the view is the Client Portal's (staff without margin see every
// field, only the WO's supplier payments and their files are hidden from them).
func (v *portalView) forClient() bool { return v != nil && !v.staff }

func (v *portalView) canSee(e domain.Evidence) bool {
	switch {
	case v == nil:
		return true
	case v.staff:
		return e.RelatedKind != domain.RelatedSupplierPayment || v.owners.PaymentPayer[e.RelatedID] != domain.PayerWO
	}
	return v.policy.CanSee(e, v.owners)
}

// portalViewFor builds the caller's view; nil for staff who see every file.
func (h *Handler) portalViewFor(ctx context.Context, claims staffClaims, projectID int64) (*portalView, error) {
	if !claims.isClient && claims.can("finance.margin.view") {
		return nil, nil
	}
	owners, err := h.suppliers.EvidenceOwners(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if !claims.isClient {
		return &portalView{staff: true, owners: owners}, nil
	}
	return &portalView{policy: domain.NewPortalPolicy(claims.can), owners: owners}, nil
}

// indexEvidence builds the index: only the files the view allows are embedded. Files are listed
// oldest first, so the first one is the original proof.
func indexEvidence(list []domain.Evidence, view *portalView) evidenceIndex {
	ix := evidenceIndex{visible: map[evidenceKey][]evidenceRefResponse{}, any: map[evidenceKey]bool{}}
	// The repository lists newest first; walk backwards for oldest first.
	for i := len(list) - 1; i >= 0; i-- {
		e := list[i]
		k := evidenceKey{e.RelatedKind, e.RelatedID}
		ix.any[k] = true
		if !view.canSee(e) {
			continue
		}
		ix.visible[k] = append(ix.visible[k], toEvidenceRef(e))
	}
	return ix
}

// refs returns the embeddable files of one item (never nil, so JSON is `[]`).
func (ix evidenceIndex) refs(kind domain.EvidenceRelatedKind, id int64) []evidenceRefResponse {
	if r := ix.visible[evidenceKey{kind, id}]; r != nil {
		return r
	}
	return []evidenceRefResponse{}
}

func (ix evidenceIndex) has(kind domain.EvidenceRelatedKind, id int64) bool {
	return ix.any[evidenceKey{kind, id}]
}
