package application

import (
	"context"
	"strings"
	"time"

	identitycontracts "elproof/internal/modules/identity/contracts"
	projectscontracts "elproof/internal/modules/projects/contracts"

	"elproof/internal/modules/clients/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/logger"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/utils"
	"elproof/internal/shared/validator"
)

type ClientRepository interface {
	ListByProject(ctx context.Context, tenantID, projectID int64) ([]domain.Client, error)
	ListByTenant(ctx context.Context, tenantID int64) ([]domain.Client, error)
	ListByTenantPaginated(ctx context.Context, tenantID int64, params pagination.Params, f domain.ListFilter, sort pagination.Sort) ([]domain.Client, int64, error)
	ListAllFiltered(ctx context.Context, tenantID int64, f domain.ListFilter) ([]domain.Client, error)
	// ListSortSpec: the columns of this module's table the list can order by in SQL.
	ListSortSpec() map[string]pagination.SortColumn
	FindByID(ctx context.Context, tenantID, id int64) (*domain.Client, error)
	Create(ctx context.Context, c *domain.Client) error
	Update(ctx context.Context, c *domain.Client) error
	SetActive(ctx context.Context, tenantID, id int64, isActive bool) error
	SetCredentialResetAt(ctx context.Context, tenantID, id int64, when time.Time) error
	Delete(ctx context.Context, tenantID, id int64) error
	CountActive(ctx context.Context, tenantID int64) (int, error)
	UpdateAccessRole(ctx context.Context, tenantID, id, roleID int64) error
	UpdateUsername(ctx context.Context, tenantID, id int64, username string) error
	// Soft delete / restore / purge (ADR-0034).
	SoftDelete(ctx context.Context, tenantID, id int64, at time.Time) (bool, error)
	FindDeletedByID(ctx context.Context, tenantID, id int64) (*domain.Client, error)
	Restore(ctx context.Context, tenantID, id int64, notBefore time.Time) (bool, error)
	ListAllByProject(ctx context.Context, tenantID, projectID int64) ([]domain.Client, error)
	ListExpired(ctx context.Context, cutoff time.Time, limit int) ([]domain.Client, error)
	PurgeDelete(ctx context.Context, tenantID, id int64, cutoff time.Time) (bool, error)
	TryLock(ctx context.Context) (release func(), ok bool, err error)
}

// RoleResolver is the slice of the access module clients need (portal
// roles). Declared here, wired in cmd/server.
type RoleResolver interface {
	// DefaultClientRole is the built-in portal role for a client relation
	// (Pengantin for bride/groom, Keluarga for family).
	DefaultClientRole(ctx context.Context, tenantID int64, family bool) (int64, error)
	// AssignableClientRole validates a portal role id.
	AssignableClientRole(ctx context.Context, tenantID, roleID int64) (string, error)
	RoleNames(ctx context.Context, tenantID int64) (map[int64]string, error)
}

type ClientService struct {
	roles    RoleResolver
	repo     ClientRepository
	projects projectscontracts.Contracts
	identity identitycontracts.Contracts
	quota    TrialQuota
}

func NewClientService(repo ClientRepository, projects projectscontracts.Contracts, identity identitycontracts.Contracts) *ClientService {
	return &ClientService{repo: repo, projects: projects, identity: identity}
}

func (s *ClientService) ListByProject(ctx context.Context, tenantID, projectID int64) ([]domain.Client, error) {
	return s.repo.ListByProject(ctx, tenantID, projectID)
}

// GetDeleted is the deleted client (404 when it is live, never existed, or has
// been purged), for the restore route's permission check.
func (s *ClientService) GetDeleted(ctx context.Context, tenantID, id int64) (*domain.Client, error) {
	c, err := s.repo.FindDeletedByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperror.NotFound("Client tidak ditemukan atau sudah tidak dapat dipulihkan")
	}
	return c, nil
}

