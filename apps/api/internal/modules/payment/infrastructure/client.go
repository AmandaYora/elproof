package infrastructure

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-sql-driver/mysql"

	identitycontracts "elproof/internal/modules/identity/contracts"
	"elproof/internal/modules/payment/contracts"
	"elproof/internal/modules/payment/domain"
	"elproof/internal/shared/apperror"
)

// PaymentService is the single concrete type backing both `contracts.Client`
// (called by App internal, in-process) and `contracts.Dispatcher` (called
// only by this module's own composition-root wiring and presentation
// layer) — see MODULE_PAYMENT.md §3's design note: this module has no
// `application/` layer of its own, since it owns no business process to
// separate from its infrastructure.
type PaymentService struct {
	configRepo   *MySQLGatewayConfigRepository
	appRepo      *MySQLAppRepository
	dispatchRepo *MySQLDispatchRepository
	webhookRepo  *MySQLWebhookLogRepository
	cryptor      *cryptor

	// identity mints bearer tokens for external Apps (Fase 10) — `payment`
	// verifies the appId+secret itself (its own registry), identity only
	// ever signs the resulting JWT. One-way dependency, same shape as
	// `vendors -> projects` (see knowledge/MODULE_MAP.md).
	identity    identitycontracts.Contracts
	appTokenTTL time.Duration

	mu        sync.RWMutex
	consumers map[string]contracts.WebhookConsumer

	// channelCache holds the gateway's own channel list (fees, active state)
	// for channelCacheTTL — this rarely changes, so avoiding a live gateway
	// round trip on every single ListChannels/QuoteFee call is a clear win.
	// Cleared by UpdateConfig so a provider/mode switch is never served stale
	// channels from the previous provider.
	channelCacheMu sync.RWMutex
	channelCache   []domain.Channel
	channelCacheAt time.Time
}

// channelCacheTTL bounds how stale the cached channel list can be.
const channelCacheTTL = 5 * time.Minute

func NewPaymentService(db *sql.DB, encryptionKey string, identity identitycontracts.Contracts, appTokenTTL time.Duration) (*PaymentService, error) {
	c, err := newCryptor(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("payment: %w", err)
	}
	return &PaymentService{
		configRepo:   NewMySQLGatewayConfigRepository(db),
		appRepo:      NewMySQLAppRepository(db),
		dispatchRepo: NewMySQLDispatchRepository(db),
		webhookRepo:  NewMySQLWebhookLogRepository(db),
		cryptor:      c,
		identity:     identity,
		appTokenTTL:  appTokenTTL,
		consumers:    make(map[string]contracts.WebhookConsumer),
	}, nil
}

// EnsureInternalApp bootstraps the fixed internal App row this module needs
// on every startup — see MySQLAppRepository.EnsureInternalApp.
func (s *PaymentService) EnsureInternalApp(ctx context.Context, appID, name string) error {
	return s.appRepo.EnsureInternalApp(ctx, appID, name)
}

// loadGateway reads the config fresh on every call — no cache, per
// MODULE_PAYMENT.md §8 ("tanpa cache pada pemeriksaan status/deaktivasi").
func (s *PaymentService) loadGateway(ctx context.Context) (gateway, *domain.GatewayConfig, error) {
	cfg, err := s.configRepo.Get(ctx)
	if err != nil {
		return nil, nil, err
	}
	if cfg == nil || !cfg.Enabled() {
		return nil, cfg, nil
	}
	switch cfg.ActiveProvider {
	case "tripay":
		apiKey, err := s.cryptor.decrypt(cfg.TripayAPIKeyEncrypted)
		if err != nil {
			return nil, nil, fmt.Errorf("payment: gagal mendekripsi kredensial tripay: %w", err)
		}
		privateKey, err := s.cryptor.decrypt(cfg.TripayPrivateKeyEncrypted)
		if err != nil {
			return nil, nil, fmt.Errorf("payment: gagal mendekripsi kredensial tripay: %w", err)
		}
		return newTripayGateway(cfg.TripayMerchantCode, apiKey, privateKey, cfg.IsSandbox), cfg, nil
	default:
		return nil, cfg, fmt.Errorf("payment: provider %q tidak dikenali", cfg.ActiveProvider)
	}
}

