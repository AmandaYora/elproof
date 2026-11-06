// Package domain holds the access module's vocabulary: the fixed permission
// catalog, the built-in role templates and the Role entity (ADR-0026).
package domain

// Kind is who a role is for.
type Kind string

const (
	KindStaff  Kind = "staff"
	KindClient Kind = "client"
)

// Scope is which projects a staff role's project permissions reach.
type Scope string

const (
	ScopeAll      Scope = "all"      // every project of the workspace
	ScopeAssigned Scope = "assigned" // only projects where the person is PIC or PIC Sales
)

// Permission is one entry of the fixed catalog. Keys are stable API
// identifiers; labels and descriptions are what WOs read on the
// "Peran & Akses" page.
type Permission struct {
	Key         string
	Kind        Kind
	Group       string
	Label       string
	Description string
}

// Catalog is every permission a role can hold. Adding a key here (and
// checking it in the owning module) is the only way to add a permission.
var Catalog = []Permission{
	{"dashboard.view", KindStaff, "Dasbor", "Lihat dasbor", "Membuka Dasbor dengan ringkasan seluruh project. Butuh cakupan semua project."},
	{"dashboard.revenue.view", KindStaff, "Dasbor", "Lihat omzet", "Melihat angka omzet dan nilai kontrak di Dasbor."},

	{"project.create", KindStaff, "Project", "Buat project", "Membuat project baru. Dengan cakupan 'project saya', pembuatnya otomatis menjadi PIC Sales."},
	{"project.edit", KindStaff, "Project", "Ubah project", "Mengubah data, status, timeline, vendor, kendala, dan dokumen project."},
	{"project.assign_pic", KindStaff, "Project", "Tentukan PIC", "Mengganti PIC dan PIC Sales sebuah project."},
	{"project.duplicate", KindStaff, "Project", "Duplikat project", "Membuat project baru dari project lain sebagai templat."},
	{"project.archive", KindStaff, "Project", "Arsipkan project", "Mengarsipkan dan memulihkan project."},
	{"project.delete", KindStaff, "Project", "Hapus project permanen", "Menghapus project yang sudah diarsipkan atau dibatalkan, beserta seluruh datanya."},

	{"finance.record", KindStaff, "Keuangan", "Catat pembayaran", "Mencatat dan mengubah pembayaran client, vendor, dan venue."},
	{"finance.margin.view", KindStaff, "Keuangan", "Lihat margin", "Menampilkan ringkasan margin (nilai kontrak dikurangi biaya vendor dan venue) di halaman project."},
	{"finance.payment.delete", KindStaff, "Keuangan", "Hapus pembayaran", "Menghapus catatan pembayaran yang salah."},

	{"client.manage", KindStaff, "Client", "Kelola client", "Menambah client, mengirim undangan dan link atur ulang sandi, serta mengganti perwakilan keluarga."},

	{"vendor.view", KindStaff, "Direktori", "Lihat vendor & venue", "Membuka menu Vendor dan Venue: daftar lengkap, ekspor, dan riwayat project. Tanpa izin ini, orang yang mengerjakan project tetap bisa memilih vendor dan venue untuk project-nya."},
	{"vendor.manage", KindStaff, "Direktori", "Kelola vendor & venue", "Menambah, mengubah, mengimpor, dan mengekspor vendor dan venue."},
	{"vendor.delete", KindStaff, "Direktori", "Hapus vendor & venue", "Menghapus vendor dan venue secara permanen."},
	{"vendor.category.manage", KindStaff, "Direktori", "Kelola kategori vendor", "Menambah dan mengubah kategori vendor."},

	{"template.manage", KindStaff, "Pengaturan", "Kelola Timeline Default", "Mengubah tahapan bawaan yang dipakai setiap project baru."},
	{"team.manage", KindStaff, "Pengaturan", "Kelola tim", "Mengundang, mengubah, dan menonaktifkan staf."},
	{"role.manage", KindStaff, "Pengaturan", "Kelola peran & akses", "Membuat dan mengubah peran serta izinnya."},
	{"subscription.manage", KindStaff, "Pengaturan", "Kelola langganan", "Melihat tagihan dan membayar langganan ElProof."},
	{"business_profile.manage", KindStaff, "Pengaturan", "Kelola profil usaha", "Mengubah nama WO, WhatsApp, kota, dan logo."},

	{"portal.finance.view", KindClient, "Portal client", "Lihat keuangan", "Melihat nilai kontrak, tagihan, dan riwayat pembayaran."},
	{"portal.vendor.view", KindClient, "Portal client", "Lihat vendor & venue", "Melihat vendor dan venue beserta progres persiapannya."},
	{"portal.timeline.view", KindClient, "Portal client", "Lihat timeline", "Melihat tahapan persiapan pernikahan."},
	{"portal.documents.view", KindClient, "Portal client", "Lihat dokumen", "Melihat dokumen yang dibagikan WO ke client."},
	{"portal.issues.view", KindClient, "Portal client", "Lihat kendala", "Melihat kendala dan penanganannya."},
}