// VerifyAssignedProject confirms a staff member whose role only reaches
// their own projects (scope "assigned", ADR-0026) is PIC or PIC Sales of
// projectID — the same rule `projects` applies to its own sub-resources.
func (s *ClientService) VerifyAssignedProject(ctx context.Context, tenantID, projectID, staffID int64) error {
	pic, err := s.projects.ProjectPICStaffID(ctx, tenantID, projectID)
	if err != nil {
		return err
	}
	if pic == staffID {
		return nil
	}
	picSales, err := s.projects.ProjectPICSalesStaffID(ctx, tenantID, projectID)
	if err != nil {
		return err
	}
	if picSales == staffID {
		return nil
	}
	return apperror.Forbidden("Anda tidak memiliki akses ke project ini")
}

// ListByTenant powers the WO Console's global search, which needs to match
// client names across every project at once rather than one project at a time.
func (s *ClientService) ListByTenant(ctx context.Context, tenantID int64) ([]domain.Client, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

// ClientSortSpec is the whitelist of GET /clients?sort= and its default order
// (today's: oldest first). name, role and createdAt are ordered by SQL; the
// three others (domain.SortProjectName, SortProjectEventDate, SortLoginStatus)
// are owned by other modules and ordered after they have been read.
func (s *ClientService) ClientSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	columns := map[string]pagination.SortColumn{}
	for name, col := range s.repo.ListSortSpec() {
		columns[name] = col
	}
	for _, name := range []string{domain.SortProjectName, domain.SortProjectEventDate, domain.SortLoginStatus} {
		columns[name] = pagination.Col("") // never rendered: these are ordered in memory
	}
	return columns, pagination.By("createdAt", false)
}

// ClientPage is one page of the list with its total.
type ClientPage struct {
	Rows  []domain.ClientRow
	Total int64
}

// ListPage is the paginated client list: the SQL filters (search, role, project)
// and ordering run in this module's table; the project's name and event date
// come from `projects` and the sign-in state from `identity`, each in ONE batch
// call for the rows that need them. When the order or the loginStatus filter
// needs those foreign values, the matching clients are read (rows of this
// tenant only, after the SQL filters), their foreign values resolved, and the
// filtering, ordering and slicing done here: a tenant has hundreds of clients,
// not millions, and no join into another module's tables is allowed.
func (s *ClientService) ListPage(ctx context.Context, tenantID int64, params pagination.Params, f domain.ListFilter, sort pagination.Sort) (*ClientPage, error) {
	if !domain.NeedsMemoryPaging(sort.Field, f.LoginStatus) {
		list, total, err := s.repo.ListByTenantPaginated(ctx, tenantID, params, f, sort)
		if err != nil {
			return nil, err
		}
		rows, err := s.enrich(ctx, tenantID, list, true, true)
		if err != nil {
			return nil, err
		}
		return &ClientPage{Rows: rows, Total: total}, nil
	}

	all, err := s.repo.ListAllFiltered(ctx, tenantID, f)
	if err != nil {
		return nil, err
	}
	needsProject := sort.Field == domain.SortProjectName || sort.Field == domain.SortProjectEventDate
	needsLogin := f.LoginStatus != "" || sort.Field == domain.SortLoginStatus
	rows, err := s.enrich(ctx, tenantID, all, needsProject, needsLogin)
	if err != nil {
		return nil, err
	}
	rows = domain.FilterByLoginStatus(rows, f.LoginStatus)
	domain.SortRows(rows, sort.Field, sort.Desc)
	total := int64(len(rows))
	start := params.Offset()
	if start > len(rows) {
		start = len(rows)
	}
	end := start + params.Limit
	if end > len(rows) {
		end = len(rows)
	}
	page := rows[start:end]
	// Whatever the page still lacks (the other foreign value) is read for the page only.
	if !needsProject || !needsLogin {
		clients := make([]domain.Client, len(page))
		for i, r := range page {
			clients[i] = r.Client
		}
		page, err = s.enrich(ctx, tenantID, clients, true, true)
		if err != nil {
			return nil, err
		}
	}
	return &ClientPage{Rows: page, Total: total}, nil
}

