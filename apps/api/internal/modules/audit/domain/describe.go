package domain

import (
	"regexp"
	"strings"
)

// Describe turns an audited request into a sentence a person reads in the
// Audit Log ("Menonaktifkan atau mengaktifkan paket #3"), EP-16. It is computed
// when the log is read, from the stored method and path, so rows recorded
// before it existed read the same way. Unknown routes return "" and the
// console falls back to the raw action label, which stays visible to technical
// staff either way.
func Describe(method, path string) string {
	segs, ids := splitPath(path)
	if len(segs) == 0 {
		return ""
	}
	key := strings.ToUpper(method) + " /" + strings.Join(segs, "/")
	if tpl, ok := actionDescriptions[key]; ok {
		return fill(tpl, ids)
	}
	return describeByShape(strings.ToUpper(method), segs, ids)
}

// idSegment matches what audit labels as {id}: numeric ids and payment App ids.
var idSegment = regexp.MustCompile(`^(\d+|app_[0-9a-f]+)$`)

// splitPath strips /api/v1, replaces ids with {id} and returns them in order.
func splitPath(path string) (segs, ids []string) {
	trimmed := strings.Trim(strings.TrimPrefix(path, "/api/v1"), "/")
	if trimmed == "" {
		return nil, nil
	}
	for _, s := range strings.Split(trimmed, "/") {
		if idSegment.MatchString(s) {
			ids = append(ids, s)
			s = "{id}"
		}
		segs = append(segs, s)
	}
	return segs, ids
}

// fill replaces {1}, {2}, … with the path's ids in order.
func fill(tpl string, ids []string) string {
	for i, id := range ids {
		tpl = strings.ReplaceAll(tpl, "{"+string(rune('1'+i))+"}", id)
	}
	return tpl
}

// actionDescriptions covers the Platform Console's own actions exactly (the
// ones the Audit Log is mostly read for). Keys are ActionLabel's shape.
var actionDescriptions = map[string]string{
	// Tenants
	"POST /tenants":                            "Mendaftarkan tenant baru",
	"PATCH /tenants/{id}":                      "Mengubah profil tenant {1}",
	"PUT /tenants/{id}/logo":                   "Mengganti logo tenant {1}",
	"POST /tenants/{id}/toggle-suspension":     "Membekukan atau mengaktifkan kembali tenant {1}",
	"POST /tenants/{id}/owner-invite":          "Mengirim ulang undangan Owner tenant {1}",
	"POST /tenants/{id}/reset-credential":      "Mereset akses masuk Owner tenant {1}",
	"POST /tenants/{id}/activate-subscription": "Mengaktifkan langganan tenant {1} secara manual",
	"GET /tenants/export":                      "Mengunduh daftar tenant (CSV)",
	"GET /audit-log/export":                    "Mengunduh audit log (CSV)",
	// Plans, vouchers, trial limits
	"POST /plans":                    "Membuat paket baru",
	"PATCH /plans/{id}":              "Mengubah paket #{1}",
	"POST /plans/{id}/toggle-active": "Menonaktifkan atau mengaktifkan paket #{1}",
	"POST /vouchers":                 "Membuat voucher baru",
	"PATCH /vouchers/{id}":           "Mengubah voucher #{1}",
	"PUT /trial-limits":              "Mengubah batas masa coba",
	// Platform users
	"POST /platform-admins":                     "Menambah pengguna platform",
	"PATCH /platform-admins/{id}":               "Mengubah pengguna platform #{1}",
	"POST /platform-admins/{id}/toggle-active":  "Menonaktifkan atau mengaktifkan pengguna platform #{1}",
	"POST /platform-admins/{id}/reset-password": "Mereset kata sandi pengguna platform #{1}",
	"POST /platform-admins/{id}/reset-mfa":      "Mereset verifikasi dua langkah pengguna platform #{1}",
	// Payment apps and gateway
	"POST /payment/apps":                    "Mendaftarkan aplikasi baru",
	"POST /payment/apps/{id}/reset-secret":  "Mereset secret aplikasi {1}",
	"POST /payment/apps/{id}/toggle-active": "Menonaktifkan atau mengaktifkan aplikasi {1}",
	"PATCH /payment/gateway-config":         "Mengubah konfigurasi gateway pembayaran",
	// Console back office
	"POST /console/tenants/{id}/notes":          "Menambah catatan untuk tenant {1}",
	"POST /console/tenants/{id}/data-requests":  "Mencatat permintaan data tenant {1}",
	"POST /console/data-requests/{id}/complete": "Menyelesaikan permintaan data #{1}",
	"POST /console/announcements":               "Membuat pengumuman",
	"PATCH /console/announcements/{id}":         "Mengubah pengumuman #{1}",
	"DELETE /console/announcements/{id}":        "Menghapus pengumuman #{1}",
	"PUT /console/subscription-policy":          "Mengubah kebijakan langganan",
	"POST /console/tenants/{id}/impersonate":    "Memulai sesi dukungan di tenant {1}",
	"POST /console/support-sessions/{id}/end":   "Mengakhiri sesi dukungan #{1}",
	"PUT /console/login-slides/{id}":            "Mengganti foto halaman login nomor {1}",
	// Own account
	"POST /auth/logout":      "Keluar",
	"POST /auth/mfa/setup":   "Menyiapkan verifikasi dua langkah",
	"POST /auth/mfa/enable":  "Mengaktifkan verifikasi dua langkah",
	"POST /auth/google/link": "Menautkan akun Google",
	// WO workspace (staff actions recorded in the same log)
	"PATCH /tenants/me/profile":                 "Mengubah profil usaha",
	"PUT /tenants/me/business-details":          "Mengubah data usaha",
	"PUT /tenants/me/logo":                      "Mengganti logo usaha",
	"PUT /tenants/me/support-access":            "Mengizinkan akses dukungan tim ElProof",
	"DELETE /tenants/me/support-access":         "Mencabut akses dukungan tim ElProof",
	"POST /tenants/me/onboarding/dismiss":       "Menutup panduan mulai",
	"POST /subscriptions/pay":                   "Membayar langganan",
	"POST /subscriptions/pending-charge/cancel": "Membatalkan tagihan langganan yang tertunda",
}

