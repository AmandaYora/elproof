package presentation

import "strconv"

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// isOwnerOrAdmin gates the payment hard-delete routes (deletePayment/
// deleteClientPayment/deleteVenuePayment) -- broadened from Owner-only to
// Owner-or-Admin per explicit user request, mirroring the "Owner atau
// Admin" bar `vendors`/`venues` already use for their own destructive
// actions (`requireManagerRole`, despite the misleading name, checks exactly
// these two roles) rather than `deleteProject`'s stricter Owner-only bar.
func isOwnerOrAdmin(role string) bool {
	return role == "Owner" || role == "Admin"
}
