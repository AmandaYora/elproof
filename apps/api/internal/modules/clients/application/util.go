package application

import (
	"strconv"
	"strings"
)

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func deriveUsername(email string) string {
	local, _, _ := strings.Cut(email, "@")
	return local
}
