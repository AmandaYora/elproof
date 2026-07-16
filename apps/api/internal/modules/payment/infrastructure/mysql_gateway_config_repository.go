package infrastructure

import (
	"context"
	"database/sql"

	"elproof/internal/modules/payment/domain"
)

type MySQLGatewayConfigRepository struct {
	db *sql.DB
}

func NewMySQLGatewayConfigRepository(db *sql.DB) *MySQLGatewayConfigRepository {
	return &MySQLGatewayConfigRepository{db: db}
}

const gatewayConfigColumns = `id, active_provider, is_sandbox, tripay_merchant_code,
	tripay_api_key_encrypted, tripay_private_key_encrypted, created_at, updated_at`

// Get reads the single config row (id=1, seeded by migration 000009 — see
// MODULE_PAYMENT.md §4, single-row-by-convention, not a DB constraint).
func (r *MySQLGatewayConfigRepository) Get(ctx context.Context) (*domain.GatewayConfig, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+gatewayConfigColumns+` FROM payment_gateway_config WHERE id = 1 LIMIT 1`)
	var c domain.GatewayConfig
	var activeProvider, merchantCode, apiKeyEnc, privateKeyEnc sql.NullString
	err := row.Scan(&c.ID, &activeProvider, &c.IsSandbox, &merchantCode, &apiKeyEnc, &privateKeyEnc, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.ActiveProvider = activeProvider.String
	c.TripayMerchantCode = merchantCode.String
	c.TripayAPIKeyEncrypted = apiKeyEnc.String
	c.TripayPrivateKeyEncrypted = privateKeyEnc.String
	return &c, nil
}

func (r *MySQLGatewayConfigRepository) Update(ctx context.Context, c *domain.GatewayConfig) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE payment_gateway_config SET active_provider = ?, is_sandbox = ?, tripay_merchant_code = ?,
		 tripay_api_key_encrypted = ?, tripay_private_key_encrypted = ? WHERE id = 1`,
		nullableString(c.ActiveProvider), c.IsSandbox, nullableString(c.TripayMerchantCode),
		nullableString(c.TripayAPIKeyEncrypted), nullableString(c.TripayPrivateKeyEncrypted),
	)
	return err
}

func nullableString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
