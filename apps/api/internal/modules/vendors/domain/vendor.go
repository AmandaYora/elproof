package domain

import "time"

type Vendor struct {
	ID                 int64
	TenantID           int64
	CategoryID         int64
	Name               string
	PICName            string
	Phone              string
	Email              *string
	SocialMedia        *string
	City               *string
	Address            *string
	PriceAkad          *int64
	PriceAkadResepsi   *int64
	Notes              string
	AttachmentPath     *string
	AttachmentMimeType *string
	IsActive           bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