func (s *PaymentService) Enabled(ctx context.Context) (bool, error) {
	_, cfg, err := s.loadGateway(ctx)
	if err != nil {
		return false, err
	}
	return cfg != nil && cfg.Enabled(), nil
}

func (s *PaymentService) ListChannels(ctx context.Context) ([]contracts.Channel, error) {
	gw, _, err := s.loadGateway(ctx)
	if err != nil {
		return nil, err
	}
	if gw == nil {
		return nil, apperror.Validation("Gateway pembayaran belum dikonfigurasi", nil)
	}
	channels, err := s.loadChannels(ctx, gw)
	if err != nil {
		return nil, err
	}
	result := make([]contracts.Channel, 0, len(channels))
	for _, c := range channels {
		result = append(result, contracts.Channel{
			Code: c.Code, Name: c.Name, Type: c.Type,
			FeeCustomer: c.FeeCustomerFlat, FeeMerchant: c.FeeMerchantFlat,
			IconURL: c.IconURL, Active: c.Active,
		})
	}
	return result, nil
}

func (s *PaymentService) QuoteFee(ctx context.Context, channel string, amount int64) (int64, error) {
	gw, _, err := s.loadGateway(ctx)
	if err != nil {
		return 0, err
	}
	if gw == nil {
		return 0, apperror.Validation("Gateway pembayaran belum dikonfigurasi", nil)
	}
	channels, err := s.loadChannels(ctx, gw)
	if err != nil {
		return 0, err
	}
	for _, c := range channels {
		if c.Code == channel {
			return c.QuoteFee(amount), nil
		}
	}
	return 0, apperror.NotFound("Kanal pembayaran tidak ditemukan")
}

// loadChannels returns the gateway's channel list, cached for channelCacheTTL
// — this data (fees, active state) rarely changes, so a live gateway round
// trip on every single call is avoidable latency. Shared by ListChannels and
// QuoteFee so both benefit from one cache instead of each hitting the
// gateway independently.
func (s *PaymentService) loadChannels(ctx context.Context, gw gateway) ([]domain.Channel, error) {
	s.channelCacheMu.RLock()
	fresh := !s.channelCacheAt.IsZero() && time.Since(s.channelCacheAt) < channelCacheTTL
	cached := s.channelCache
	s.channelCacheMu.RUnlock()
	if fresh {
		return cached, nil
	}

	channels, err := gw.ListChannels(ctx)
	if err != nil {
		return nil, err
	}

	s.channelCacheMu.Lock()
	s.channelCache = channels
	s.channelCacheAt = time.Now()
	s.channelCacheMu.Unlock()

	return channels, nil
}

const defaultChargeChannel = "QRIS"

func (s *PaymentService) CreateCharge(ctx context.Context, appID, contextID, orderRef string, amount int64) (*contracts.ChargeResult, error) {
	return s.CreateChannelCharge(ctx, appID, contextID, orderRef, amount, defaultChargeChannel, contracts.ChargeOptions{})
}