// nouns names the resources of the other (mostly WO workspace) routes, for
// the sentence built from the route's shape.
var nouns = map[string]string{
	"tenants": "tenant", "plans": "paket", "vouchers": "voucher", "platform-admins": "pengguna platform",
	"announcements": "pengumuman", "notes": "catatan", "data-requests": "permintaan data",
	"projects": "proyek", "clients": "klien", "staff": "anggota tim", "vendors": "vendor", "venues": "venue",
	"vendor-categories": "kategori vendor", "milestone-templates": "templat jadwal", "milestones": "tahapan",
	"payments": "pembayaran", "client-payments": "pembayaran klien", "documents": "dokumen", "evidence": "bukti",
	"issues": "kendala", "roles": "peran", "invite": "undangan", "uploads": "unggahan",
	"subscription-transactions": "tagihan langganan",
}

// actionVerbs: POST .../<resource>/{id}/<action>.
var actionVerbs = map[string]string{
	"toggle-active":         "Menonaktifkan atau mengaktifkan",
	"toggle-archive":        "Mengarsipkan atau memulihkan",
	"toggle-client-visible": "Mengubah tampilan untuk klien pada",
	"restore":               "Memulihkan",
	"duplicate":             "Menduplikasi",
}

// What a method does to one item, or to a collection.
var itemVerbs = map[string]string{"PATCH": "Mengubah", "PUT": "Mengubah", "DELETE": "Menghapus"}
var collectionVerbs = map[string]string{"POST": "Menambah", "PUT": "Mengubah", "PATCH": "Mengubah", "DELETE": "Menghapus"}

// describeByShape builds "Menambah pembayaran di proyek #12" or "Menghapus
// vendor #4" from a route whose last segment (or the one before its id) is a
// known resource. Anything else (an action route like ".../restore") is "".
func describeByShape(method string, segs, ids []string) string {
	last := len(segs) - 1
	var verb, noun, ownID string
	parentEnd := last // segments before this index may name a parent
	switch {
	case last >= 2 && segs[last-1] == "{id}" && actionVerbs[segs[last]] != "" && method == "POST":
		// ".../vendors/{id}/toggle-active" and the like (also old rows of
		// routes since replaced by PATCH, ADR-0036).
		n, ok := nouns[segs[last-2]]
		if !ok {
			return ""
		}
		verb, noun, ownID = actionVerbs[segs[last]], n, ids[len(ids)-1]
		parentEnd = last - 2
	case segs[last] == "{id}" && last > 0:
		n, ok := nouns[segs[last-1]]
		v, vok := itemVerbs[method]
		if !ok || !vok {
			return ""
		}
		verb, noun, ownID = v, n, ids[len(ids)-1]
		parentEnd = last - 1
	default:
		n, ok := nouns[segs[last]]
		v, vok := collectionVerbs[method]
		if !ok || !vok {
			return ""
		}
		verb, noun = v, n
	}

	out := verb + " " + noun
	if ownID != "" {
		out += " #" + ownID
	}
	// The nearest parent "noun/{id}" before the resource, e.g. projects/12.
	idIndex := 0
	for i := 0; i < parentEnd; i++ {
		if segs[i] == "{id}" {
			idIndex++
		}
	}
	for i := parentEnd - 1; i >= 1; i-- {
		if segs[i] != "{id}" {
			continue
		}
		if parent, ok := nouns[segs[i-1]]; ok {
			out += " di " + parent + " #" + ids[idIndex-1]
		}
		break
	}
	return out
}
