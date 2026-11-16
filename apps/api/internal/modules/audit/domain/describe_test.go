package domain

import (
	"strings"
	"testing"
)

// EP-16: the Audit Log reads as sentences; unknown routes give "" so the
// console shows the raw action instead.
func TestDescribe(t *testing.T) {
	cases := []struct{ method, path, want string }{
		{"POST", "/api/v1/plans/3/toggle-active", "Menonaktifkan atau mengaktifkan paket #3"},
		{"PATCH", "/api/v1/plans/3", "Mengubah paket #3"},
		{"POST", "/api/v1/plans", "Membuat paket baru"},
		{"POST", "/api/v1/tenants/12/toggle-suspension", "Membekukan atau mengaktifkan kembali tenant 12"},
		{"POST", "/api/v1/tenants/12/activate-subscription", "Mengaktifkan langganan tenant 12 secara manual"},
		{"GET", "/api/v1/tenants/export", "Mengunduh daftar tenant (CSV)"},
		{"POST", "/api/v1/payment/apps/app_02ef90c52704/reset-secret", "Mereset secret aplikasi app_02ef90c52704"},
		{"DELETE", "/api/v1/console/announcements/5", "Menghapus pengumuman #5"},
		{"PUT", "/api/v1/console/login-slides/2", "Mengganti foto halaman masuk nomor 2"},
		{"post", "/api/v1/vouchers/", "Membuat voucher baru"}, // method case and a trailing slash do not matter
		// WO workspace routes, by shape.
		{"POST", "/api/v1/projects/12/payments", "Menambah pembayaran di proyek #12"},
		{"PATCH", "/api/v1/projects/12/payments/40", "Mengubah pembayaran #40 di proyek #12"},
		{"DELETE", "/api/v1/vendors/4", "Menghapus vendor #4"},
		{"POST", "/api/v1/vendors", "Menambah vendor"},
		{"POST", "/api/v1/projects/7/evidence/9/toggle-client-visible", "Mengubah tampilan untuk klien pada bukti #9 di proyek #7"},
		{"POST", "/api/v1/vendors/4/toggle-active", "Menonaktifkan atau mengaktifkan vendor #4"}, // a route since replaced; old rows still read well
		// Unknown: fall back to the raw label.
		{"POST", "/api/v1/projects/12/something-new", ""},
		{"POST", "/api/v1/unknown", ""},
		{"POST", "/api/v1", ""},
		{"GET", "/api/v1/projects/12", ""},
	}
	for _, c := range cases {
		if got := Describe(c.method, c.path); got != c.want {
			t.Errorf("%s %s: got %q, want %q", c.method, c.path, got, c.want)
		}
	}
}

// Every exact entry must be reachable: its key has ActionLabel's shape
// (upper-case method, a space, a path starting with "/") and its {n}
// placeholders match the number of {id} segments.
func TestDescribeTableShape(t *testing.T) {
	for key, tpl := range actionDescriptions {
		method, path, ok := strings.Cut(key, " ")
		if !ok || method != strings.ToUpper(method) || !strings.HasPrefix(path, "/") {
			t.Errorf("%q: not \"METHOD /path\"", key)
			continue
		}
		ids := strings.Count(path, "{id}")
		for n := 1; n <= 3; n++ {
			placeholder := "{" + string(rune('0'+n)) + "}"
			if strings.Contains(tpl, placeholder) && n > ids {
				t.Errorf("%q uses %s but the route has %d id(s)", key, placeholder, ids)
			}
		}
	}
}
