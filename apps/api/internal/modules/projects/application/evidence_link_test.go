package application

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

type linkRepo struct {
	EvidenceRepository
	byProject map[int64]map[int64]*domain.Evidence
}

func (r linkRepo) FindByID(_ context.Context, projectID, id int64) (*domain.Evidence, error) {
	return r.byProject[projectID][id], nil
}

// plainStorage cannot presign (the case of a storage that is not the S3 client).
type plainStorage struct{}

func (plainStorage) Save(context.Context, string, []byte, string) (string, error) { return "", nil }
func (plainStorage) Open(context.Context, string) (io.ReadCloser, error)          { return nil, nil }
func (plainStorage) Delete(context.Context, string) error                         { return nil }

type presigningStorage struct {
	plainStorage
	enabled bool
	gotKey  string
	gotAtt  bool
	gotTTL  time.Duration
}

func (p *presigningStorage) CanPresign() bool { return p.enabled }
func (p *presigningStorage) PresignGet(_ context.Context, key, _ string, attachment bool, ttl time.Duration) (string, error) {
	p.gotKey, p.gotAtt, p.gotTTL = key, attachment, ttl
	return "https://files.example/" + key + "?sig=1", nil
}

func evidenceFixture() linkRepo {
	return linkRepo{byProject: map[int64]map[int64]*domain.Evidence{
		5: {7: {ID: 7, ProjectID: 5, StoragePath: "elproof/upload/1/5/evidence/a.pdf", FileName: "a.pdf"}},
	}}
}

func TestEvidenceLinkIsScopedToTheProject(t *testing.T) {
	store := &presigningStorage{enabled: true}
	s := &EvidenceService{repo: evidenceFixture(), storage: store}

	e, url, expires, err := s.Link(context.Background(), 5, 7, true)
	if err != nil || e == nil || url == "" {
		t.Fatalf("own project: %v %v %q", e, err, url)
	}
	if store.gotKey != "elproof/upload/1/5/evidence/a.pdf" || !store.gotAtt || store.gotTTL != EvidenceLinkTTL {
		t.Errorf("presigned the wrong thing: key=%q attachment=%v ttl=%v", store.gotKey, store.gotAtt, store.gotTTL)
	}
	if time.Until(expires) > EvidenceLinkTTL || time.Until(expires) < EvidenceLinkTTL-5*time.Second {
		t.Errorf("expiry %v does not match the TTL", expires)
	}

	// The same evidence id asked through another project is not found, and
	// nothing is presigned for it.
	store.gotKey = ""
	if _, _, _, err := s.Link(context.Background(), 6, 7, false); err == nil {
		t.Fatal("another project's id must not resolve")
	} else if ae, ok := apperror.As(err); !ok || ae.Kind != apperror.KindNotFound {
		t.Errorf("want not found, got %v", err)
	}
	if store.gotKey != "" {
		t.Error("a link was created for evidence outside the project")
	}
}

func TestEvidenceLinkFallsBackWhenUnavailable(t *testing.T) {
	for name, st := range map[string]ObjectStorage{
		"storage without presign": plainStorage{},
		"presign switched off":    &presigningStorage{enabled: false},
	} {
		s := &EvidenceService{repo: evidenceFixture(), storage: st}
		e, url, _, err := s.Link(context.Background(), 5, 7, false)
		if !errors.Is(err, ErrLinkUnavailable) || url != "" {
			t.Errorf("%s: want ErrLinkUnavailable, got err=%v url=%q", name, err, url)
		}
		// The evidence is still returned so the handler can judge visibility first.
		if e == nil {
			t.Errorf("%s: evidence must still come back for the visibility check", name)
		}
	}
}
