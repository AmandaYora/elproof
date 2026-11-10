package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/utils"
)

// MySQLVoucherRepository backs subscription_vouchers and
// subscription_voucher_redemptions (migration 000074). Every table it reads
// is owned by `billing` (the plan name join and the ledgers in ReleaseStale).
type MySQLVoucherRepository struct {
	db *sql.DB
}

func NewMySQLVoucherRepository(db *sql.DB) *MySQLVoucherRepository {
	return &MySQLVoucherRepository{db: db}
}

// voucherSelect reads a voucher with its plan's name and its live use counts.
const voucherSelect = `SELECT v.id, v.code, v.app_id, v.plan_id, COALESCE(p.name, ''), v.discount_type, v.discount_value,
	v.max_discount, v.valid_from, v.valid_until, v.quota, v.once_per_customer, v.is_active, v.created_at, v.updated_at,
	(SELECT COUNT(*) FROM subscription_voucher_redemptions r WHERE r.voucher_id = v.id AND r.status = 'used'),
	(SELECT COUNT(*) FROM subscription_voucher_redemptions r WHERE r.voucher_id = v.id AND r.status = 'reserved')
	FROM subscription_vouchers v LEFT JOIN subscription_plans p ON p.id = v.plan_id`

func scanVoucher(scan func(dest ...interface{}) error) (*domain.Voucher, error) {
	var v domain.Voucher
	var planID, maxDiscount, quota sql.NullInt64
	var validFrom, validUntil sql.NullTime
	var discountType string
	if err := scan(&v.ID, &v.Code, &v.AppID, &planID, &v.PlanName, &discountType, &v.Value,
		&maxDiscount, &validFrom, &validUntil, &quota, &v.OncePerCustomer, &v.IsActive, &v.CreatedAt, &v.UpdatedAt,
		&v.UsedCount, &v.ReservedCount); err != nil {
		return nil, err
	}
	v.Type = domain.DiscountType(discountType)
	if planID.Valid {
		id := planID.Int64
		v.PlanID = &id
	}
	if maxDiscount.Valid {
		m := maxDiscount.Int64
		v.MaxDiscount = &m
	}
	if quota.Valid {
		q := quota.Int64
		v.Quota = &q
	}
	if validFrom.Valid {
		v.ValidFrom = validFrom.Time.Format("2006-01-02")
	}
	if validUntil.Valid {
		v.ValidUntil = validUntil.Time.Format("2006-01-02")
	}
	return &v, nil
}

func nullDate(day string) interface{} {
	if day == "" {
		return nil
	}
	return day
}

func nullInt(p *int64) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func voucherWhere(f domain.VoucherFilter) (string, []interface{}) {
	var conds []string
	var args []interface{}
	if f.AppID != "" {
		conds = append(conds, `v.app_id = ?`)
		args = append(args, f.AppID)
	}
	if f.Search != "" {
		conds = append(conds, `v.code LIKE ?`)
		args = append(args, "%"+utils.EscapeLike(strings.ToUpper(f.Search))+"%")
	}
	if len(conds) == 0 {
		return "", nil
	}
	return ` WHERE ` + strings.Join(conds, " AND "), args
}

func (r *MySQLVoucherRepository) ListPaginated(ctx context.Context, params pagination.Params, f domain.VoucherFilter) ([]domain.Voucher, int64, error) {
	where, args := voucherWhere(f)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM subscription_vouchers v`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, voucherSelect+where+` ORDER BY v.created_at DESC, v.id DESC LIMIT ? OFFSET ?`,
		append(args, params.Limit, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]domain.Voucher, 0)
	for rows.Next() {
		v, err := scanVoucher(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *v)
	}
	return out, total, rows.Err()
}

func (r *MySQLVoucherRepository) findOne(ctx context.Context, q queryRower, cond string, arg interface{}) (*domain.Voucher, error) {
	v, err := scanVoucher(q.QueryRowContext(ctx, voucherSelect+` WHERE `+cond+` LIMIT 1`, arg).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return v, err
}

type queryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func (r *MySQLVoucherRepository) FindByID(ctx context.Context, id int64) (*domain.Voucher, error) {
	return r.findOne(ctx, r.db, `v.id = ?`, id)
}

func (r *MySQLVoucherRepository) FindByCode(ctx context.Context, code string) (*domain.Voucher, error) {
	return r.findOne(ctx, r.db, `v.code = ?`, code)
}

func (r *MySQLVoucherRepository) CodeTaken(ctx context.Context, code string, exceptID int64) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM subscription_vouchers WHERE code = ? AND id <> ?`, code, exceptID).Scan(&n)
	return n > 0, err
}

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