func (s *PaymentService) CreateChannelCharge(
	ctx context.Context,
	appID, contextID, orderRef string,
	amount int64,
	channel string,
	opts contracts.ChargeOptions,
) (*contracts.ChargeResult, error) {
	// Check order_ref uniqueness before anything else — see
	// MODULE_PAYMENT.md §4/§7.3. This is orthogonal to whether a gateway is
	// even configured, so a repeat attempt gets a clean 409 regardless
	// (including in simulation mode), without a second real charge ever
	// being created upstream.
	existing, err := s.dispatchRepo.FindByOrderRef(ctx, orderRef)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperror.Conflict(fmt.Sprintf("order_ref %q sudah pernah dipakai", orderRef))
	}

	gw, _, err := s.loadGateway(ctx)
	if err != nil {
		return nil, err
	}
	if gw == nil {
		return nil, apperror.Validation("Gateway pembayaran belum dikonfigurasi", nil)
	}

	charge, err := gw.CreateCharge(ctx, ChargeRequest{
		OrderRef: orderRef, Amount: amount, Channel: channel,
		CustomerName: opts.CustomerName, CustomerEmail: opts.CustomerEmail, CustomerPhone: opts.CustomerPhone,
	})
	if err != nil {
		return nil, err
	}

	if err := s.dispatchRepo.Create(ctx, &domain.ChargeDispatch{
		OrderRef: orderRef, AppID: appID, ProviderRef: charge.ProviderRef, ExpiresAt: charge.ExpiresAt,
	}); err != nil {
		// Race with another request for the same order_ref that won between
		// our pre-check and this insert — the DB's own uniqueness is the
		// real guarantee, the pre-check above is just the common-case
		// optimization that avoids a wasted gateway call.
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, apperror.Conflict(fmt.Sprintf("order_ref %q sudah pernah dipakai", orderRef))
		}
		return nil, fmt.Errorf("payment: gagal menyimpan dispatch order_ref %q: %w", orderRef, err)
	}

	return toChargeResult(charge), nil
}

// CheckStatusForApp is CheckStatus scoped to charges the calling App actually
// owns — used by the external-mode status route (Fase 10) so one App can
// never poll another App's order_ref (see knowledge/MODULE_PAYMENT.md §7.2).
func (s *PaymentService) CheckStatusForApp(ctx context.Context, appID, orderRef string) (*contracts.ChargeResult, error) {
	dispatch, err := s.dispatchRepo.FindByOrderRef(ctx, orderRef)
	if err != nil {
		return nil, err
	}
	if dispatch == nil || dispatch.AppID != appID {
		return nil, apperror.NotFound("order_ref tidak ditemukan")
	}
	return s.CheckStatus(ctx, dispatch.ProviderRef)
}

func (s *PaymentService) CheckStatus(ctx context.Context, providerRef string) (*contracts.ChargeResult, error) {
	gw, _, err := s.loadGateway(ctx)
	if err != nil {
		return nil, err
	}
	if gw == nil {
		return nil, apperror.Validation("Gateway pembayaran belum dikonfigurasi", nil)
	}
	charge, err := gw.GetChargeStatus(ctx, providerRef)
	if err != nil {
		return nil, err
	}
	return toChargeResult(charge), nil
}

func toChargeResult(c *domain.Charge) *contracts.ChargeResult {
	return &contracts.ChargeResult{
		OrderRef: c.OrderRef, ProviderRef: c.ProviderRef, Channel: c.Channel,
		QRImageURL: c.QRImageURL, PayCode: c.PayCode, CheckoutURL: c.CheckoutURL,
		Amount: c.Amount, FeeAmount: c.FeeAmount, ExpiresAt: c.ExpiresAt, Status: string(c.Status),
	}
}

// --- Dispatcher (composition root + this module's own presentation layer only) ---

func (s *PaymentService) RegisterConsumer(appID string, consumer contracts.WebhookConsumer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.consumers[appID] = consumer
}

