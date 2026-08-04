package utils

import "strings"

// Placeholders returns "?, ?, ?" (n question marks) for building a
// dynamically-sized SQL IN (...) clause with database/sql, which has no
// native slice-expansion like sqlx.In.
func Placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?, ", n-1) + "?"
}

// Int64Args converts a []int64 to []interface{} for QueryContext/ExecContext,
// which take variadic interface{} rather than a typed slice.
func Int64Args(ids []int64) []interface{} {
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return args
}