// enrich attaches the project label and the login state to clients, one batch
// call to `projects` and one to `identity` (skipped when not wanted).
func (s *ClientService) enrich(ctx context.Context, tenantID int64, list []domain.Client, withProject, withLogin bool) ([]domain.ClientRow, error) {
	rows := make([]domain.ClientRow, len(list))
	for i, c := range list {
		rows[i].Client = c
	}
	if len(rows) == 0 {
		return rows, nil
	}
	if withProject {
		seen := map[int64]bool{}
		var projectIDs []int64
		for _, c := range list {
			if !seen[c.ProjectID] {
				seen[c.ProjectID] = true
				projectIDs = append(projectIDs, c.ProjectID)
			}
		}
		labels, err := s.projects.ProjectLabels(ctx, tenantID, projectIDs)
		if err != nil {
			return nil, err
		}
		for i := range rows {
			if l, ok := labels[rows[i].Client.ProjectID]; ok {
				rows[i].ProjectName = l.Name
				d := l.EventDate
				rows[i].ProjectEventDate = &d
			}
		}
	}
	if withLogin {
		ids := make([]string, len(list))
		for i, c := range list {
			ids[i] = formatID(c.ID)
		}
		statuses, err := s.identity.LoginStatuses(ctx, identitycontracts.PrincipalClient, tenantID, ids)
		if err != nil {
			return nil, err
		}
		for i := range rows {
			rows[i].LoginStatus = statuses[formatID(rows[i].Client.ID)]
		}
	}
	return rows, nil
}

func (s *ClientService) Get(ctx context.Context, tenantID, id int64) (*domain.Client, error) {
	c, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperror.NotFound("Client tidak ditemukan")
	}
	return c, nil
}

type CreateClientInput struct {
	ProjectID    int64
	Role         domain.ClientRole
	RelationNote string
	Name         string
	Phone        string
	// Username is optional: empty derives one from Name. The client sets
	// their own password through the invitation link.
	Username string
	Email    string
	// AccessRoleID is optional: 0 uses the default portal role for Role.
	AccessRoleID int64
}

// AccessLinkResult is a client invitation or reset link (/undangan/{Token}).
type AccessLinkResult struct {
	Token     string
	Purpose   string // client_invite or client_reset
	ExpiresAt time.Time
}

// Create validates project_id belongs to the caller's tenant via the
// `projects` module's contract (no cross-module FK — see ADR-0004) and
// provisions a real login credential in the same flow, mirroring how
// `platform`'s tenant registration works (ADR-0008).
func (s *ClientService) Create(ctx context.Context, tenantID int64, input CreateClientInput) (*domain.Client, *AccessLinkResult, error) {
	// Checked here, not left to the column: an unknown value was a 500
	// ("Data truncated"), and the ENUM's case-insensitive match let "bride"
	// through while the login kept the lowercase spelling.
	if !domain.IsClientRole(string(input.Role)) {
		return nil, nil, apperror.Validation("Hubungan client tidak dikenal", map[string][]string{
			"role": {"Pilih salah satu: Pengantin wanita, Pengantin pria, atau Perwakilan keluarga"},
		})
	}
	exists, err := s.projects.ProjectExists(ctx, tenantID, input.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		return nil, nil, apperror.Validation("Project tidak valid", map[string][]string{"projectId": {"Project tidak ditemukan"}})
	}
	if err := s.checkClientQuota(ctx, tenantID); err != nil {
		return nil, nil, err
	}
	input.Username, err = s.freeUsername(ctx, input.Username, input.Name)
	if err != nil {
		return nil, nil, err
	}
	accessRoleID, err := s.portalRole(ctx, tenantID, input.AccessRoleID, input.Role)
	if err != nil {
		return nil, nil, err
	}

	c := &domain.Client{
		TenantID: tenantID, ProjectID: input.ProjectID, Role: input.Role, Username: input.Username, RelationNote: input.RelationNote,
		Name: input.Name, Phone: input.Phone, Email: input.Email, IsActive: true, AccessRoleID: accessRoleID,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, nil, err
	}

	// identity.CreateCredential can fail on its own terms (e.g. username
	// already taken by ANY principal on the platform — usernames aren't
	// tenant-scoped, see identity's credentials.username column) after the
	// client row above already committed. clients and identity own separate
	// tables in separate modules, so there's no single DB transaction to
	// wrap both in (see backend-modular-monolith rule); compensate manually
	// instead, or the client row is left orphaned with no login credential
	// — permanently unable to log in or have "Reset Credential" succeed,
	// since that also resolves through identity by principal.
	if err := s.identity.CreateCredential(ctx, identitycontracts.CreateCredentialInput{
		TenantID: &tenantID, PrincipalType: identitycontracts.PrincipalClient, PrincipalID: formatID(c.ID),
		Username: input.Username, Email: input.Email, Pending: true, Role: string(input.Role), RoleID: accessRoleID, DisplayName: input.Name,
	}); err != nil {
		if delErr := s.repo.Delete(ctx, tenantID, c.ID); delErr != nil {
			logger.Error("failed to roll back orphaned client %d after credential creation failed: %v", c.ID, delErr)
		}
		return nil, nil, err
	}
	link, err := s.identity.IssueAccessLink(ctx, identitycontracts.PrincipalClient, formatID(c.ID))
	if err != nil {
		if delErr := s.identity.DeleteCredential(ctx, identitycontracts.PrincipalClient, formatID(c.ID)); delErr != nil {
			logger.Error("failed to remove login of client %d after invitation failed: %v", c.ID, delErr)
		}
		if delErr := s.repo.Delete(ctx, tenantID, c.ID); delErr != nil {
			logger.Error("failed to roll back client %d after invitation failed: %v", c.ID, delErr)
		}
		return nil, nil, err
	}
	return c, &AccessLinkResult{Token: link.Token, Purpose: link.Purpose, ExpiresAt: link.ExpiresAt}, nil
}