// HandleWebhook verifies the signature, checks idempotency, resolves which
// App owns the order ref, and dispatches to that App's registered consumer —
// see MODULE_PAYMENT.md §6 step 5. Only `kind=internal` dispatch is
// implemented so far (Fase 9); `kind=external` webhook relay is Fase 10.
func (s *PaymentService) HandleWebhook(ctx context.Context, signature string, rawBody []byte) error {
	gw, _, err := s.loadGateway(ctx)
	if err != nil {
		return err
	}
	if gw == nil {
		return apperror.Validation("Gateway pembayaran belum dikonfigurasi", nil)
	}

	event, eventID, err := gw.VerifyAndParseWebhook(signature, rawBody)
	if err != nil {
		return apperror.Unauthorized("Signature webhook tidak valid")
	}

	firstTime, err := s.webhookRepo.MarkSeen(ctx, gw.Name(), eventID)
	if err != nil {
		return err
	}
	if !firstTime {
		return nil // already processed — idempotent no-op, not an error
	}

	dispatch, err := s.dispatchRepo.FindByOrderRef(ctx, event.OrderRef)
	if err != nil {
		return err
	}
	if dispatch == nil {
		return apperror.NotFound("order_ref pada webhook tidak dikenali")
	}

	app, err := s.appRepo.FindByAppID(ctx, dispatch.AppID)
	if err != nil {
		return err
	}
	if app == nil || !app.IsActive {
		return apperror.NotFound("App pemilik charge ini tidak ditemukan atau nonaktif")
	}

	// Only a non-terminal status (still "unpaid") leaves the dispatch open —
	// anything else is this charge's final word, whether it arrives here via
	// webhook or is discovered later by the reconciliation sweep polling
	// Tripay directly. Claiming resolution here (not just relying on the
	// webhook-event idempotency log above) is what stops the SAME underlying
	// charge being dispatched twice if the sweep and a delayed webhook both
	// end up reporting the same outcome — see MySQLDispatchRepository.ClaimUnresolved.
	if event.Status != domain.ChargeUnpaid {
		won, err := s.dispatchRepo.ClaimUnresolved(ctx, event.OrderRef)
		if err != nil {
			return err
		}
		if !won {
			return nil // reconciliation sweep (or an earlier webhook) already dispatched this outcome
		}
	}

	consumerEvent := contracts.WebhookEvent{
		ProviderRef: event.ProviderRef, OrderRef: event.OrderRef,
		Paid: event.Paid(), Amount: event.Amount, PaidAt: event.PaidAt,
	}

	switch app.Kind {
	case domain.AppKindInternal:
		s.mu.RLock()
		consumer, ok := s.consumers[app.AppID]
		s.mu.RUnlock()
		if !ok {
			return fmt.Errorf("payment: tidak ada consumer terdaftar untuk App internal %q", app.AppID)
		}
		return consumer.ApplyWebhookEvent(ctx, event.OrderRef, consumerEvent)
	default:
		s.relayWebhookToExternalApp(app, consumerEvent)
		return nil
	}
}

// externalWebhookPayload is the fixed JSON shape relayed to an external
// App's callback_url — see knowledge/MODULE_PAYMENT.md §7 relay note.
type externalWebhookPayload struct {
	OrderRef string `json:"orderRef"`
	Paid     bool   `json:"paid"`
	Amount   int64  `json:"amount"`
	PaidAt   string `json:"paidAt,omitempty"`
}

// relayWebhookToExternalApp signs and POSTs one fire-and-forget attempt to an
// external App's callback_url — no retry, short timeout. Genuinely
// asynchronous: the network call runs in its own goroutine so neither
// Tripay's webhook response (HandleWebhook) nor the reconciliation sweep's
// throughput (reconcileOne) ever blocks on a third party's server — a slow
// or unreachable external App must never add its own latency to either of
// those. Deliberately detached from the caller's context (`context.Background()`,
// not the caller's `ctx`): this goroutine outlives the request/sweep-tick
// that launched it, so a context tied to that would already be cancelled
// before the relay even has a chance to complete. A missed relay (network
// blip, App down) is recovered by the App itself polling
// `GET /external/payments/charges/{orderRef}/status` (see §7.2) — errors
// here are only ever logged, never surfaced to the caller.
func (s *PaymentService) relayWebhookToExternalApp(app *domain.App, event contracts.WebhookEvent) {
	go func() {
		if app.CallbackURL == "" {
			log.Printf("payment: App eksternal %q tidak punya callback_url, relay dilewati", app.AppID)
			return
		}
		secret, err := s.cryptor.decrypt(app.SecretEncrypted)
		if err != nil {
			log.Printf("payment: gagal mendekripsi secret App %q untuk relay: %v", app.AppID, err)
			return
		}

		var paidAt string
		if !event.PaidAt.IsZero() {
			paidAt = event.PaidAt.Format(time.RFC3339)
		}
		payload, err := json.Marshal(externalWebhookPayload{
			OrderRef: event.OrderRef, Paid: event.Paid, Amount: event.Amount, PaidAt: paidAt,
		})
		if err != nil {
			log.Printf("payment: gagal membuat payload relay untuk App %q: %v", app.AppID, err)
			return
		}

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		signature := hex.EncodeToString(mac.Sum(nil))

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, app.CallbackURL, bytes.NewReader(payload))
		if err != nil {
			log.Printf("payment: gagal menyusun request relay ke App %q: %v", app.AppID, err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Webhook-Signature", signature)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("payment: relay webhook ke App eksternal %q (%s) gagal, App diharapkan polling status: %v", app.AppID, app.CallbackURL, err)
			return
		}
		defer resp.Body.Close()
	}()
}

