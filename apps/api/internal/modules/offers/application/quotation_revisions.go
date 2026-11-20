package application

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"sort"
	"time"

	"elproof/internal/modules/offers/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/compress"
)

// ErrDuplicateRevision is what the repository answers when a revision with
// the same content already exists (UNIQUE (tenant_id, quotation_id,
// content_hash)): two "Buat revisi" at once.
var ErrDuplicateRevision = errors.New("quotation revision: same content exists")

// Messages of revisions and PDFs.
const (
	MsgRevisionNeedsLines  = "Tambahkan minimal satu baris rincian sebelum membuat revisi"
	MsgRevisionTermsUneven = "Jumlah termin belum sama dengan nilai kontrak. Sesuaikan termin dulu."
	MsgCompareInvalid      = "Pilih dua revisi untuk dibandingkan"
	MsgPDFFailed           = "Gagal membuat PDF penawaran"
	CodeRevisionNeedsLines = "revision_needs_lines"
	CodeTermsUneven        = "terms_uneven"
)

// collectCategoryIDs is every category the content names (lines and
// suppliers), for the names frozen in a snapshot.
func collectCategoryIDs(c domain.Content) []int64 {
	seen := map[int64]bool{}
	var ids []int64
	add := func(id *int64) {
		if id != nil && !seen[*id] {
			seen[*id] = true
			ids = append(ids, *id)
		}
	}
	for _, l := range c.Lines {
		add(l.CategoryID)
	}
	for _, s := range c.Suppliers {
		add(s.CategoryID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// snapshotOf wraps content with what the PDF prints around it: the WO's
// identity as it is now, the PIC's name, and the category names.
func (s *QuotationService) snapshotOf(ctx context.Context, tenantID int64, q *domain.Quotation, c domain.Content, n int, issuedAt time.Time) (domain.Snapshot, error) {
	p, err := s.quotationProfile(ctx, tenantID)
	if err != nil {
		return domain.Snapshot{}, err
	}
	snap := domain.Snapshot{
		SchemaVersion: domain.SnapshotSchemaVersion, Number: q.Number, RevisionN: n,
		IssuedAt: issuedAt.UTC().Truncate(time.Microsecond), ValidUntil: q.ValidUntil.Format("2006-01-02"),
		Workspace: p.Identity, CategoryNames: map[int64]string{}, Content: c,
	}
	if s.staff != nil {
		name, found, err := s.staff.GetName(ctx, tenantID, q.PICStaffID)
		if err != nil {
			return domain.Snapshot{}, err
		}
		if found {
			snap.PICName = name
		}
	}
	if ids := collectCategoryIDs(c); len(ids) > 0 && s.directory != nil {
		names, err := s.directory.CategoryNames(ctx, tenantID, ids)
		if err != nil {
			return domain.Snapshot{}, err
		}
		for id, name := range names {
			snap.CategoryNames[id] = name
		}
	}
	return snap, nil
}

// contentOf freezes a working copy, refusing what a client should never get.
func contentOf(w *domain.WorkingCopy) (domain.Content, error) {
	c, err := domain.BuildContent(*w)
	if err != nil {
		return domain.Content{}, linesErr(err, 0)
	}
	return c, nil
}

// CreateRevision freezes the working copy into a revision ("Buat revisi untuk
// klien"). The same content answers the revision that already has it
// (created=false), so two clicks at once give one revision: the quotation row
// is locked first, and the unique hash backs it up.
func (s *QuotationService) CreateRevision(ctx context.Context, tenantID, id int64, v Viewer) (*domain.QuotationRevision, bool, error) {
	var out *domain.QuotationRevision
	created := false
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		q, err := s.lock(ctx, tenantID, id, v)
		if err != nil {
			return err
		}
		if err := editable(q); err != nil {
			return err
		}
		w, err := s.workingCopy(ctx, tenantID, q)
		if err != nil {
			return err
		}
		if len(w.Lines) == 0 {
			return apperror.Validation(MsgRevisionNeedsLines, map[string][]string{"lines": {MsgRevisionNeedsLines}}).WithCode(CodeRevisionNeedsLines)
		}
		c, err := contentOf(w)
		if err != nil {
			return err
		}
		if _, mismatch := domain.TermAmounts(w.Terms, c.Totals.Total); mismatch {
			return apperror.Validation(MsgRevisionTermsUneven, map[string][]string{"terms": {MsgRevisionTermsUneven}}).WithCode(CodeTermsUneven)
		}
		hash := c.Hash()
		existing, err := s.repo.FindRevisionByHash(ctx, tenantID, q.ID, hash)
		if err != nil {
			return err
		}
		if existing != nil {
			out = existing
			return nil
		}
		n := q.RevisionCount + 1
		snap, err := s.snapshotOf(ctx, tenantID, q, c, n, s.now())
		if err != nil {
			return err
		}
		rev := &domain.QuotationRevision{QuotationID: q.ID, N: n, Snapshot: snap, ContentHash: hash,
			ContractTotal: c.Totals.Total, CreatedByStaffID: v.actor(), CreatedAt: snap.IssuedAt}
		if err := s.repo.InsertRevision(ctx, tenantID, rev); err != nil {
			if errors.Is(err, ErrDuplicateRevision) {
				existing, ferr := s.repo.FindRevisionByHash(ctx, tenantID, q.ID, hash)
				if ferr != nil {
					return ferr
				}
				if existing != nil {
					out = existing
					return nil
				}
			}
			return err
		}
		out, created = rev, true
		return s.activity(ctx, tenantID, q.ID, v, domain.ActivityRevisionCreated,
			fmt.Sprintf("Revisi %d dibuat · nilai kontrak %s", n, Rupiah(c.Totals.Total)))
	})
	if err != nil {
		return nil, false, err
	}
	return out, created, nil
}

// ListRevisions is every revision of the quotation, newest first.
func (s *QuotationService) ListRevisions(ctx context.Context, tenantID, id int64, v Viewer) ([]domain.QuotationRevision, error) {
	if _, err := s.get(ctx, tenantID, id, v); err != nil {
		return nil, err
	}
	return s.repo.ListRevisions(ctx, tenantID, id)
}

// GetRevision is revision n of the quotation.
func (s *QuotationService) GetRevision(ctx context.Context, tenantID, id int64, n int, v Viewer) (*domain.QuotationRevision, error) {
	if _, err := s.get(ctx, tenantID, id, v); err != nil {
		return nil, err
	}
	return s.revision(ctx, tenantID, id, n)
}

func (s *QuotationService) revision(ctx context.Context, tenantID, id int64, n int) (*domain.QuotationRevision, error) {
	rev, err := s.repo.FindRevision(ctx, tenantID, id, n)
	if err != nil {
		return nil, err
	}
	if rev == nil {
		return nil, apperror.NotFound(MsgRevisionNotFound)
	}
	return rev, nil
}

// ComparisonResult is "Bandingkan revisi": revision From against revision To
// (To = 0: the working copy as it is now).
type ComparisonResult struct {
	From, To   int
	Comparison domain.Comparison
}

// Compare compares two revisions, or a revision with the working copy.
func (s *QuotationService) Compare(ctx context.Context, tenantID, id int64, from, to int, v Viewer) (*ComparisonResult, error) {
	q, err := s.get(ctx, tenantID, id, v)
	if err != nil {
		return nil, err
	}
	if from < 1 || to < 0 || from == to {
		return nil, apperror.Validation(MsgCompareInvalid, map[string][]string{"from": {MsgCompareInvalid}})
	}
	before, err := s.revision(ctx, tenantID, id, from)
	if err != nil {
		return nil, err
	}
	var after domain.Content
	if to == 0 {
		w, err := s.workingCopy(ctx, tenantID, q)
		if err != nil {
			return nil, err
		}
		if after, err = contentOf(w); err != nil {
			return nil, err
		}
	} else {
		rev, err := s.revision(ctx, tenantID, id, to)
		if err != nil {
			return nil, err
		}
		after = rev.Snapshot.Content
	}
	return &ComparisonResult{From: from, To: to, Comparison: domain.Compare(before.Snapshot.Content, after)}, nil
}

// ShareRevision records that revision n was sent to the client ("Bagikan
// PDF"); the sending itself happens on the person's phone.
func (s *QuotationService) ShareRevision(ctx context.Context, tenantID, id int64, n int, v Viewer) error {
	if _, err := s.get(ctx, tenantID, id, v); err != nil {
		return err
	}
	if _, err := s.revision(ctx, tenantID, id, n); err != nil {
		return err
	}
	return s.activity(ctx, tenantID, id, v, domain.ActivityShared, fmt.Sprintf("Revisi %d dibagikan ke klien", n))
}

// RevisionPDF renders revision n again from its snapshot, so the PDF of a
// revision stays the same however the quotation changes afterwards.
func (s *QuotationService) RevisionPDF(ctx context.Context, tenantID, id int64, n int, v Viewer) ([]byte, string, error) {
	if _, err := s.get(ctx, tenantID, id, v); err != nil {
		return nil, "", err
	}
	rev, err := s.revision(ctx, tenantID, id, n)
	if err != nil {
		return nil, "", err
	}
	data, err := s.render(ctx, rev.Snapshot, false, fmt.Sprintf("Revisi %d", rev.N))
	if err != nil {
		return nil, "", err
	}
	return data, fmt.Sprintf("%s-Rev%d.pdf", rev.Snapshot.Number, rev.N), nil
}

// PreviewPDF renders the working copy with the "DRAF" watermark. It creates
// no revision and records nothing.
func (s *QuotationService) PreviewPDF(ctx context.Context, tenantID, id int64, v Viewer) ([]byte, string, error) {
	q, err := s.get(ctx, tenantID, id, v)
	if err != nil {
		return nil, "", err
	}
	w, err := s.workingCopy(ctx, tenantID, q)
	if err != nil {
		return nil, "", err
	}
	c, err := contentOf(w)
	if err != nil {
		return nil, "", err
	}
	snap, err := s.snapshotOf(ctx, tenantID, q, c, 0, s.now())
	if err != nil {
		return nil, "", err
	}
	data, err := s.render(ctx, snap, true, "Pratinjau")
	if err != nil {
		return nil, "", err
	}
	return data, q.Number + "-Pratinjau.pdf", nil
}

func (s *QuotationService) render(ctx context.Context, snap domain.Snapshot, draft bool, label string) ([]byte, error) {
	if s.renderer == nil {
		return nil, apperror.Internal(MsgPDFFailed)
	}
	logo, logoType := s.logo(ctx, snap.Workspace.LogoKey)
	data, err := s.renderer.Render(BuildDocument(snap, draft, label, logo, logoType))
	if err != nil {
		return nil, apperror.Internal(MsgPDFFailed)
	}
	return data, nil
}

// maxLogoBytes bounds the logo read from storage (the upload allows 2 MB).
const maxLogoBytes = 4 << 20

// logo reads the WO's logo and re-encodes it as PNG or JPEG for the PDF
// ("PNG"/"JPG"). A logo that cannot be read leaves the letterhead without one
// rather than failing the document.
func (s *QuotationService) logo(ctx context.Context, key *string) ([]byte, string) {
	if key == nil || *key == "" || s.files == nil {
		return nil, ""
	}
	r, err := s.files.Open(ctx, *key)
	if err != nil {
		return nil, ""
	}
	defer r.Close()
	raw, err := io.ReadAll(io.LimitReader(r, maxLogoBytes+1))
	if err != nil || len(raw) > maxLogoBytes {
		return nil, ""
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, ""
	}
	data, mime, err := compress.Photo(raw, "image/"+format)
	if err != nil {
		return nil, ""
	}
	if mime == "image/png" {
		return data, "PNG"
	}
	return data, "JPG"
}

// ---- detail ----------------------------------------------------------------

// QuotationDetail is the quotation page: the header, the content, what it
// adds up to, how it stands against its revisions, and its group.
type QuotationDetail struct {
	Quotation   domain.Quotation
	Content     *domain.WorkingCopy
	Totals      domain.Totals
	TotalsError error
	TermAmounts []int64
	// TermsMismatch: the terms do not add up to the contract value.
	TermsMismatch bool
	// MatchingRevisionN is the revision equal to the working copy (nil: the
	// working copy has changes no revision has).
	MatchingRevisionN *int
	LatestRevisionN   int
	// CatalogChanged marks package lines whose package changed in the catalog
	// after it was copied (line id → true).
	CatalogChanged map[int64]bool
	// Group is the quotation's group of alternatives (itself included).
	Group             []domain.Quotation
	AcceptedRevisionN *int
}

// Detail is the quotation page's data.
func (s *QuotationService) Detail(ctx context.Context, tenantID, id int64, v Viewer) (*QuotationDetail, error) {
	q, err := s.get(ctx, tenantID, id, v)
	if err != nil {
		return nil, err
	}
	w, err := s.workingCopy(ctx, tenantID, q)
	if err != nil {
		return nil, err
	}
	d := &QuotationDetail{Quotation: *q, Content: w, LatestRevisionN: q.RevisionCount, CatalogChanged: map[int64]bool{}}
	d.Totals, d.TotalsError = domain.TotalsOf(w.Lines, q.IsPKP)
	d.TermAmounts, d.TermsMismatch = domain.TermAmounts(w.Terms, d.Totals.Total)
	if d.TotalsError == nil && q.RevisionCount > 0 {
		if c, err := domain.BuildContent(*w); err == nil {
			rev, err := s.repo.FindRevisionByHash(ctx, tenantID, q.ID, c.Hash())
			if err != nil {
				return nil, err
			}
			if rev != nil {
				n := rev.N
				d.MatchingRevisionN = &n
			}
		}
	}
	if q.AcceptedRevisionID != nil {
		if rev, err := s.repo.FindRevisionByID(ctx, tenantID, *q.AcceptedRevisionID); err != nil {
			return nil, err
		} else if rev != nil {
			n := rev.N
			d.AcceptedRevisionN = &n
		}
	}
	for _, l := range w.Lines {
		if l.Kind != domain.LinePackage || l.PackageID == nil || l.CopiedAt == nil {
			continue
		}
		p, err := s.packages.FindByID(ctx, tenantID, *l.PackageID)
		if err != nil {
			if isNotFound(err) {
				continue
			}
			return nil, err
		}
		if p != nil && p.UpdatedAt.After(*l.CopiedAt) {
			d.CatalogChanged[l.ID] = true
		}
	}
	if d.Group, err = s.group(ctx, tenantID, q); err != nil {
		return nil, err
	}
	return d, nil
}
