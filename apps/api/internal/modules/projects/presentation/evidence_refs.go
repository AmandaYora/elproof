package presentation

import (
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

// indexEvidence builds the index. For a client principal only files that
// domain.ClientCanSeeEvidence allows are embedded; staff get every file. Files
// are listed oldest first, so the first one is the original proof.
func indexEvidence(list []domain.Evidence, isClient bool, can func(string) bool) evidenceIndex {
	ix := evidenceIndex{visible: map[evidenceKey][]evidenceRefResponse{}, any: map[evidenceKey]bool{}}
	// The repository lists newest first; walk backwards for oldest first.
	for i := len(list) - 1; i >= 0; i-- {
		e := list[i]
		k := evidenceKey{e.RelatedKind, e.RelatedID}
		ix.any[k] = true
		if isClient && !domain.ClientCanSeeEvidence(e, can) {
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