// --- Reconciliation (safety net for a charge whose webhook was never received) ---

const (
	// reconcileMinAge keeps a charge out of the sweep until it's had a
	// realistic chance to be paid — no point re-checking something created
	// seconds ago.
	reconcileMinAge = 2 * time.Minute
	// reconcileBatchLimit caps how many dispatches one sweep tick re-checks
	// against the gateway, so a large backlog can't turn one tick into a
	// long-running burst of outbound requests.
	reconcileBatchLimit = 50
	// reconcileExpiryGrace is added on top of a charge's own expiresAt
	// before it's force-resolved locally as expired even without a
	// definitive answer from the gateway — generous buffer in case the
	// gateway's own status update itself lags a bit past nominal expiry.
	reconcileExpiryGrace = 1 * time.Hour
	// reconcileMaxAgeFallback is the deadline used for dispatch rows that
	// predate the expires_at column (NULL) — so nothing pre-existing is left
	// open forever just because it wasn't recorded.
	reconcileMaxAgeFallback = 24 * time.Hour
	// reconcileConcurrency bounds how many dispatches are checked against the
	// gateway at once within a single sweep tick — keeps a full batch from
	// running as `reconcileBatchLimit` sequential round trips.
	reconcileConcurrency = 10
)

// StartReconciler runs ReconcilePending on a fixed interval until ctx is
// cancelled. Errors are logged, never fatal — a bad tick just retries next
// interval. See knowledge/MODULE_PAYMENT.md's reconciliation section.
func (s *PaymentService) StartReconciler(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				checked, resolved, err := s.ReconcilePending(ctx)
				if err != nil {
					log.Printf("payment: reconciliation sweep gagal: %v", err)
					continue
				}
				if checked > 0 {
					log.Printf("payment: reconciliation sweep — %d diperiksa, %d diselesaikan", checked, resolved)
				}
			}
		}
	}()
}

// ReconcilePending is the safety net for a charge whose webhook was never
// received (network blip, an external App's endpoint down at delivery time,
// etc.): it re-checks every still-open dispatch directly against the
// gateway and dispatches a terminal result exactly the way HandleWebhook
// would — same consumer/relay code path, so downstream behavior is
// identical regardless of how the outcome was discovered.
func (s *PaymentService) ReconcilePending(ctx context.Context) (checked int, resolved int, err error) {
	pending, err := s.dispatchRepo.ListUnresolvedForSweep(ctx, time.Now().Add(-reconcileMinAge), reconcileBatchLimit)
	if err != nil {
		return 0, 0, err
	}

	// Bounded fan-out, not sequential: each iteration's dominant cost is a
	// live network round trip to the gateway (CheckStatus), so checking a
	// full batch one at a time could turn a single sweep tick into minutes
	// under a real backlog. reconcileConcurrency caps how many of those
	// round trips are in flight at once — ClaimUnresolved's atomic UPDATE is
	// what makes this safe (it's the same guarantee that already protects a
	// concurrent webhook delivery racing this same sweep).
	sem := make(chan struct{}, reconcileConcurrency)
	var wg sync.WaitGroup
	var resolvedCount int64
	for _, dispatch := range pending {
		wg.Add(1)
		sem <- struct{}{}
		go func(d domain.ChargeDispatch) {
			defer wg.Done()
			defer func() { <-sem }()
			if s.reconcileOne(ctx, d) {
				atomic.AddInt64(&resolvedCount, 1)
			}
		}(dispatch)
	}
	wg.Wait()

	return len(pending), int(resolvedCount), nil
}