func (r *MySQLVoucherRepository) Create(ctx context.Context, v *domain.Voucher) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO subscription_vouchers (code, app_id, plan_id, discount_type, discount_value, max_discount,
		   valid_from, valid_until, quota, once_per_customer, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		v.Code, v.AppID, nullInt(v.PlanID), string(v.Type), v.Value, nullInt(v.MaxDiscount),
		nullDate(v.ValidFrom), nullDate(v.ValidUntil), nullInt(v.Quota), v.OncePerCustomer, v.IsActive)
	if isDuplicate(err) {
		return apperror.Validation("Kode voucher sudah dipakai", map[string][]string{"code": {"Kode ini sudah dipakai voucher lain"}})
	}
	if err != nil {
		return err
	}
	v.ID, err = res.LastInsertId()
	return err
}

func (r *MySQLVoucherRepository) Update(ctx context.Context, v *domain.Voucher, expected *time.Time) (bool, error) {
	query := `UPDATE subscription_vouchers SET code = ?, app_id = ?, plan_id = ?, discount_type = ?, discount_value = ?,
		max_discount = ?, valid_from = ?, valid_until = ?, quota = ?, once_per_customer = ?, updated_at = CURRENT_TIMESTAMP(6)
		WHERE id = ?`
	args := []interface{}{v.Code, v.AppID, nullInt(v.PlanID), string(v.Type), v.Value, nullInt(v.MaxDiscount),
		nullDate(v.ValidFrom), nullDate(v.ValidUntil), nullInt(v.Quota), v.OncePerCustomer, v.ID}
	if expected != nil {
		query += ` AND updated_at = ?`
		args = append(args, *expected)
	}
	res, err := r.db.ExecContext(ctx, query, args...)
	if isDuplicate(err) {
		return false, apperror.Validation("Kode voucher sudah dipakai", map[string][]string{"code": {"Kode ini sudah dipakai voucher lain"}})
	}
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func (r *MySQLVoucherRepository) SetActive(ctx context.Context, id int64, active bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE subscription_vouchers SET is_active = ?, updated_at = CURRENT_TIMESTAMP(6) WHERE id = ?`, active, id)
	return err
}

func (r *MySQLVoucherRepository) CountRedemptions(ctx context.Context, voucherID int64) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM subscription_voucher_redemptions WHERE voucher_id = ?`, voucherID).Scan(&n)
	return n, err
}

const redemptionColumns = `id, voucher_id, voucher_code, order_ref, customer_type, customer_ref, plan_id,
	original_amount, discount_amount, final_amount, status, created_at, resolved_at`

func (r *MySQLVoucherRepository) ListRedemptions(ctx context.Context, voucherID int64, params pagination.Params) ([]domain.VoucherRedemption, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM subscription_voucher_redemptions WHERE voucher_id = ?`, voucherID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+redemptionColumns+` FROM subscription_voucher_redemptions WHERE voucher_id = ?
		 ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`, voucherID, params.Limit, params.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]domain.VoucherRedemption, 0)
	for rows.Next() {
		var x domain.VoucherRedemption
		var ctype, status string
		var resolved sql.NullTime
		if err := rows.Scan(&x.ID, &x.VoucherID, &x.VoucherCode, &x.OrderRef, &ctype, &x.Customer.Ref, &x.PlanID,
			&x.OriginalAmount, &x.DiscountAmount, &x.FinalAmount, &status, &x.CreatedAt, &resolved); err != nil {
			return nil, 0, err
		}
		x.Customer.Type = domain.CustomerType(ctype)
		x.Status = domain.RedemptionStatus(status)
		if resolved.Valid {
			x.ResolvedAt = &resolved.Time
		}
		out = append(out, x)
	}
	return out, total, rows.Err()
}

func customerUses(ctx context.Context, q queryRower, voucherID int64, c domain.Customer) (int64, error) {
	var n int64
	err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM subscription_voucher_redemptions
		 WHERE voucher_id = ? AND customer_type = ? AND customer_ref = ? AND status IN ('reserved', 'used')`,
		voucherID, string(c.Type), c.Ref).Scan(&n)
	return n, err
}

