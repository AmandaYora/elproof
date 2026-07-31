package domain

// AllowedCities is the fixed set of 128 official kota/kabupaten across Java's six provinces
// (DKI Jakarta, Banten, Jawa Barat, Jawa Tengah, DI Yogyakarta, Jawa Timur) plus Bali — the only
// two islands this feature covers (ADR-0016). Deliberately kota AND kabupaten, not kota alone:
// Bali has exactly one official kota (Denpasar), so a kota-only list would leave every venue in
// Ubud/Kuta/Nusa Dua/Seminyak (all kabupaten Gianyar/Badung) with no accurate city to pick.
// Originally added for Venue only (hence the file's former name, venue_city.go); Vendor's own
// "Kota" field (per the "DATA VENDOR" slide) reuses this exact same list rather than duplicating
// it, since both entities live in this same module.
var AllowedCities = []string{
	// DKI Jakarta (5 kota administrasi + 1 kabupaten)
	"Jakarta Pusat", "Jakarta Utara", "Jakarta Barat", "Jakarta Selatan", "Jakarta Timur",
	"Kepulauan Seribu",
	// Banten (4 kota + 4 kabupaten)
	"Kota Tangerang", "Kota Tangerang Selatan", "Kota Serang", "Kota Cilegon",
	"Kabupaten Tangerang", "Kabupaten Serang", "Kabupaten Pandeglang", "Kabupaten Lebak",
	// Jawa Barat (9 kota + 18 kabupaten)
	"Kota Bandung", "Kota Bekasi", "Kota Bogor", "Kota Depok", "Kota Cimahi", "Kota Cirebon",
	"Kota Sukabumi", "Kota Tasikmalaya", "Kota Banjar",
	"Kabupaten Bandung", "Kabupaten Bandung Barat", "Kabupaten Bekasi", "Kabupaten Bogor",
	"Kabupaten Ciamis", "Kabupaten Cianjur", "Kabupaten Cirebon", "Kabupaten Garut",
	"Kabupaten Indramayu", "Kabupaten Karawang", "Kabupaten Kuningan", "Kabupaten Majalengka",
	"Kabupaten Pangandaran", "Kabupaten Purwakarta", "Kabupaten Subang", "Kabupaten Sukabumi",
	"Kabupaten Sumedang", "Kabupaten Tasikmalaya",
	// Jawa Tengah (6 kota + 29 kabupaten)
	"Kota Semarang", "Kota Surakarta", "Kota Salatiga", "Kota Magelang", "Kota Pekalongan",
	"Kota Tegal",
	"Kabupaten Banjarnegara", "Kabupaten Banyumas", "Kabupaten Batang", "Kabupaten Blora",
	"Kabupaten Boyolali", "Kabupaten Brebes", "Kabupaten Cilacap", "Kabupaten Demak",
	"Kabupaten Grobogan", "Kabupaten Jepara", "Kabupaten Karanganyar", "Kabupaten Kebumen",
	"Kabupaten Kendal", "Kabupaten Klaten", "Kabupaten Kudus", "Kabupaten Magelang",
	"Kabupaten Pati", "Kabupaten Pekalongan", "Kabupaten Pemalang", "Kabupaten Purbalingga",
	"Kabupaten Purworejo", "Kabupaten Rembang", "Kabupaten Semarang", "Kabupaten Sragen",
	"Kabupaten Sukoharjo", "Kabupaten Tegal", "Kabupaten Temanggung", "Kabupaten Wonogiri",
	"Kabupaten Wonosobo",
	// DI Yogyakarta (1 kota + 4 kabupaten)
	"Kota Yogyakarta", "Kabupaten Sleman", "Kabupaten Bantul", "Kabupaten Kulon Progo",
	"Kabupaten Gunung Kidul",
	// Jawa Timur (9 kota + 29 kabupaten)
	"Kota Surabaya", "Kota Malang", "Kota Batu", "Kota Kediri", "Kota Blitar", "Kota Madiun",
	"Kota Mojokerto", "Kota Pasuruan", "Kota Probolinggo",
	"Kabupaten Bangkalan", "Kabupaten Banyuwangi", "Kabupaten Blitar", "Kabupaten Bojonegoro",
	"Kabupaten Bondowoso", "Kabupaten Gresik", "Kabupaten Jember", "Kabupaten Jombang",
	"Kabupaten Kediri", "Kabupaten Lamongan", "Kabupaten Lumajang", "Kabupaten Madiun",
	"Kabupaten Magetan", "Kabupaten Malang", "Kabupaten Mojokerto", "Kabupaten Nganjuk",
	"Kabupaten Ngawi", "Kabupaten Pacitan", "Kabupaten Pamekasan", "Kabupaten Pasuruan",
	"Kabupaten Ponorogo", "Kabupaten Probolinggo", "Kabupaten Sampang", "Kabupaten Sidoarjo",
	"Kabupaten Situbondo", "Kabupaten Sumenep", "Kabupaten Trenggalek", "Kabupaten Tuban",
	"Kabupaten Tulungagung",
	// Bali (1 kota + 8 kabupaten)
	"Kota Denpasar", "Kabupaten Badung", "Kabupaten Bangli", "Kabupaten Buleleng",
	"Kabupaten Gianyar", "Kabupaten Jembrana", "Kabupaten Karangasem", "Kabupaten Klungkung",
	"Kabupaten Tabanan",
}

// allowedCitySet mirrors AllowedCities as a set, built exactly once at package load (not per
// call) so IsValidCity is an O(1) map lookup instead of an O(128) linear scan. AllowedCities
// itself stays a plain []string (needed in-order for the Kota dropdown/Excel template — a map
// has no stable iteration order).
var allowedCitySet = buildCitySet()

func buildCitySet() map[string]struct{} {
	set := make(map[string]struct{}, len(AllowedCities))
	for _, c := range AllowedCities {
		set[c] = struct{}{}
	}
	return set
}

func IsValidCity(city string) bool {
	_, ok := allowedCitySet[city]
	return ok
}
