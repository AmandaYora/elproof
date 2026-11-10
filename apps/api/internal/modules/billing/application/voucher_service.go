package application

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/etag"
	"elproof/internal/shared/pagination"
)

// VoucherRepository is the storage of vouchers and their uses (this module's
// own tables only).
type VoucherRepository interface {
	ListPaginated(ctx context.Context, params pagination.Params, f domain.VoucherFilter) ([]domain.Voucher, int64, error)
	// FindByID / FindByCode return nil when there is no such voucher; both fill
	// UsedCount and ReservedCount.
	FindByID(ctx context.Context, id int64) (*domain.Voucher, error)
	FindByCode(ctx context.Context, code string) (*domain.Voucher, error)
	// CodeTaken reports whether another voucher (not exceptID) has this code.
	CodeTaken(ctx context.Context, code string, exceptID int64) (bool, error)
	Create(ctx context.Context, v *domain.Voucher) error
	// Update writes every editable field; with expected set it only writes when
	// updated_at is still that instant and answers false otherwise.
	Update(ctx context.Context, v *domain.Voucher, expected *time.Time) (bool, error)
	SetActive(ctx context.Context, id int64, active bool) error
	// CountRedemptions counts every use of the voucher, whatever its status.
	CountRedemptions(ctx context.Context, voucherID int64) (int64, error)
	ListRedemptions(ctx context.Context, voucherID int64, params pagination.Params) ([]domain.VoucherRedemption, int64, error)
	// CustomerUses counts the customer's uses that still count (reserved or used).
	CustomerUses(ctx context.Context, voucherID int64, c domain.Customer) (int64, error)
	// Reserve locks the voucher row of code (SELECT ... FOR UPDATE), re-reads
	// its counts and the customer's uses under that lock, asks decide for the
	// redemption to write and inserts it — all in one transaction, so two
	// charges at once can never both take the last unit of a quota or both
	// pass the once-per-customer rule. A nil voucher (unknown code) is passed
	// to decide as is. A duplicate order_ref is a Conflict.
	Reserve(ctx context.Context, code string, c domain.Customer,
		decide func(v *domain.Voucher, customerUses int64) (*domain.VoucherRedemption, error)) (*domain.VoucherRedemption, error)
	// DeleteReservation removes a reservation that never became a charge.
	DeleteReservation(ctx context.Context, id int64) error
	// Settle moves the reserved use of orderRef to status (used or released);
	// a use already settled is left alone.
	Settle(ctx context.Context, orderRef string, status domain.RedemptionStatus) error
	// ReleaseStale settles reservations older than olderThan from the ledgers:
	// one whose charge is paid becomes used; one whose charge no longer exists
	// as pending or paid (a process that died between reserving and recording
	// the charge, or a settle that failed) is released. Returns how many changed.
	ReleaseStale(ctx context.Context, olderThan time.Time) (int64, error)
}

// VoucherPlans is what vouchers need from the plan catalog (same module).
type VoucherPlans interface {
	FindByID(ctx context.Context, id int64) (*domain.Plan, error)
}

type VoucherService struct {
	repo    VoucherRepository
	plans   VoucherPlans
	tenants TenantDirectory
	now     func() time.Time
}

func NewVoucherService(repo VoucherRepository, plans VoucherPlans) *VoucherService {
	return &VoucherService{repo: repo, plans: plans, now: time.Now}
}

// SetTenants completes the two-phase wiring with `platform` (tenant names in
// the usage list).
func (s *VoucherService) SetTenants(d TenantDirectory) { s.tenants = d }

// wib is the calendar every voucher date is read in (the people using
// ElProof are in WIB); FixedZone keeps it working without tzdata.
var wib = time.FixedZone("WIB", 7*60*60)

// Today is the current WIB calendar day, "YYYY-MM-DD".
func (s *VoucherService) Today() string { return s.now().In(wib).Format("2006-01-02") }

// maxVoucherQuota keeps a quota inside its INT UNSIGNED column (a larger one would be a 500).
const maxVoucherQuota = 1_000_000_000

var voucherCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{2,31}$`)

// NormalizeVoucherCode is how every code is stored and looked up: trimmed and
// upper case, so "hemat10" and "HEMAT10" are the same voucher.
func NormalizeVoucherCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// VoucherInput is the editable part of a voucher.
type VoucherInput struct {
	Code            string
	AppID           string
	PlanID          *int64
	Type            domain.DiscountType
	Value           int64
	MaxDiscount     *int64
	ValidFrom       string
	ValidUntil      string
	Quota           *int64
	OncePerCustomer bool
}

func validDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// validate checks the input and the plan it names. The App itself is checked
// against the registry by the handler (as for plans, ADR-0040).
func (s *VoucherService) validate(ctx context.Context, in *VoucherInput) error {
	in.Code = NormalizeVoucherCode(in.Code)
	in.AppID = strings.TrimSpace(in.AppID)
	in.ValidFrom = strings.TrimSpace(in.ValidFrom)
	in.ValidUntil = strings.TrimSpace(in.ValidUntil)
	fields := map[string][]string{}
	if !voucherCodePattern.MatchString(in.Code) {
		fields["code"] = []string{"Kode 3–32 karakter: huruf, angka, tanda hubung (-) atau garis bawah (_)"}
	}
	if in.AppID == "" {
		fields["appId"] = []string{"Pilih aplikasi voucher ini"}
	}
	switch in.Type {
	case domain.DiscountPercent:
		if in.Value < 1 || in.Value > 99 {
			fields["discountValue"] = []string{"Potongan persen 1–99%"}
		}
		if in.MaxDiscount != nil && *in.MaxDiscount < 1 {
			fields["maxDiscount"] = []string{"Batas potongan minimal Rp1, atau kosongkan"}
		}
	case domain.DiscountAmount:
		if in.Value < 1 {
			fields["discountValue"] = []string{"Potongan minimal Rp1"}
		}
		in.MaxDiscount = nil // a cap only applies to a percentage
	default:
		fields["discountType"] = []string{"Pilih jenis potongan"}
	}
	if in.ValidFrom != "" && !validDate(in.ValidFrom) {
		fields["validFrom"] = []string{"Pilih tanggal dari kalender"}
	}
	if in.ValidUntil != "" && !validDate(in.ValidUntil) {
		fields["validUntil"] = []string{"Pilih tanggal dari kalender"}
	}
	if in.ValidFrom != "" && in.ValidUntil != "" && in.ValidUntil < in.ValidFrom {
		fields["validUntil"] = []string{"Tanggal akhir tidak boleh sebelum tanggal mulai"}
	}
	if in.Quota != nil && (*in.Quota < 1 || *in.Quota > maxVoucherQuota) {
		fields["quota"] = []string{"Kuota 1–1.000.000.000, atau kosongkan untuk tanpa batas"}
	}
	if in.PlanID != nil && *in.PlanID <= 0 {
		in.PlanID = nil
	}
	if in.PlanID != nil && in.AppID != "" {
		plan, err := s.plans.FindByID(ctx, *in.PlanID)
		if err != nil {
			return err
		}
		switch {
		case plan == nil || plan.AppID != in.AppID:
			fields["planId"] = []string{"Pilih paket milik aplikasi ini, atau kosongkan untuk semua paket"}
		case in.Type == domain.DiscountAmount && in.Value >= plan.Price:
			fields["discountValue"] = []string{"Potongan harus di bawah harga paket (" + formatRupiah(plan.Price) + "); harga akhir tidak boleh Rp0"}
		}
	}
	if len(fields) > 0 {
		return apperror.Validation("Data voucher belum benar", fields)
	}
	return nil
}

func (s *VoucherService) ensureCodeFree(ctx context.Context, code string, exceptID int64) error {
	taken, err := s.repo.CodeTaken(ctx, code, exceptID)
	if err != nil {
		return err
	}
	if taken {
		return apperror.Validation("Kode voucher sudah dipakai", map[string][]string{"code": {"Kode ini sudah dipakai voucher lain"}})
	}
	return nil
}

func (s *VoucherService) List(ctx context.Context, params pagination.Params, f domain.VoucherFilter) ([]domain.Voucher, int64, error) {
	f.Search = strings.TrimSpace(f.Search)
	return s.repo.ListPaginated(ctx, params, f)
}

func (s *VoucherService) Get(ctx context.Context, id int64) (*domain.Voucher, error) {
	v, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, apperror.NotFound("Voucher tidak ditemukan")
	}
	return v, nil
}

func (s *VoucherService) Create(ctx context.Context, in VoucherInput) (*domain.Voucher, error) {
	if err := s.validate(ctx, &in); err != nil {
		return nil, err
	}
	if err := s.ensureCodeFree(ctx, in.Code, 0); err != nil {
		return nil, err
	}
	v := &domain.Voucher{IsActive: true}
	apply(v, in)
	if err := s.repo.Create(ctx, v); err != nil {
		return nil, err
	}
	return s.Get(ctx, v.ID)
}

func apply(v *domain.Voucher, in VoucherInput) {
	v.Code, v.AppID, v.PlanID = in.Code, in.AppID, in.PlanID
	v.Type, v.Value, v.MaxDiscount = in.Type, in.Value, in.MaxDiscount
	v.ValidFrom, v.ValidUntil, v.Quota, v.OncePerCustomer = in.ValidFrom, in.ValidUntil, in.Quota, in.OncePerCustomer
}

// Update edits a voucher. expected is the If-Match version (nil = write
// unconditionally). Its App cannot change once it has been used: its uses
// would otherwise sit under an App they were never charged to.
func (s *VoucherService) Update(ctx context.Context, id int64, in VoucherInput, expected *time.Time) (*domain.Voucher, error) {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if expected != nil && !etag.Same(existing.UpdatedAt, *expected) {
		return nil, apperror.Stale(existing)
	}
	if err := s.validate(ctx, &in); err != nil {
		return nil, err
	}
	if in.AppID != existing.AppID {
		used, err := s.repo.CountRedemptions(ctx, id)
		if err != nil {
			return nil, err
		}
		if used > 0 {
			return nil, apperror.Validation("Aplikasi voucher tidak bisa diubah", map[string][]string{"appId": {"Voucher ini sudah pernah dipakai, aplikasinya tidak bisa diganti"}})
		}
	}
	if err := s.ensureCodeFree(ctx, in.Code, id); err != nil {
		return nil, err
	}
	apply(existing, in)
	ok, err := s.repo.Update(ctx, existing, expected)
	if err != nil {
		return nil, err
	}
	if !ok {
		current, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		return nil, apperror.Stale(current)
	}
	return s.Get(ctx, id)
}

// SetActive is the explicit state change (ADR-0036): a deactivated voucher is
// refused at once; uses already reserved keep their discount.
func (s *VoucherService) SetActive(ctx context.Context, id int64, active bool) (*domain.Voucher, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	if err := s.repo.SetActive(ctx, id, active); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// RedemptionRow is a use with its customer's display name ("" when unknown).
type RedemptionRow struct {
	domain.VoucherRedemption
	CustomerName string
}

func (s *VoucherService) ListRedemptions(ctx context.Context, voucherID int64, params pagination.Params) ([]RedemptionRow, int64, error) {
	if _, err := s.Get(ctx, voucherID); err != nil {
		return nil, 0, err
	}
	list, total, err := s.repo.ListRedemptions(ctx, voucherID, params)
	if err != nil {
		return nil, 0, err
	}
	var tenantIDs []int64
	for _, r := range list {
		if r.Customer.Type == domain.CustomerTenant {
			if id, err := strconv.ParseInt(r.Customer.Ref, 10, 64); err == nil {
				tenantIDs = append(tenantIDs, id)
			}
		}
	}
	names := map[int64]string{}
	if s.tenants != nil && len(tenantIDs) > 0 {
		if names, err = s.tenants.TenantNames(ctx, tenantIDs); err != nil {
			return nil, 0, err
		}
	}
	rows := make([]RedemptionRow, len(list))
	for i, r := range list {
		rows[i] = RedemptionRow{VoucherRedemption: r}
		if r.Customer.Type == domain.CustomerTenant {
			if id, err := strconv.ParseInt(r.Customer.Ref, 10, 64); err == nil {
				rows[i].CustomerName = names[id]
			}
		}
	}
	return rows, total, nil
}

// ---- using a voucher ----

// Machine-readable reasons a code is refused (errors.code / errors.reason).
const (
	VoucherNotFound    = "voucher_not_found"
	VoucherInactive    = "voucher_inactive"
	VoucherNotStarted  = "voucher_not_started"
	VoucherExpired     = "voucher_expired"
	VoucherExhausted   = "voucher_exhausted"
	VoucherAlreadyUsed = "voucher_already_used"
	VoucherWrongPlan   = "voucher_wrong_plan"
	VoucherTooLarge    = "voucher_too_large"
)

func voucherRefusal(reason, message string) error {
	return apperror.Validation(message, map[string][]string{"voucherCode": {message}}).WithCode(reason)
}

// VoucherQuote is what a valid code does to a plan's price.
type VoucherQuote struct {
	Code           string
	PlanID         int64
	OriginalAmount int64
	DiscountAmount int64
	FinalAmount    int64
}

// check is every rule a code must pass for plan and customer, in the order a
// customer can act on: a code that does not exist (or belongs to another App,
// which looks the same), then whether it can be used at all today, then
// whether it fits this plan, then this customer, then the price.
func (s *VoucherService) check(v *domain.Voucher, plan *domain.Plan, customerUses int64) (*VoucherQuote, error) {
	if v == nil || v.AppID != plan.AppID {
		return nil, voucherRefusal(VoucherNotFound, "Kode voucher tidak ditemukan. Periksa lagi penulisannya.")
	}
	today := s.Today()
	switch {
	case !v.IsActive:
		return nil, voucherRefusal(VoucherInactive, "Voucher ini sudah tidak berlaku.")
	case v.ValidFrom != "" && today < v.ValidFrom:
		return nil, voucherRefusal(VoucherNotStarted, "Voucher ini baru bisa dipakai mulai "+formatDay(v.ValidFrom)+".")
	case v.ValidUntil != "" && today > v.ValidUntil:
		return nil, voucherRefusal(VoucherExpired, "Voucher ini sudah kedaluwarsa (berlaku sampai "+formatDay(v.ValidUntil)+").")
	case v.PlanID != nil && *v.PlanID != plan.ID:
		name := v.PlanName
		if name == "" {
			return nil, voucherRefusal(VoucherWrongPlan, "Voucher ini tidak berlaku untuk paket yang dipilih.")
		}
		return nil, voucherRefusal(VoucherWrongPlan, "Voucher ini hanya berlaku untuk paket "+name+".")
	case v.OncePerCustomer && customerUses > 0:
		return nil, voucherRefusal(VoucherAlreadyUsed, "Voucher ini sudah pernah Anda pakai. Satu pelanggan hanya boleh memakainya sekali.")
	case v.Quota != nil && v.UsedCount+v.ReservedCount >= *v.Quota:
		return nil, voucherRefusal(VoucherExhausted, "Kuota voucher ini sudah habis.")
	}
	discount := v.DiscountFor(plan.Price)
	if discount >= plan.Price {
		// Never Rp0 (owner's decision): a code that would take the whole price is refused.
		return nil, voucherRefusal(VoucherTooLarge, "Potongan voucher ini melebihi harga paket, jadi tidak bisa dipakai untuk paket ini.")
	}
	return &VoucherQuote{Code: v.Code, PlanID: plan.ID, OriginalAmount: plan.Price, DiscountAmount: discount, FinalAmount: plan.Price - discount}, nil
}

func emptyCode() error {
	return voucherRefusal(VoucherNotFound, "Masukkan kode voucher.")
}

// Quote previews code on plan for customer without reserving anything.
func (s *VoucherService) Quote(ctx context.Context, code string, plan *domain.Plan, c domain.Customer) (*VoucherQuote, error) {
	code = NormalizeVoucherCode(code)
	if code == "" {
		return nil, emptyCode()
	}
	v, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	var uses int64
	if v != nil && v.OncePerCustomer {
		if uses, err = s.repo.CustomerUses(ctx, v.ID, c); err != nil {
			return nil, err
		}
	}
	return s.check(v, plan, uses)
}

// Reservation is a reserved use: the quote and the row to delete if the charge
// it was taken for never comes to exist.
type Reservation struct {
	VoucherQuote
	RedemptionID int64
}

// Reserve takes one use of code for the charge orderRef (checked again under a
// row lock). The use counts against the quota until Settle releases it.
func (s *VoucherService) Reserve(ctx context.Context, code string, plan *domain.Plan, c domain.Customer, orderRef string) (*Reservation, error) {
	code = NormalizeVoucherCode(code)
	if code == "" {
		return nil, emptyCode()
	}
	var quote *VoucherQuote
	red, err := s.repo.Reserve(ctx, code, c, func(v *domain.Voucher, uses int64) (*domain.VoucherRedemption, error) {
		q, err := s.check(v, plan, uses)
		if err != nil {
			return nil, err
		}
		quote = q
		return &domain.VoucherRedemption{
			VoucherID: v.ID, VoucherCode: v.Code, OrderRef: orderRef, Customer: c, PlanID: plan.ID,
			OriginalAmount: q.OriginalAmount, DiscountAmount: q.DiscountAmount, FinalAmount: q.FinalAmount,
			Status: domain.RedemptionReserved,
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return &Reservation{VoucherQuote: *quote, RedemptionID: red.ID}, nil
}

// Cancel deletes a reservation whose charge was never created.
func (s *VoucherService) Cancel(ctx context.Context, r *Reservation) error {
	if r == nil {
		return nil
	}
	return s.repo.DeleteReservation(ctx, r.RedemptionID)
}

// Settle records how the charge orderRef ended: paid -> used, otherwise
// released (no longer counted). A charge without a voucher is a no-op.
func (s *VoucherService) Settle(ctx context.Context, orderRef string, paid bool) error {
	status := domain.RedemptionReleased
	if paid {
		status = domain.RedemptionUsed
	}
	return s.repo.Settle(ctx, orderRef, status)
}

// staleReservationAge is how long a reservation may exist without its charge
// being recorded: recording takes seconds, so an hour means the process died.
const staleReservationAge = time.Hour

// ReleaseStale is the safety net run by the subscription reconciler.
func (s *VoucherService) ReleaseStale(ctx context.Context) (int64, error) {
	return s.repo.ReleaseStale(ctx, s.now().Add(-staleReservationAge))
}

// ---- formatting for messages ----

var monthsID = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

// formatDay renders "2026-11-10" as "10 November 2026".
func formatDay(day string) string {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return day
	}
	return fmt.Sprintf("%d %s %d", t.Day(), monthsID[t.Month()-1], t.Year())
}

// formatRupiah renders 1500000 as "Rp1.500.000".
func formatRupiah(n int64) string {
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	return "Rp" + b.String()
}