// reconcileOne handles a single dispatch, returning whether it ended up
// resolved (a terminal event was dispatched) this round.
func (s *PaymentService) reconcileOne(ctx context.Context, dispatch domain.ChargeDispatch) bool {
	deadline := dispatch.ExpiresAt.Add(reconcileExpiryGrace)
	if dispatch.ExpiresAt.IsZero() {
		// Pre-existing rows from before expires_at was tracked — fall back
		// to a generous max age from creation instead of never expiring.
		deadline = dispatch.CreatedAt.Add(reconcileMaxAgeFallback)
	}

	result, statusErr := s.CheckStatus(ctx, dispatch.ProviderRef)

	var event contracts.WebhookEvent
	switch {
	case statusErr == nil && result.Status != string(domain.ChargeUnpaid):
		// The gateway gave us a definitive terminal answer.
		event = contracts.WebhookEvent{
			OrderRef: dispatch.OrderRef, ProviderRef: dispatch.ProviderRef,
			Paid: result.Status == string(domain.ChargePaid), Amount: result.Amount, PaidAt: time.Now(),
		}
	case time.Now().After(deadline):
		// Still no definitive answer (unpaid, or the status check itself
		// failed) long past its own expiry — force-resolve as expired
		// locally. The gateway's real state stops mattering practically once
		// the payment window has been closed this long.
		if statusErr != nil {
			log.Printf("payment: reconciliation — order_ref %q gagal dicek (%v), dipaksa expired karena sudah lewat masa berlaku", dispatch.OrderRef, statusErr)
		}
		event = contracts.WebhookEvent{OrderRef: dispatch.OrderRef, ProviderRef: dispatch.ProviderRef, Paid: false}
	default:
		// Still legitimately open — leave it, check again next sweep.
		return false
	}

	won, err := s.dispatchRepo.ClaimUnresolved(ctx, dispatch.OrderRef)
	if err != nil {
		log.Printf("payment: reconciliation — gagal klaim order_ref %q: %v", dispatch.OrderRef, err)
		return false
	}
	if !won {
		return false // a webhook resolved it in the meantime — not our job anymore
	}

	app, err := s.appRepo.FindByAppID(ctx, dispatch.AppID)
	if err != nil || app == nil {
		log.Printf("payment: reconciliation — App %q untuk order_ref %q tidak ditemukan: %v", dispatch.AppID, dispatch.OrderRef, err)
		return true
	}

	switch app.Kind {
	case domain.AppKindInternal:
		s.mu.RLock()
		consumer, ok := s.consumers[app.AppID]
		s.mu.RUnlock()
		if !ok {
			log.Printf("payment: reconciliation — tidak ada consumer terdaftar untuk App internal %q", app.AppID)
			return true
		}
		if err := consumer.ApplyWebhookEvent(ctx, dispatch.OrderRef, event); err != nil {
			log.Printf("payment: reconciliation — consumer App %q gagal memproses order_ref %q: %v", app.AppID, dispatch.OrderRef, err)
		}
	default:
		s.relayWebhookToExternalApp(app, event)
	}
	return true
}

// --- Admin CRUD (called directly by this module's own presentation layer) ---

type GatewayConfigView struct {
	ActiveProvider     string
	IsSandbox          bool
	TripayMerchantCode string
	HasTripayAPIKey    bool
	HasTripayPrivateKey bool
}

