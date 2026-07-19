package application

import (
	"strconv"
)

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}
