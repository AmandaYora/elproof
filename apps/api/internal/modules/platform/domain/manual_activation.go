package domain

import "time"

// ManualActivationReason explains why a subscription was granted without
// going through the payment gateway.
type ManualActivationReason string

const (
	ReasonOfflinePayment ManualActivationReason = "offline_payment" // paid by transfer/invoice outside the gateway
	ReasonSalesDeal      ManualActivationReason = "sales_deal"      // negotiated deal
	ReasonCompensation   ManualActivationReason = "compensation"    // service-disruption compensation
	ReasonTrialExtension ManualActivationReason = "trial_extension" // extended evaluation
	ReasonInternal       ManualActivationReason = "internal"        // internal / demo account
	ReasonOther          ManualActivationReason = "other"           // anything else (note required)
)

// ManualActivationReasonLabels are the Indonesian labels shown in the
// Platform Console and written into the ledger's payment method.
var ManualActivationReasonLabels = map[ManualActivationReason]string{
	ReasonOfflinePayment: "Pembayaran di luar gateway",
	ReasonSalesDeal:      "Kesepakatan sales",
	ReasonCompensation:   "Kompensasi layanan",
	ReasonTrialExtension: "Perpanjangan masa coba",
	ReasonInternal:       "Akun internal/demo",
	ReasonOther:          "Lainnya",
}

// ManualActivation is the record of one manual grant: who, why, with what
// proof, and the ledger reference it produced.
type ManualActivation struct {
	ID                   int64
	TenantID             int64
	PlanID               int64
	ActorAdminID         string
	Reason               ManualActivationReason
	ProofReference       string
	Note                 string
	TransactionReference string
	ExpiresAt            time.Time
	CreatedAt            time.Time
}