var catalogByKey = func() map[string]Permission {
	m := make(map[string]Permission, len(Catalog))
	for _, p := range Catalog {
		m[p.Key] = p
	}
	return m
}()

// Lookup returns a catalog entry.
func Lookup(key string) (Permission, bool) {
	p, ok := catalogByKey[key]
	return p, ok
}

// AllKeys lists every key of one kind (the Owner role holds all staff keys).
func AllKeys(kind Kind) []string {
	var keys []string
	for _, p := range Catalog {
		if p.Kind == kind {
			keys = append(keys, p.Key)
		}
	}
	return keys
}

// Template is a built-in role every workspace starts with; WOs may edit
// every template except Owner, duplicate them, or create their own roles.
type Template struct {
	Key         string
	Kind        Kind
	Name        string
	Description string
	System      bool
	Scope       Scope
	Permissions []string
}

const (
	TemplateOwner          = "owner"
	TemplateAdmin          = "admin"
	TemplateWeddingPlanner = "wedding_planner"
	TemplateSales          = "sales"
	TemplateFinance        = "finance"
	TemplateBride          = "bride"
	TemplateFamily         = "family"
)

// Templates mirror migration 000058 (which seeded existing workspaces).
var Templates = []Template{
	{TemplateOwner, KindStaff, "Owner", "Pemilik workspace. Semua akses; tidak bisa diubah atau dihapus.", true, ScopeAll, AllKeys(KindStaff)},
	{TemplateAdmin, KindStaff, "Admin", "Mengelola semua project, client, vendor, dan keuangan. Tanpa pengaturan workspace.", false, ScopeAll,
		[]string{"dashboard.view", "dashboard.revenue.view", "project.create", "project.edit", "project.assign_pic", "project.duplicate", "project.archive",
			"finance.record", "finance.margin.view", "finance.payment.delete", "client.manage", "vendor.view", "vendor.manage"}},
	{TemplateWeddingPlanner, KindStaff, "Wedding Planner", "Menjalankan project yang ditugaskan kepadanya.", false, ScopeAssigned,
		[]string{"project.edit", "finance.record"}},
	{TemplateSales, KindStaff, "Sales", "Membuat project baru dan mengelola client dari project yang ia pegang.", false, ScopeAssigned,
		[]string{"project.create", "project.edit", "project.assign_pic", "project.archive", "finance.record", "finance.margin.view", "client.manage"}},
	{TemplateFinance, KindStaff, "Keuangan", "Mencatat dan memantau pembayaran di semua project.", false, ScopeAll,
		[]string{"dashboard.view", "dashboard.revenue.view", "finance.record", "finance.margin.view", "finance.payment.delete", "vendor.view"}},
	{TemplateBride, KindClient, "Pengantin", "Melihat seluruh portal, termasuk tagihan dan riwayat pembayaran.", false, ScopeAll,
		[]string{"portal.finance.view", "portal.vendor.view", "portal.timeline.view", "portal.documents.view", "portal.issues.view"}},
	{TemplateFamily, KindClient, "Keluarga", "Melihat portal tanpa informasi keuangan.", false, ScopeAll,
		[]string{"portal.vendor.view", "portal.timeline.view", "portal.documents.view", "portal.issues.view"}},
}
