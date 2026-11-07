package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"elproof/internal/modules/platform/domain"
)

// MySQLConsoleRepository holds the Platform Console's own tables (notes, data
// requests, announcements, policy, support access). Every table is owned by
// this module; tenants is joined only for display names.
type MySQLConsoleRepository struct {
	db *sql.DB
}

func NewMySQLConsoleRepository(db *sql.DB) *MySQLConsoleRepository {
	return &MySQLConsoleRepository{db: db}
}

// ---- notes ----

func (r *MySQLConsoleRepository) ListNotes(ctx context.Context, tenantID int64) ([]domain.TenantNote, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, tenant_id, admin_id, admin_name, body, created_at FROM tenant_notes WHERE tenant_id = ? ORDER BY created_at DESC, id DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.TenantNote
	for rows.Next() {
		var n domain.TenantNote
		if err := rows.Scan(&n.ID, &n.TenantID, &n.AdminID, &n.AdminName, &n.Body, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *MySQLConsoleRepository) AddNote(ctx context.Context, n *domain.TenantNote) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO tenant_notes (tenant_id, admin_id, admin_name, body) VALUES (?, ?, ?, ?)`, n.TenantID, n.AdminID, n.AdminName, n.Body)
	if err != nil {
		return err
	}
	n.ID, err = res.LastInsertId()
	return err
}

// ---- data requests ----

const dataRequestColumns = `d.id, d.tenant_id, t.business_name, d.kind, d.status, d.requested_by, COALESCE(d.note, ''), d.due_at, d.created_at,
	d.completed_at, COALESCE(d.completed_by, ''), COALESCE(d.completion_note, '')`

func scanDataRequest(scan func(...interface{}) error) (domain.DataRequest, error) {
	var d domain.DataRequest
	var kind, status string
	var completedAt sql.NullTime
	if err := scan(&d.ID, &d.TenantID, &d.TenantName, &kind, &status, &d.RequestedBy, &d.Note, &d.DueAt, &d.CreatedAt, &completedAt, &d.CompletedBy, &d.CompletionNote); err != nil {
		return d, err
	}
	d.Kind, d.Status = domain.DataRequestKind(kind), domain.DataRequestStatus(status)
	if completedAt.Valid {
		d.CompletedAt = &completedAt.Time
	}
	return d, nil
}

// ListDataRequests: one tenant's requests, or (tenantID nil) every request,
// optionally only the ones still open, soonest deadline first.
func (r *MySQLConsoleRepository) ListDataRequests(ctx context.Context, tenantID *int64, openOnly bool) ([]domain.DataRequest, error) {
	q := `SELECT ` + dataRequestColumns + ` FROM data_requests d JOIN tenants t ON t.id = d.tenant_id WHERE 1 = 1`
	var args []interface{}
	if tenantID != nil {
		q += ` AND d.tenant_id = ?`
		args = append(args, *tenantID)
	}
	if openOnly {
		q += ` AND d.status = 'open'`
	}
	q += ` ORDER BY (d.status = 'open') DESC, d.due_at ASC, d.id DESC`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.DataRequest
	for rows.Next() {
		d, err := scanDataRequest(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *MySQLConsoleRepository) AddDataRequest(ctx context.Context, d *domain.DataRequest, adminID string) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO data_requests (tenant_id, kind, requested_by, note, due_at, created_by_admin) VALUES (?, ?, ?, ?, ?, ?)`,
		d.TenantID, string(d.Kind), d.RequestedBy, d.Note, d.DueAt, adminID)
	if err != nil {
		return err
	}
	d.ID, err = res.LastInsertId()
	return err
}

