package application

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"elproof/internal/modules/offers/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/etag"
)

// EventKindView is an event kind with how many packages cover it and how
// many quotations have an event of it ("Dipakai di").
type EventKindView struct {
	domain.EventKind
	PackageCount   int
	QuotationCount int
}

// EventKindService manages the WO's list of event kinds (Katalog › Jenis acara).
type EventKindService struct {
	repo  EventKindRepository
	usage QuotationUsage
}

func NewEventKindService(repo EventKindRepository) *EventKindService {
	return &EventKindService{repo: repo}
}

// Messages answered by the event kind service.
const (
	MsgEventKindNotFound = "Jenis acara tidak ditemukan"
	MsgEventKindName     = "Isi nama jenis acara"
	MsgEventKindTaken    = "Jenis acara dengan nama ini sudah ada"
	CodeEventKindInUse   = "event_kind_in_use"
)

// List is every kind of the tenant in display order, with usage counts.
func (s *EventKindService) List(ctx context.Context, tenantID int64) ([]EventKindView, error) {
	kinds, err := s.repo.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	counts, err := s.repo.PackageCounts(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	qcounts, err := s.quotationCounts(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]EventKindView, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, EventKindView{EventKind: k, PackageCount: counts[k.ID], QuotationCount: qcounts[k.ID]})
	}
	return out, nil
}

func (s *EventKindService) Get(ctx context.Context, tenantID, id int64) (*domain.EventKind, error) {
	k, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if k == nil {
		return nil, apperror.NotFound(MsgEventKindNotFound)
	}
	return k, nil
}

// view is one kind with its usage count (for a write's response).
func (s *EventKindService) view(ctx context.Context, k *domain.EventKind) (*EventKindView, error) {
	counts, err := s.repo.PackageCounts(ctx, k.TenantID)
	if err != nil {
		return nil, err
	}
	qcounts, err := s.quotationCounts(ctx, k.TenantID)
	if err != nil {
		return nil, err
	}
	return &EventKindView{EventKind: *k, PackageCount: counts[k.ID], QuotationCount: qcounts[k.ID]}, nil
}

// checkName trims the name and refuses an empty, too long, or duplicate one
// (another kind of the tenant, compared without regard to case).
func (s *EventKindService) checkName(ctx context.Context, tenantID, selfID int64, name string) (string, error) {
	name = strings.TrimSpace(name)
	msg := ""
	switch {
	case name == "":
		msg = MsgEventKindName
	case utf8.RuneCountInString(name) > domain.MaxEventKindNameLen:
		msg = fmt.Sprintf("Nama jenis acara maksimal %d karakter", domain.MaxEventKindNameLen)
	default:
		kinds, err := s.repo.List(ctx, tenantID)
		if err != nil {
			return "", err
		}
		for _, k := range kinds {
			if k.ID != selfID && strings.EqualFold(strings.TrimSpace(k.Name), name) {
				msg = MsgEventKindTaken
				break
			}
		}
	}
	if msg != "" {
		return "", apperror.Validation(msg, map[string][]string{"name": {msg}})
	}
	return name, nil
}

func (s *EventKindService) Create(ctx context.Context, tenantID int64, name string) (*EventKindView, error) {
	name, err := s.checkName(ctx, tenantID, 0, name)
	if err != nil {
		return nil, err
	}
	kinds, err := s.repo.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	order := 0
	for _, k := range kinds {
		if k.SortOrder > order {
			order = k.SortOrder
		}
	}
	k := &domain.EventKind{TenantID: tenantID, Name: name, SortOrder: order + 1, IsActive: true}
	if err := s.repo.Create(ctx, k); err != nil {
		return nil, err
	}
	created, err := s.Get(ctx, tenantID, k.ID)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, created)
}

// Rename changes a kind's name (built-in kinds included). Events already in a
// quotation or project keep the name they copied. expected is the If-Match
// version.
func (s *EventKindService) Rename(ctx context.Context, tenantID, id int64, name string, expected *time.Time) (*EventKindView, error) {
	k, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if expected != nil && !etag.Same(k.UpdatedAt, *expected) {
		return nil, s.stale(ctx, k)
	}
	name, err = s.checkName(ctx, tenantID, id, name)
	if err != nil {
		return nil, err
	}
	ok, err := s.repo.Rename(ctx, tenantID, id, name, expected)
	if err != nil {
		return nil, err
	}
	current, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, s.stale(ctx, current)
	}
	return s.view(ctx, current)
}

func (s *EventKindService) stale(ctx context.Context, k *domain.EventKind) error {
	v, err := s.view(ctx, k)
	if err != nil {
		return err
	}
	return apperror.Stale(v)
}

// SetActive sets whether the kind can be picked for a new package or event.
func (s *EventKindService) SetActive(ctx context.Context, tenantID, id int64, isActive bool) (*EventKindView, error) {
	k, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if k.IsActive != isActive {
		if err := s.repo.SetActive(ctx, tenantID, id, isActive); err != nil {
			return nil, err
		}
		if k, err = s.Get(ctx, tenantID, id); err != nil {
			return nil, err
		}
	}
	return s.view(ctx, k)
}

// Delete removes a kind no package covers and no quotation uses; one in use
// is deactivated instead.
func (s *EventKindService) Delete(ctx context.Context, tenantID, id int64) error {
	k, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return err
	}
	counts, err := s.repo.PackageCounts(ctx, tenantID)
	if err != nil {
		return err
	}
	qcounts, err := s.quotationCounts(ctx, tenantID)
	if err != nil {
		return err
	}
	if n, qn := counts[k.ID], qcounts[k.ID]; n > 0 || qn > 0 {
		return apperror.Conflict(inUseMessage(k.Name, n, qn)).WithCode(CodeEventKindInUse)
	}
	return s.repo.Delete(ctx, tenantID, id)
}

// SeedDefaultEventKinds gives a workspace the built-in kinds it is still
// missing (matched by their key, so a renamed one is not added again). Called
// when a WO signs up; idempotent, like SeedDefaultCategories.
func (s *EventKindService) SeedDefaultEventKinds(ctx context.Context, tenantID int64) error {
	kinds, err := s.repo.List(ctx, tenantID)
	if err != nil {
		return err
	}
	haveKey := map[string]bool{}
	haveName := map[string]bool{}
	for _, k := range kinds {
		haveKey[k.DefaultKey] = true
		haveName[strings.ToLower(k.Name)] = true
	}
	for i, d := range domain.DefaultEventKinds {
		if haveKey[d.Key] || haveName[strings.ToLower(d.Name)] {
			continue
		}
		k := &domain.EventKind{TenantID: tenantID, Name: d.Name, DefaultKey: d.Key, SortOrder: i + 1, IsActive: true}
		if err := s.repo.Create(ctx, k); err != nil {
			return err
		}
	}
	return nil
}
