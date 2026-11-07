package presentation

import (
	"strconv"
	"time"
)

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// wib is Asia/Jakarta (UTC+7, no daylight saving). ADR-0007: "today" is the
// Indonesian calendar day, whatever timezone the server itself runs in.
var wib = time.FixedZone("WIB", 7*60*60)

func nowWIB() time.Time { return time.Now().In(wib) }