type UpdateContactInput struct {
	Name  string
	Phone string
	Email string
}

func (s *ClientService) UpdateContact(ctx context.Context, tenantID, id int64, input UpdateContactInput) (*domain.Client, error) {
	c, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	previousEmail := c.Email
	c.Name = input.Name
	c.Phone = input.Phone
	c.Email = input.Email
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	if err := s.syncEmail(ctx, c, previousEmail); err != nil {
		return nil, err
	}
	return c, nil
}

// syncEmail keeps the login's email (clients may sign in with it) in step,
// rolling the row back if another login already uses it.
func (s *ClientService) syncEmail(ctx context.Context, c *domain.Client, previousEmail string) error {
	if c.Email == previousEmail {
		return nil
	}
	if err := s.identity.UpdateEmail(ctx, identitycontracts.PrincipalClient, formatID(c.ID), c.Email); err != nil {
		c.Email = previousEmail
		if rollbackErr := s.repo.Update(ctx, c); rollbackErr != nil {
			logger.Error("failed to roll back client %d email after credential sync failed: %v", c.ID, rollbackErr)
		}
		return err
	}
	return nil
}

// Delete soft-deletes a client (ADR-0034): the row disappears from every list,
// count and the project's client slots, and the login is switched off at once (it
// is kept, deactivated, so Restore can bring the client back exactly as they
// were; the username and email stay reserved until the purge removes the login).
// It returns the instant after which Restore is no longer possible. If the login
// cannot be switched off the client is NOT deleted: a deleted client must not be
// able to sign in.
func (s *ClientService) Delete(ctx context.Context, tenantID, id int64, now time.Time) (time.Time, error) {
	c, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return time.Time{}, err
	}
	if err := s.identity.SetActive(ctx, identitycontracts.PrincipalClient, formatID(c.ID), false); err != nil {
		return time.Time{}, err
	}
	ok, err := s.repo.SoftDelete(ctx, tenantID, id, now)
	if err == nil && !ok {
		err = apperror.NotFound("Client tidak ditemukan")
	}
	if err != nil {
		// Put the login back as it was; the row is still live.
		if rbErr := s.identity.SetActive(ctx, identitycontracts.PrincipalClient, formatID(c.ID), c.IsActive); rbErr != nil {
			logger.Error("failed to re-activate the login of client %d after a failed delete: %v", c.ID, rbErr)
		}
		return time.Time{}, err
	}
	return domain.RestorableUntil(now), nil
}

