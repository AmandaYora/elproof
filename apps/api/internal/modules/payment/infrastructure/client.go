package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

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

	mu        sync.RWMutex
	consumers map[string]contracts.WebhookConsumer
}

func NewPaymentService(db *sql.DB, encryptionKey string) (*PaymentService, error) {
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
	channels, err := gw.ListChannels(ctx)
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
	channels, err := gw.ListChannels(ctx)
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

	// Uniqueness on order_ref doubles as idempotency for repeat attempts
	// with the same ref — see MODULE_PAYMENT.md §4/§7.3.
	if err := s.dispatchRepo.Create(ctx, &domain.ChargeDispatch{
		OrderRef: orderRef, AppID: appID, ProviderRef: charge.ProviderRef,
	}); err != nil {
		return nil, fmt.Errorf("payment: order_ref %q sudah pernah dipakai: %w", orderRef, err)
	}

	return toChargeResult(charge), nil
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
		// kind=external relay (fire-and-forget, HMAC-signed) is Fase 10 scope.
		return fmt.Errorf("payment: relay webhook ke App eksternal belum diimplementasikan (Fase 10)")
	}
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
	return s.configRepo.Update(ctx, cfg)
}
