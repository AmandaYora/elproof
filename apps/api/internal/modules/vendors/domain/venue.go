package domain

import "time"

// Venue is its own directory, not a vendor category (see ADR-0016). Unlike
// Vendor, it carries fields meaningful only to a bookable location: a
// mandatory-at-creation city and PIC contact, an official venue phone kept
// distinct from the PIC's personal number, a mandatory-at-creation rental
// price, a separate fixed charge, capacity, free-text facilities/social-media
// blocks, and a single document-or-photo attachment (no photo gallery).
type Venue struct {
	ID                 int64
	TenantID           int64
	Name               string
	PICName            string
	PhonePIC           string
	PhoneVenue         *string
	Email              *string
	Address            *string
	City               *string
	RentalPrice        *int64
	Charge             *int64
	Capacity           *int
	Facilities         *string
	SocialMedia        *string
	Notes              string
	AttachmentPath     *string
	AttachmentMimeType *string
	IsActive           bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