func (s *PaymentService) GetConfig(ctx context.Context) (*GatewayConfigView, error) {
	cfg, err := s.configRepo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return &GatewayConfigView{}, nil
	}
	return &GatewayConfigView{
		ActiveProvider:      cfg.ActiveProvider,
		IsSandbox:           cfg.IsSandbox,
		TripayMerchantCode:  cfg.TripayMerchantCode,
		HasTripayAPIKey:     cfg.TripayAPIKeyEncrypted != "",
		HasTripayPrivateKey: cfg.TripayPrivateKeyEncrypted != "",
	}, nil
}

type UpdateConfigInput struct {
	ActiveProvider     string
	IsSandbox          bool
	TripayMerchantCode string
	// TripayAPIKey/TripayPrivateKey are write-only — an empty string means
	// "leave the existing stored value unchanged" (see MODULE_PAYMENT.md §8:
	// submit without retyping must not erase the old value).
	TripayAPIKey     string
	TripayPrivateKey string
}

func (s *PaymentService) UpdateConfig(ctx context.Context, input UpdateConfigInput) error {
	existing, err := s.configRepo.Get(ctx)
	if err != nil {
		return err
	}
	cfg := &domain.GatewayConfig{
		ActiveProvider:     input.ActiveProvider,
		IsSandbox:          input.IsSandbox,
		TripayMerchantCode: input.TripayMerchantCode,
	}
	if existing != nil {
		cfg.TripayAPIKeyEncrypted = existing.TripayAPIKeyEncrypted
		cfg.TripayPrivateKeyEncrypted = existing.TripayPrivateKeyEncrypted
	}
	if input.TripayAPIKey != "" {
		enc, err := s.cryptor.encrypt(input.TripayAPIKey)
		if err != nil {
			return err
		}
		cfg.TripayAPIKeyEncrypted = enc
	}
	if input.TripayPrivateKey != "" {
		enc, err := s.cryptor.encrypt(input.TripayPrivateKey)
		if err != nil {
			return err
		}
		cfg.TripayPrivateKeyEncrypted = enc
	}
	if err := s.configRepo.Update(ctx, cfg); err != nil {
		return err
	}
	// A provider/mode switch must never keep serving the previous provider's
	// cached channel list — clear it so the next ListChannels/QuoteFee call
	// fetches fresh from whatever gateway is now active.
	s.channelCacheMu.Lock()
	s.channelCacheAt = time.Time{}
	s.channelCacheMu.Unlock()
	return nil
}

// --- App registry (Fase 10 — Platform Console "Manajemen Aplikasi") ---

// AppView is the safe-to-display shape of an App row — never the secret
// hash/encrypted copy, see knowledge/MODULE_PAYMENT.md §7.5.
type AppView struct {
	AppID       string
	Name        string
	Kind        string
	CallbackURL string
	IsActive    bool
	CreatedAt   time.Time
}

func toAppView(a domain.App) AppView {
	return AppView{
		AppID: a.AppID, Name: a.Name, Kind: string(a.Kind),
		CallbackURL: a.CallbackURL, IsActive: a.IsActive, CreatedAt: a.CreatedAt,
	}
}

func (s *PaymentService) ListApps(ctx context.Context) ([]AppView, error) {
	apps, err := s.appRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]AppView, 0, len(apps))
	for _, a := range apps {
		result = append(result, toAppView(a))
	}
	return result, nil
}

func (s *PaymentService) GetApp(ctx context.Context, appID string) (*AppView, error) {
	app, err := s.appRepo.FindByAppID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, apperror.NotFound("Aplikasi tidak ditemukan")
	}
	view := toAppView(*app)
	return &view, nil
}

// IsAppActive is checked fresh on every external-mode request (never cached
// off the JWT) — deactivating an App must immediately reject its in-flight
// token, not wait for expiry. See knowledge/MODULE_PAYMENT.md §7.1.
func (s *PaymentService) IsAppActive(ctx context.Context, appID string) (bool, error) {
	app, err := s.appRepo.FindByAppID(ctx, appID)
	if err != nil {
		return false, err
	}
	return app != nil && app.IsActive, nil
}

