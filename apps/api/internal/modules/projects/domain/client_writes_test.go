package domain

import (
	"testing"
	"time"
)

func TestPaymentSource(t *testing.T) {
	if (ClientPayment{RecordedByStaffID: 3}).Source() != SourceStaff || (ClientPayment{}).Source() != SourceStaff {
		t.Error("a payment staff recorded (or of unknown origin) is staff's")
	}
	if (ClientPayment{SubmittedByClientID: 9}).Source() != SourceClient {
		t.Error("a payment the couple sent is the client's")
	}
}

func TestSubmissionType(t *testing.T) {
	cases := []struct {
		applied, contract, amount int64
		want                      PaymentType
	}{
		{0, 100, 30, PaymentDP},
		{-10, 100, 30, PaymentDP}, // only a refund so far
		{30, 100, 20, PaymentTermin},
		{30, 100, 70, PaymentPelunasan},
		{30, 0, 70, PaymentTermin}, // no contract value yet: never "settles"
		{30, 100, 90, PaymentTermin},
	}
	for _, c := range cases {
		if got := SubmissionType(c.applied, c.contract, c.amount); got != c.want {
			t.Errorf("applied %d contract %d amount %d: got %s, want %s", c.applied, c.contract, c.amount, got, c.want)
		}
	}
}

func TestCompleteByClient(t *testing.T) {
	today := time.Date(2026, 11, 16, 23, 30, 0, 0, time.FixedZone("WIB", 7*3600))
	m := ProjectMilestone{ForClient: true, Status: MilestoneInProgress}
	changed, err := m.CompleteByClient(7, today)
	if err != nil || !changed || m.Status != MilestoneCompleted || m.CompletedByClientID != 7 || m.CompletedDate.Format("2006-01-02") != "2026-11-16" {
		t.Fatalf("completed on the Jakarta day: %v %v %+v", changed, err, m)
	}
	changed, err = m.CompleteByClient(8, today.AddDate(0, 0, 1))
	if err != nil || changed || m.CompletedByClientID != 7 {
		t.Errorf("idempotent: %v %v %+v", changed, err, m)
	}
	staff := ProjectMilestone{Status: MilestoneNotStarted}
	if _, err := staff.CompleteByClient(7, today); err != ErrNotClientTask || staff.Status != MilestoneNotStarted {
		t.Errorf("a staff step: %v", err)
	}
	cancelled := ProjectMilestone{ForClient: true, Status: MilestoneCancelled}
	if _, err := cancelled.CompleteByClient(7, today); err != ErrClientTaskCancelled || cancelled.Status != MilestoneCancelled {
		t.Errorf("a cancelled task: %v", err)
	}
	// A step staff already completed is "already completed" for the couple too.
	done := ProjectMilestone{ForClient: true, Status: MilestoneCompleted}
	if changed, err := done.CompleteByClient(7, today); err != nil || changed || done.CompletedByClientID != 0 {
		t.Errorf("completed by staff: %v %v", changed, err)
	}
}

func TestSelectClientTasks(t *testing.T) {
	d := func(s string) time.Time { v, _ := time.Parse("2006-01-02", s); return v }
	dp := func(s string) *time.Time { v := d(s); return &v }
	var in []ProjectMilestone
	// Open tasks out of order, one staff step, one cancelled task.
	in = append(in,
		ProjectMilestone{ID: 1, SortOrder: 3, Name: "Pilih menu", Status: MilestoneNotStarted, TargetDate: d("2027-02-01"), ForClient: true},
		ProjectMilestone{ID: 2, SortOrder: 1, Name: "Daftar tamu", Status: MilestoneInProgress, TargetDate: d("2027-01-10"), ForClient: true},
		ProjectMilestone{ID: 3, SortOrder: 2, Name: "Survei venue", Status: MilestoneNotStarted, TargetDate: d("2027-01-01")},
		ProjectMilestone{ID: 4, SortOrder: 4, Name: "Batal", Status: MilestoneCancelled, TargetDate: d("2027-01-01"), ForClient: true},
		ProjectMilestone{ID: 5, SortOrder: 0, Name: "Kirim foto", Status: MilestoneBlocked, TargetDate: d("2027-01-10"), ForClient: true},
	)
	// Seven completed tasks: only the five most recent stay.
	for i := 0; i < 7; i++ {
		in = append(in, ProjectMilestone{ID: int64(10 + i), Name: "Selesai", Status: MilestoneCompleted, ForClient: true,
			TargetDate: d("2026-10-01"), CompletedDate: dp(time.Date(2026, 10, 1+i, 0, 0, 0, 0, time.UTC).Format("2006-01-02"))})
	}
	got := SelectClientTasks(in)
	var ids []int64
	for _, m := range got {
		ids = append(ids, m.ID)
	}
	want := []int64{5, 2, 1, 16, 15, 14, 13, 12}
	if len(ids) != len(want) {
		t.Fatalf("got %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("got %v, want %v", ids, want)
		}
	}
	if len(SelectClientTasks(nil)) != 0 {
		t.Error("no steps, no tasks")
	}
}

func TestPortalWhatsApp(t *testing.T) {
	cases := []struct{ pic, business, want string }{
		{"0812-3456-7890", "6281199998888", "0812-3456-7890"}, // the PIC's phone comes first, as typed
		{" +62 812 3456 7890 ", "", "+62 812 3456 7890"},
		{"8123456789", "", "8123456789"},
		{"", "6281199998888", "6281199998888"}, // then the business WhatsApp
		{"-", "081199998888", "081199998888"},  // a placeholder does not hide it
		{"0812", "081199998888", "081199998888"},
		{"0212345678901234567", "081199998888", "081199998888"}, // too long
		{"6201234567890", "081199998888", "081199998888"},       // 62 then 0
		{"", " ", ""}, // neither: empty (the contact section is absent)
		{"abc", "x", ""},
	}
	for _, c := range cases {
		if got := PortalWhatsApp(c.pic, c.business); got != c.want {
			t.Errorf("PortalWhatsApp(%q, %q) = %q, want %q", c.pic, c.business, got, c.want)
		}
	}
}