// Restore brings a deleted client back with their login as it was (active again
// only if the client was active when deleted) after re-checking what can now
// fail: the project must still exist (422 `projectId`) and, for an active client,
// the workspace's trial quota must have room (the same 422 as creating one). A
// client that is not deleted, or was deleted longer ago than the retention, is a
// 404.
func (s *ClientService) Restore(ctx context.Context, tenantID, id int64, now time.Time) (*domain.Client, error) {
	c, err := s.repo.FindDeletedByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if c == nil || c.DeletedAt == nil || !domain.CanRestore(*c.DeletedAt, now) {
		return nil, apperror.NotFound("Client tidak ditemukan atau sudah tidak dapat dipulihkan")
	}
	exists, err := s.projects.ProjectExists(ctx, tenantID, c.ProjectID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, apperror.Validation("Project client ini sudah tidak ada", map[string][]string{"projectId": {"Project tidak ditemukan"}})
	}
	if c.IsActive {
		if err := s.checkClientQuota(ctx, tenantID); err != nil {
			return nil, err
		}
	}
	ok, err := s.repo.Restore(ctx, tenantID, id, domain.PurgeCutoff(now))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperror.NotFound("Client tidak ditemukan atau sudah tidak dapat dipulihkan")
	}
	if err := s.identity.SetActive(ctx, identitycontracts.PrincipalClient, formatID(c.ID), c.IsActive); err != nil {
		// Without its login the client would be restored but locked out: undo.
		_, _ = s.repo.SoftDelete(ctx, tenantID, id, *c.DeletedAt)
		return nil, err
	}
	c.DeletedAt = nil
	return c, nil
}

// hardDeleteLogin removes a client's login for good (the purge). The login goes
// before the row and must succeed unless the client never had one; a failure
// leaves the row for the next run, so nothing is half removed.
func (s *ClientService) hardDeleteLogin(ctx context.Context, c domain.Client) error {
	exists, err := s.identity.CredentialExists(ctx, identitycontracts.PrincipalClient, formatID(c.ID))
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	return s.identity.DeleteCredential(ctx, identitycontracts.PrincipalClient, formatID(c.ID))
}