// CompleteDataRequest closes an OPEN request; false when it was not open.
func (r *MySQLConsoleRepository) CompleteDataRequest(ctx context.Context, id int64, status domain.DataRequestStatus, adminID, note string, at time.Time) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE data_requests SET status = ?, completed_at = ?, completed_by = ?, completion_note = ? WHERE id = ? AND status = 'open'`,
		string(status), at, adminID, note, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// ---- announcements ----

const announcementColumns = `id, title, body, kind, audience, COALESCE(audience_value, ''), starts_at, ends_at, created_by, created_at`

func scanAnnouncement(scan func(...interface{}) error) (domain.Announcement, error) {
	var a domain.Announcement
	var kind, audience string
	var ends sql.NullTime
	if err := scan(&a.ID, &a.Title, &a.Body, &kind, &audience, &a.AudienceValue, &a.StartsAt, &ends, &a.CreatedBy, &a.CreatedAt); err != nil {
		return a, err
	}
	a.Kind, a.Audience = domain.AnnouncementKind(kind), domain.AnnouncementAudience(audience)
	if ends.Valid {
		a.EndsAt = &ends.Time
	}
	return a, nil
}

func (r *MySQLConsoleRepository) ListAnnouncements(ctx context.Context) ([]domain.Announcement, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+announcementColumns+` FROM announcements ORDER BY starts_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Announcement
	for rows.Next() {
		a, err := scanAnnouncement(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *MySQLConsoleRepository) CreateAnnouncement(ctx context.Context, a *domain.Announcement) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO announcements (title, body, kind, audience, audience_value, starts_at, ends_at, created_by) VALUES (?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?)`,
		a.Title, a.Body, string(a.Kind), string(a.Audience), a.AudienceValue, a.StartsAt, a.EndsAt, a.CreatedBy)
	if err != nil {
		return err
	}
	a.ID, err = res.LastInsertId()
	return err
}

func (r *MySQLConsoleRepository) UpdateAnnouncement(ctx context.Context, a *domain.Announcement) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE announcements SET title = ?, body = ?, kind = ?, audience = ?, audience_value = NULLIF(?, ''), starts_at = ?, ends_at = ? WHERE id = ?`,
		a.Title, a.Body, string(a.Kind), string(a.Audience), a.AudienceValue, a.StartsAt, a.EndsAt, a.ID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (r *MySQLConsoleRepository) DeleteAnnouncement(ctx context.Context, id int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM announcements WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// ---- subscription policy ----

func (r *MySQLConsoleRepository) GetPolicy(ctx context.Context) (domain.SubscriptionPolicy, bool, error) {
	var p domain.SubscriptionPolicy
	err := r.db.QueryRowContext(ctx, `SELECT trial_days, expiring_soon_days, grace_days, read_only_days FROM subscription_policy WHERE id = 1`).
		Scan(&p.TrialDays, &p.ExpiringSoonDays, &p.GraceDays, &p.ReadOnlyDays)
	if err == sql.ErrNoRows {
		return p, false, nil
	}
	return p, err == nil, err
}

func (r *MySQLConsoleRepository) SavePolicy(ctx context.Context, p domain.SubscriptionPolicy, adminID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO subscription_policy (id, trial_days, expiring_soon_days, grace_days, read_only_days, updated_by) VALUES (1, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE trial_days = VALUES(trial_days), expiring_soon_days = VALUES(expiring_soon_days),
		   grace_days = VALUES(grace_days), read_only_days = VALUES(read_only_days), updated_by = VALUES(updated_by)`,
		p.TrialDays, p.ExpiringSoonDays, p.GraceDays, p.ReadOnlyDays, adminID)
	return err
}

// ---- support access ----

func (r *MySQLConsoleRepository) LatestGrant(ctx context.Context, tenantID int64) (*domain.SupportAccessGrant, error) {
	var g domain.SupportAccessGrant
	var revoked sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, granted_by_staff_id, granted_at, expires_at, revoked_at FROM support_access_grants WHERE tenant_id = ? ORDER BY id DESC LIMIT 1`, tenantID).
		Scan(&g.ID, &g.TenantID, &g.GrantedByStaffID, &g.GrantedAt, &g.ExpiresAt, &revoked)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if revoked.Valid {
		g.RevokedAt = &revoked.Time
	}
	return &g, nil
}

func (r *MySQLConsoleRepository) CreateGrant(ctx context.Context, g *domain.SupportAccessGrant) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO support_access_grants (tenant_id, granted_by_staff_id, expires_at) VALUES (?, ?, ?)`, g.TenantID, g.GrantedByStaffID, g.ExpiresAt)
	if err != nil {
		return err
	}
	g.ID, err = res.LastInsertId()
	return err
}

func (r *MySQLConsoleRepository) RevokeGrants(ctx context.Context, tenantID int64, at time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE support_access_grants SET revoked_at = ? WHERE tenant_id = ? AND revoked_at IS NULL`, at, tenantID)
	return err
}

