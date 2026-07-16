package httpx

import "strings"

// Segments trims a known subtree prefix (registered on net/http.ServeMux with
// a trailing slash) and splits what remains by "/", dropping empty parts.
// Kept dependency-free (no chi/gorilla) per PLAN.md §6's router decision —
// revisit only if this stops being sufficient across modules.
func Segments(path, prefix string) []string {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return nil
	}
	return strings.Split(rest, "/")
}