// CreateExternalApp registers a new external App and returns its plaintext
// secret exactly once — the caller (Platform Console) must display it now;
// it can never be retrieved again, only reset.
func (s *PaymentService) CreateExternalApp(ctx context.Context, name, callbackURL string) (appID, secret string, err error) {
	fields := map[string][]string{}
	if name == "" {
		fields["name"] = []string{"Nama aplikasi wajib diisi"}
	}
	if callbackURL == "" {
		fields["callbackUrl"] = []string{"URL callback wajib diisi"}
	}
	if len(fields) > 0 {
		return "", "", apperror.Validation("Data aplikasi tidak valid", fields)
	}

	appID, err = generateAppID()
	if err != nil {
		return "", "", err
	}
	secret, err = generateSecret()
	if err != nil {
		return "", "", err
	}
	hash, err := hashSecret(secret)
	if err != nil {
		return "", "", err
	}
	encrypted, err := s.cryptor.encrypt(secret)
	if err != nil {
		return "", "", err
	}

	app := &domain.App{
		AppID: appID, Name: name, Kind: domain.AppKindExternal,
		SecretHash: hash, SecretEncrypted: encrypted, CallbackURL: callbackURL, IsActive: true,
	}
	if err := s.appRepo.Create(ctx, app); err != nil {
		return "", "", err
	}
	return appID, secret, nil
}

// ResetAppSecret replaces an external App's secret — like CreateExternalApp,
// the plaintext is returned exactly once.
func (s *PaymentService) ResetAppSecret(ctx context.Context, appID string) (secret string, err error) {
	app, err := s.appRepo.FindByAppID(ctx, appID)
	if err != nil {
		return "", err
	}
	if app == nil || app.Kind != domain.AppKindExternal {
		return "", apperror.NotFound("Aplikasi eksternal tidak ditemukan")
	}

	secret, err = generateSecret()
	if err != nil {
		return "", err
	}
	hash, err := hashSecret(secret)
	if err != nil {
		return "", err
	}
	encrypted, err := s.cryptor.encrypt(secret)
	if err != nil {
		return "", err
	}
	if err := s.appRepo.SetSecret(ctx, appID, hash, encrypted); err != nil {
		return "", err
	}
	return secret, nil
}

// SetAppStatus enables/disables an external App. The internal App
// (`platform-billing`) is never toggleable through this admin path — turning
// it off would sever ElProof's own subscription billing.
func (s *PaymentService) SetAppStatus(ctx context.Context, appID string, isActive bool) error {
	app, err := s.appRepo.FindByAppID(ctx, appID)
	if err != nil {
		return err
	}
	if app == nil {
		return apperror.NotFound("Aplikasi tidak ditemukan")
	}
	if app.Kind == domain.AppKindInternal {
		return apperror.Forbidden("Aplikasi internal tidak dapat dinonaktifkan lewat halaman ini")
	}
	return s.appRepo.SetActive(ctx, appID, isActive)
}

// IssueAppToken verifies an external App's appId+secret against this
// module's own registry, then asks identity to sign an access token for it —
// see knowledge/MODULE_PAYMENT.md §7.1. No refresh token: the App simply
// repeats this exchange once its token expires.
func (s *PaymentService) IssueAppToken(ctx context.Context, appID, secret string) (token string, expiresAt time.Time, err error) {
	app, err := s.appRepo.FindByAppID(ctx, appID)
	if err != nil {
		return "", time.Time{}, err
	}
	if app == nil || app.Kind != domain.AppKindExternal || !app.IsActive || app.SecretHash == "" {
		return "", time.Time{}, apperror.Unauthorized("appId atau secret tidak valid")
	}
	if err := compareSecret(app.SecretHash, secret); err != nil {
		return "", time.Time{}, apperror.Unauthorized("appId atau secret tidak valid")
	}

	token, err = s.identity.IssueServiceToken(ctx, "app", app.AppID, s.appTokenTTL)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, time.Now().Add(s.appTokenTTL), nil
}