func (r *MySQLConsoleRepository) CreateSession(ctx context.Context, s *domain.SupportSession) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO support_sessions (tenant_id, admin_id, admin_name, reason, expires_at) VALUES (?, ?, ?, ?, ?)`,
		s.TenantID, s.AdminID, s.AdminName, s.Reason, s.ExpiresAt)
	if err != nil {
		return err
	}
	s.ID, err = res.LastInsertId()
	return err
}

// EndSession closes a session owned by adminID; false when not found or already ended.
func (r *MySQLConsoleRepository) EndSession(ctx context.Context, id int64, adminID string, at time.Time) (bool, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE support_sessions SET ended_at = ? WHERE id = ? AND admin_id = ? AND ended_at IS NULL`, at, id, adminID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (r *MySQLConsoleRepository) FindSession(ctx context.Context, id int64) (*domain.SupportSession, error) {
	var s domain.SupportSession
	var ended sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, admin_id, admin_name, reason, started_at, expires_at, ended_at FROM support_sessions WHERE id = ?`, id).
		Scan(&s.ID, &s.TenantID, &s.AdminID, &s.AdminName, &s.Reason, &s.StartedAt, &s.ExpiresAt, &ended)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if ended.Valid {
		s.EndedAt = &ended.Time
	}
	return &s, nil
}

// DeleteSession removes a session whose token was never issued.
func (r *MySQLConsoleRepository) DeleteSession(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM support_sessions WHERE id = ? AND ended_at IS NULL`, id)
	return err
}

func (r *MySQLConsoleRepository) ListSessions(ctx context.Context, tenantID int64, limit int) ([]domain.SupportSession, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, admin_id, admin_name, reason, started_at, expires_at, ended_at FROM support_sessions WHERE tenant_id = ? ORDER BY id DESC LIMIT ?`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SupportSession
	for rows.Next() {
		var s domain.SupportSession
		var ended sql.NullTime
		if err := rows.Scan(&s.ID, &s.TenantID, &s.AdminID, &s.AdminName, &s.Reason, &s.StartedAt, &s.ExpiresAt, &ended); err != nil {
			return nil, err
		}
		if ended.Valid {
			s.EndedAt = &ended.Time
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ---- tenant 360 helpers ----

func (r *MySQLConsoleRepository) ListManualActivations(ctx context.Context, tenantID int64) ([]domain.ManualActivation, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, plan_id, actor_admin_id, reason, proof_reference, note, transaction_reference, expires_at, created_at
		 FROM manual_activations WHERE tenant_id = ? ORDER BY created_at DESC, id DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ManualActivation
	for rows.Next() {
		var a domain.ManualActivation
		var reason string
		if err := rows.Scan(&a.ID, &a.TenantID, &a.PlanID, &a.ActorAdminID, &reason, &a.ProofReference, &a.Note, &a.TransactionReference, &a.ExpiresAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.Reason = domain.ManualActivationReason(reason)
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetSignupConsent returns the consent record of a self-service tenant (nil when created by the team).
func (r *MySQLConsoleRepository) GetSignupConsent(ctx context.Context, tenantID int64) (*domain.SignupConsent, error) {
	var c domain.SignupConsent
	err := r.db.QueryRowContext(ctx,
		`SELECT terms_version, terms_accepted_at, marketing_opt_in, ip_address FROM signup_requests WHERE tenant_id = ? LIMIT 1`, tenantID).
		Scan(&c.TermsVersion, &c.TermsAcceptedAt, &c.MarketingOptIn, &c.IPAddress)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}
