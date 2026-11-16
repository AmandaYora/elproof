package application

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"elproof/internal/shared/apperror"
)

// FQ-04: a Jadwal bawaan step is at most two years either side of the event;
// the service refuses before touching its repository (nil here: reaching it
// would panic).
func TestTemplateDaysAreBounded(t *testing.T) {
	svc := NewMilestoneTemplateService(nil)
	ctx := context.Background()
	for _, tc := range []struct {
		name  string
		in    MilestoneTemplateInput
		field string
		msg   string
	}{
		{"far before", MilestoneTemplateInput{Name: "Survey venue", DaysBeforeEvent: 99999}, "daysBeforeEvent", "Jumlah hari maksimal 730 (2 tahun) sebelum acara."},
		{"just past two years", MilestoneTemplateInput{Name: "Survey venue", DaysBeforeEvent: 731}, "daysBeforeEvent", "Jumlah hari maksimal 730 (2 tahun) sebelum acara."},
		{"far after", MilestoneTemplateInput{Name: "Kirim album", DaysBeforeEvent: -731}, "daysBeforeEvent", "Jumlah hari maksimal 730 (2 tahun) setelah acara."},
		{"short name", MilestoneTemplateInput{Name: " ab ", DaysBeforeEvent: 10}, "name", "Isi nama tahapan, minimal 3 karakter"},
		{"long name", MilestoneTemplateInput{Name: strings.Repeat("a", 256), DaysBeforeEvent: 10}, "name", "Nama tahapan maksimal 255 karakter"},
	} {
		for _, call := range []func() error{
			func() error { _, err := svc.Create(ctx, 1, tc.in); return err },
			func() error { _, err := svc.Update(ctx, 1, 2, tc.in); return err },
		} {
			err := call()
			if apperror.HTTPStatus(err) != http.StatusUnprocessableEntity {
				t.Fatalf("%s: want 422, got %v", tc.name, err)
			}
			ae, _ := apperror.As(err)
			if got := ae.Fields[tc.field]; len(got) != 1 || got[0] != tc.msg || ae.Message != tc.msg {
				t.Errorf("%s: %q %v", tc.name, ae.Message, ae.Fields)
			}
		}
	}
	for _, days := range []int{0, 730, -730, 45} {
		in := MilestoneTemplateInput{Name: "  Fitting busana ", DaysBeforeEvent: days}
		if err := validateTemplate(&in); err != nil || in.Name != "Fitting busana" {
			t.Errorf("%d days refused or name not trimmed: %v %q", days, err, in.Name)
		}
	}
	for _, seed := range defaultMilestoneTemplateSeed {
		in := MilestoneTemplateInput{Name: seed.Name, DaysBeforeEvent: seed.DaysBeforeEvent}
		if err := validateTemplate(&in); err != nil {
			t.Errorf("seeded step %q is refused by its own rule: %v", seed.Name, err)
		}
	}
}
