package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"

	"elproof/internal/modules/platform/domain"
)

// MySQLSignupRepository stores self-service registrations (signup_requests).
type MySQLSignupRepository struct {
	db *sql.DB
}

func NewMySQLSignupRepository(db *sql.DB) *MySQLSignupRepository {
	return &MySQLSignupRepository{db: db}
}

const signupColumns = `id, google_sub, email, owner_name, business_name, terms_version, terms_accepted_at,
	marketing_opt_in, ip_address, user_agent, status, tenant_id, completed_at, created_at`

func (r *MySQLSignupRepository) FindByGoogleSub(ctx context.Context, sub string) (*domain.SignupRequest, error) {
	var s domain.SignupRequest
	var status string
	var tenantID sql.NullInt64
	var completedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT `+signupColumns+` FROM signup_requests WHERE google_sub = ?`, sub).Scan(
		&s.ID, &s.GoogleSub, &s.Email, &s.OwnerName, &s.BusinessName, &s.TermsVersion, &s.TermsAcceptedAt,
		&s.MarketingOptIn, &s.IPAddress, &s.UserAgent, &status, &tenantID, &completedAt, &s.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.Status = domain.SignupStatus(status)
	if tenantID.Valid {
		s.TenantID = &tenantID.Int64
	}
	if completedAt.Valid {
		s.CompletedAt = &completedAt.Time
	}
	return &s, nil
}

func (r *MySQLSignupRepository) Create(ctx context.Context, s *domain.SignupRequest) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO signup_requests (google_sub, email, owner_name, business_name, terms_version, terms_accepted_at,
		   marketing_opt_in, ip_address, user_agent, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.GoogleSub, s.Email, s.OwnerName, s.BusinessName, s.TermsVersion, s.TermsAcceptedAt,
		s.MarketingOptIn, s.IPAddress, s.UserAgent, string(domain.SignupProvisioning),
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.ErrSignupExists
		}
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	s.ID = id
	s.Status = domain.SignupProvisioning
	return nil
}

// UpdateBusinessName records a changed WO name on a signup whose tenant
// does not exist yet.
func (r *MySQLSignupRepository) UpdateBusinessName(ctx context.Context, id int64, name string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE signup_requests SET business_name = ? WHERE id = ? AND tenant_id IS NULL`, name, id)
	return err
}

func (r *MySQLSignupRepository) MarkCompleted(ctx context.Context, id int64, when time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE signup_requests SET status = ?, completed_at = ? WHERE id = ?`, string(domain.SignupCompleted), when, id)
	return err
}

// MySQLTrialLimitsRepository reads/writes the single trial_limits row.
type MySQLTrialLimitsRepository struct {
	db *sql.DB
}

func NewMySQLTrialLimitsRepository(db *sql.DB) *MySQLTrialLimitsRepository {
	return &MySQLTrialLimitsRepository{db: db}
}

func (r *MySQLTrialLimitsRepository) Get(ctx context.Context) (domain.TrialLimits, error) {
	var l domain.TrialLimits
	err := r.db.QueryRowContext(ctx,
		`SELECT max_staff, max_active_projects, max_clients FROM trial_limits WHERE id = 1`,
	).Scan(&l.MaxStaff, &l.MaxActiveProjects, &l.MaxClients)
	return l, err
}

func (r *MySQLTrialLimitsRepository) Update(ctx context.Context, l domain.TrialLimits) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE trial_limits SET max_staff = ?, max_active_projects = ?, max_clients = ? WHERE id = 1`,
		l.MaxStaff, l.MaxActiveProjects, l.MaxClients)
	return err
}