func (r *MySQLVoucherRepository) CustomerUses(ctx context.Context, voucherID int64, c domain.Customer) (int64, error) {
	return customerUses(ctx, r.db, voucherID, c)
}

// reserveTx: READ COMMITTED so the counts read after the row lock see every use
// committed by the transaction that held the lock before (and no gap locks are
// taken on the redemptions table, see the project's deadlock note on 1213).
var reserveTx = &sql.TxOptions{Isolation: sql.LevelReadCommitted}

func (r *MySQLVoucherRepository) Reserve(ctx context.Context, code string, c domain.Customer,
	decide func(v *domain.Voucher, customerUses int64) (*domain.VoucherRedemption, error)) (*domain.VoucherRedemption, error) {
	tx, err := r.db.BeginTx(ctx, reserveTx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	// Lock the voucher row alone first, then read it with its counts in a NEW
	// statement: under READ COMMITTED every statement takes a fresh snapshot,
	// so that read sees each use committed by whoever held the lock before us
	// (a snapshot taken by the locking statement itself could predate them).
	var lockedID int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM subscription_vouchers WHERE code = ? FOR UPDATE`, code).Scan(&lockedID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	var v *domain.Voucher
	if err == nil {
		if v, err = r.findOne(ctx, tx, `v.id = ?`, lockedID); err != nil {
			return nil, err
		}
	}
	var uses int64
	if v != nil {
		if uses, err = customerUses(ctx, tx, v.ID, c); err != nil {
			return nil, err
		}
	}
	red, err := decide(v, uses)
	if err != nil {
		return nil, err
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO subscription_voucher_redemptions (voucher_id, voucher_code, order_ref, customer_type, customer_ref,
		   plan_id, original_amount, discount_amount, final_amount, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		red.VoucherID, red.VoucherCode, red.OrderRef, string(red.Customer.Type), red.Customer.Ref, red.PlanID,
		red.OriginalAmount, red.DiscountAmount, red.FinalAmount, string(red.Status))
	if isDuplicate(err) {
		return nil, apperror.Conflict("orderRef sudah pernah dipakai")
	}
	if err != nil {
		return nil, err
	}
	if red.ID, err = res.LastInsertId(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return red, nil
}

func (r *MySQLVoucherRepository) DeleteReservation(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM subscription_voucher_redemptions WHERE id = ? AND status = 'reserved'`, id)
	return err
}

func (r *MySQLVoucherRepository) Settle(ctx context.Context, orderRef string, status domain.RedemptionStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE subscription_voucher_redemptions SET status = ?, resolved_at = CURRENT_TIMESTAMP(6)
		 WHERE order_ref = ? AND status = 'reserved'`, string(status), orderRef)
	return err
}

func (r *MySQLVoucherRepository) ReleaseStale(ctx context.Context, olderThan time.Time) (int64, error) {
	// A use whose charge is already paid but was never settled (the settle call
	// failed, or the process died in between) becomes used, so the console
	// shows it as it is. It already counted against the quota.
	paid, err := r.db.ExecContext(ctx,
		`UPDATE subscription_voucher_redemptions r SET r.status = 'used', r.resolved_at = CURRENT_TIMESTAMP(6)
		 WHERE r.status = 'reserved' AND r.created_at < ?
		   AND (EXISTS (SELECT 1 FROM subscription_transactions t WHERE t.payment_reference = r.order_ref AND t.status = 'paid')
		     OR EXISTS (SELECT 1 FROM subscription_app_charges c WHERE c.order_ref = r.order_ref AND c.status = 'paid'))`, olderThan)
	if err != nil {
		return 0, err
	}
	used, err := paid.RowsAffected()
	if err != nil {
		return 0, err
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE subscription_voucher_redemptions r SET r.status = 'released', r.resolved_at = CURRENT_TIMESTAMP(6)
		 WHERE r.status = 'reserved' AND r.created_at < ?
		   AND NOT EXISTS (SELECT 1 FROM subscription_transactions t
		                   WHERE t.payment_reference = r.order_ref AND t.status IN ('pending', 'paid'))
		   AND NOT EXISTS (SELECT 1 FROM subscription_app_charges c
		                   WHERE c.order_ref = r.order_ref AND c.status IN ('pending', 'paid'))`, olderThan)
	if err != nil {
		return 0, err
	}
	released, err := res.RowsAffected()
	return used + released, err
}