// DeleteAllForProject permanently deletes every client tied to a project, the
// ones waiting in the "deleted" state included — called only by `projects`'
// hard-delete flow (ADR-0013) via the ClientCleaner bridge (see
// projects.module.go's two-phase wiring). Continues past an individual client's
// failure (logged) rather than aborting, same best-effort spirit as before: the
// whole point is clearing out a project being permanently removed.
func (s *ClientService) DeleteAllForProject(ctx context.Context, tenantID, projectID int64) error {
	list, err := s.repo.ListAllByProject(ctx, tenantID, projectID)
	if err != nil {
		return err
	}
	var firstErr error
	for _, c := range list {
		// Remove the login entirely so its username and email are free again;
		// like the old delete, a failure here does not stop the row from going.
		if err := s.identity.DeleteCredential(ctx, identitycontracts.PrincipalClient, formatID(c.ID)); err != nil {
			logger.Error("failed to remove login of client %d before delete: %v", c.ID, err)
		}
		if err := s.repo.Delete(ctx, tenantID, c.ID); err != nil {
			logger.Error("failed to delete client %d while cleaning up deleted project %d: %v", c.ID, projectID, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// PurgeExpired removes the clients that have been deleted for longer than the
// retention, with their logins (which frees the username and email). It returns
// how many were removed; ran is false when another instance holds the purge lock.
// Idempotent. A client whose login cannot be removed is left for the next run.
func (s *ClientService) PurgeExpired(ctx context.Context, now time.Time) (removed int, ran bool, err error) {
	release, ok, err := s.repo.TryLock(ctx)
	if err != nil {
		return 0, false, err
	}
	if !ok {
		return 0, false, nil
	}
	defer release()
	cutoff := domain.PurgeCutoff(now)
	for pass := 0; pass < 50; pass++ {
		list, err := s.repo.ListExpired(ctx, cutoff, 200)
		if err != nil {
			return removed, true, err
		}
		progress := 0
		for _, c := range list {
			if err := s.hardDeleteLogin(ctx, c); err != nil {
				logger.Error("purge: removing the login of client %d: %v", c.ID, err)
				continue
			}
			deleted, err := s.repo.PurgeDelete(ctx, c.TenantID, c.ID, cutoff)
			if err != nil {
				logger.Error("purge: deleting client %d: %v", c.ID, err)
				continue
			}
			if deleted {
				progress++
			}
		}
		removed += progress
		if progress == 0 {
			break
		}
	}
	if removed > 0 {
		logger.Info("purge: removed %d clients", removed)
	}
	return removed, true, nil
}

// ProjectClient is one client of a project as the overview lists them.
type ProjectClient struct {
	ID   int64
	Name string
	Role string
	// HasLogin: the client set a password and can sign in (an invitation not yet
	// accepted does not count).
	HasLogin bool
}

// ListProjectClients names a project's live clients with whether each can sign
// in (one login lookup per client; a project has at most a handful).
func (s *ClientService) ListProjectClients(ctx context.Context, tenantID, projectID int64) ([]ProjectClient, error) {
	list, err := s.repo.ListByProject(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]ProjectClient, 0, len(list))
	for _, c := range list {
		out = append(out, ProjectClient{ID: c.ID, Name: c.Name, Role: string(c.Role), HasLogin: c.IsActive && s.LoginStatus(ctx, c.ID) == "active"})
	}
	return out, nil
}

func (s *ClientService) SetActive(ctx context.Context, tenantID, id int64, isActive bool) (*domain.Client, error) {
	c, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if isActive && !c.IsActive {
		if err := s.checkClientQuota(ctx, tenantID); err != nil {
			return nil, err
		}
	}
	if err := s.repo.SetActive(ctx, tenantID, id, isActive); err != nil {
		return nil, err
	}
	// Keep the login identity in step: before Fase 1 a deactivated client
	// could still log in (only project reads were refused).
	if err := s.identity.SetActive(ctx, identitycontracts.PrincipalClient, formatID(c.ID), isActive); err != nil {
		_ = s.repo.SetActive(ctx, tenantID, id, !isActive)
		return nil, err
	}
	c.IsActive = isActive
	return c, nil
}

// AccessLink issues a one-time link for the client: an invitation while
// they never set a password (7 days), otherwise a password reset (24 hours).
// The WO sends it by WhatsApp; the WO never types or sees the password
// (PLAN.md §3). Earlier links stop working.
func (s *ClientService) AccessLink(ctx context.Context, tenantID, id int64) (*AccessLinkResult, error) {
	c, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if !c.IsActive {
		return nil, apperror.Conflict("Aktifkan client ini dulu sebelum membuat link")
	}
	link, err := s.identity.IssueAccessLink(ctx, identitycontracts.PrincipalClient, formatID(c.ID))
	if err != nil {
		return nil, err
	}
	if link.Purpose == "client_reset" {
		now := time.Now()
		if err := s.repo.SetCredentialResetAt(ctx, tenantID, id, now); err != nil {
			logger.Error("record reset link for client %d: %v", id, err)
		}
	}
	return &AccessLinkResult{Token: link.Token, Purpose: link.Purpose, ExpiresAt: link.ExpiresAt}, nil
}

// LoginStatus is "invited" until the client set their password.
func (s *ClientService) LoginStatus(ctx context.Context, id int64) string {
	st, err := s.identity.LoginStatus(ctx, identitycontracts.PrincipalClient, formatID(id))
	if err != nil {
		logger.Error("login status for client %d: %v", id, err)
		return "unknown"
	}
	return st
}

// freeUsername returns the requested username (validated, must be free) or,
// when none was given, one derived from the client's name.
func (s *ClientService) freeUsername(ctx context.Context, requested, name string) (string, error) {
	if requested != "" {
		if err := validator.Username(requested); err != nil {
			return "", err
		}
		taken, err := s.identity.UsernameTaken(ctx, requested)
		if err != nil {
			return "", err
		}
		if taken {
			return "", apperror.Validation("Nama pengguna sudah dipakai", map[string][]string{"username": {"Nama pengguna ini sudah dipakai, pilih yang lain"}})
		}
		return requested, nil
	}
	base := utils.UsernameBase(name, "client.wo")
	candidate := base
	for attempt := 0; attempt < 8; attempt++ {
		if attempt > 0 {
			candidate = utils.UsernameWithSuffix(base)
		}
		taken, err := s.identity.UsernameTaken(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	return "", apperror.Internal("Gagal membuat nama pengguna. Silakan coba lagi.")
}

// ReplaceRepresentative overwrites the same Family Representative row with
// new contact details — no history is kept (matches the frontend mock's
// existing behavior; see PLAN.md §6.3, an explicitly open decision, not an
// oversight).
//
// The login now belongs to a different person: the previous
// representative's password, sessions and links stop working at once, and
// an invitation link for the new person is returned.
func (s *ClientService) ReplaceRepresentative(ctx context.Context, tenantID, id int64, input UpdateContactInput, relationNote string) (*domain.Client, *AccessLinkResult, error) {
	c, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, nil, err
	}
	if c.Role != domain.RoleFamilyRepresentative {
		return nil, nil, apperror.Forbidden("Hanya Family Representative yang dapat diganti")
	}
	if !c.IsActive {
		return nil, nil, apperror.Conflict("Aktifkan client ini dulu sebelum mengganti perwakilannya")
	}
	// Check what can fail before changing anything, then cut off the
	// previous person first: a later failure must never leave them with
	// access while the card already shows the new person.
	input.Email = strings.TrimSpace(input.Email)
	if input.Email != "" && !strings.EqualFold(input.Email, c.Email) {
		taken, err := s.identity.EmailTaken(ctx, input.Email)
		if err != nil {
			return nil, nil, err
		}
		if taken {
			return nil, nil, apperror.Validation("Email sudah dipakai", map[string][]string{"email": {"Email ini sudah dipakai akun lain"}})
		}
	}
	username, err := s.freeUsername(ctx, "", input.Name)
	if err != nil {
		return nil, nil, err
	}
	if err := s.identity.ResetSignIn(ctx, identitycontracts.PrincipalClient, formatID(c.ID), username); err != nil {
		return nil, nil, err
	}
	previousEmail := c.Email
	c.Name = input.Name
	c.Phone = input.Phone
	c.Email = input.Email
	c.RelationNote = relationNote
	c.Username = username
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, nil, err
	}
	if err := s.repo.UpdateUsername(ctx, tenantID, c.ID, username); err != nil {
		return nil, nil, err
	}
	if err := s.syncEmail(ctx, c, previousEmail); err != nil {
		return nil, nil, err
	}
	link, err := s.identity.IssueAccessLink(ctx, identitycontracts.PrincipalClient, formatID(c.ID))
	if err != nil {
		return nil, nil, err
	}
	return c, &AccessLinkResult{Token: link.Token, Purpose: link.Purpose, ExpiresAt: link.ExpiresAt}, nil
}

// SetRoles completes two-phase wiring with the access module.
func (s *ClientService) SetRoles(r RoleResolver) {
	s.roles = r
}

// portalRole picks the client's portal role: the requested one (validated)
// or the default for their relation.
func (s *ClientService) portalRole(ctx context.Context, tenantID, requested int64, relation domain.ClientRole) (int64, error) {
	if s.roles == nil {
		return 0, apperror.Internal("Peran portal belum dapat diperiksa")
	}
	if requested > 0 {
		if _, err := s.roles.AssignableClientRole(ctx, tenantID, requested); err != nil {
			return 0, err
		}
		return requested, nil
	}
	return s.roles.DefaultClientRole(ctx, tenantID, relation == domain.RoleFamilyRepresentative)
}

// SetAccessRole changes what a client sees in the portal (their portal
// role); the change reaches their sessions at once.
func (s *ClientService) SetAccessRole(ctx context.Context, tenantID, id, roleID int64) (*domain.Client, error) {
	c, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if roleID == c.AccessRoleID {
		return c, nil
	}
	if _, err := s.roles.AssignableClientRole(ctx, tenantID, roleID); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateAccessRole(ctx, tenantID, id, roleID); err != nil {
		return nil, err
	}
	if err := s.identity.UpdateRole(ctx, identitycontracts.PrincipalClient, formatID(c.ID), string(c.Role), roleID); err != nil {
		_ = s.repo.UpdateAccessRole(ctx, tenantID, id, c.AccessRoleID)
		return nil, err
	}
	c.AccessRoleID = roleID
	return c, nil
}

// RoleNames maps role ids to names (nil when unavailable).
func (s *ClientService) RoleNames(ctx context.Context, tenantID int64) map[int64]string {
	if s.roles == nil {
		return nil
	}
	m, err := s.roles.RoleNames(ctx, tenantID)
	if err != nil {
		return nil
	}
	return m
}
