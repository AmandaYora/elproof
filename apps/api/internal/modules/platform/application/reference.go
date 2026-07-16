package application

import (
	"strconv"
	"time"
)

// Mirrors the frontend's own reference-generation convention
// (`generatePaymentReference`/`generateGrantReference` in the pre-integration
// mock stores) — last 10 digits of a millisecond timestamp.
func lastDigits(n int64, count int) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= count {
		return s
	}
	return s[len(s)-count:]
}

func generatePaymentReference() string {
	return "INV" + lastDigits(time.Now().UnixMilli(), 10)
}

func generateGrantReference() string {
	return "GRANT" + lastDigits(time.Now().UnixMilli(), 10)
}
