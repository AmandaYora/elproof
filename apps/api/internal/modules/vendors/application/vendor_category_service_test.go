package application

import (
	"context"
	"strings"
	"testing"

	"elproof/internal/modules/vendors/domain"
)

// fakeCategoryRepo keeps categories in memory; methods the service does not
// call here panic through the nil embedded interface.
type fakeCategoryRepo struct {
	VendorCategoryRepository
	rows []domain.VendorCategory
}

func (f *fakeCategoryRepo) List(context.Context, int64) ([]domain.VendorCategory, error) {
	return f.rows, nil
}
func (f *fakeCategoryRepo) FindByID(_ context.Context, _, id int64) (*domain.VendorCategory, error) {
	for _, c := range f.rows {
		if c.ID == id {
			c := c
			return &c, nil
		}
	}
	return nil, nil
}
func (f *fakeCategoryRepo) Create(_ context.Context, c *domain.VendorCategory) error {
	c.ID = int64(len(f.rows) + 1)
	f.rows = append(f.rows, *c)
	return nil
}
func (f *fakeCategoryRepo) Update(_ context.Context, c *domain.VendorCategory) error {
	for i := range f.rows {
		if f.rows[i].ID == c.ID {
			f.rows[i] = *c
		}
	}
	return nil
}

// FQ-03: a category name is unique per workspace, without regard to case or
// surrounding spaces, on create and on rename.
func TestCategoryNameIsUniquePerTenant(t *testing.T) {
	ctx := context.Background()
	repo := &fakeCategoryRepo{rows: []domain.VendorCategory{{ID: 1, Name: "Katering"}, {ID: 2, Name: "Dekorasi "}}}
	svc := NewVendorCategoryService(repo, nil)

	for _, name := range []string{"Katering", "  katering ", "KATERING", "dekorasi"} {
		_, err := svc.Create(ctx, 1, VendorCategoryInput{Name: name, Description: "d"})
		f := fieldsOf(t, err)
		if len(f["name"]) != 1 || f["name"][0] != MsgCategoryNameTaken {
			t.Errorf("create %q: %v", name, f)
		}
	}
	c, err := svc.Create(ctx, 1, VendorCategoryInput{Name: "  Bunga Segar ", Description: "d"})
	if err != nil || c.Name != "Bunga Segar" {
		t.Fatalf("new name refused or not trimmed: %+v %v", c, err)
	}

	// Rename onto another category's name is refused; changing only the case
	// (or the description) of its own name is fine.
	if f := fieldsOf(t, func() error {
		_, err := svc.Update(ctx, 1, 3, VendorCategoryInput{Name: "katering", Description: "d"})
		return err
	}()); f["name"][0] != MsgCategoryNameTaken {
		t.Errorf("rename onto Katering: %v", f)
	}
	if _, err := svc.Update(ctx, 1, 1, VendorCategoryInput{Name: "KATERING", Description: "baru"}); err != nil {
		t.Errorf("own name in another case refused: %v", err)
	}

	if f := fieldsOf(t, func() error { _, err := svc.Create(ctx, 1, VendorCategoryInput{Name: "   "}); return err }()); len(f["name"]) == 0 {
		t.Errorf("blank name: %v", f)
	}
	if f := fieldsOf(t, func() error {
		_, err := svc.Create(ctx, 1, VendorCategoryInput{Name: strings.Repeat("a", 101), Description: "d"})
		return err
	}()); len(f["name"]) == 0 {
		t.Errorf("long name: %v", f)
	}
}
