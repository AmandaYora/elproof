# ADR-0016: Venue extraction — its own entity, not a vendor category

## Status
Accepted — implemented, per the "Revisi" section below (not the original pre-revision plan;
`venue_photos` never shipped, see "What actually shipped" at the end of this document).

## Context

"Venue" has never been a real concept in this system — it is one of 8 hardcoded strings
(`vendor_category_service.go`'s `defaultCategoryTemplate`) seeded into every new tenant's
`vendor_categories` table, indistinguishable from "Katering," "Dekorasi," etc. Every vendor
(including venues) is backed by the exact same generic `vendors` row: name, PIC, phone, email,
address, notes — nothing category-specific.

The user wants a venue-booking workflow a generic vendor directory can't support: a mandatory
rental price, a separate fixed charge, capacity, a facilities list, social media links, a photo
gallery, and a contract document — none of which make sense on a caterer or MUA's record. This
can't be bolted onto `vendors` as nullable columns without the schema smell of columns meaningful
to one category out of eight; it needs its own table.

Separately (and unrelated in the code, despite the name collision), `projects.venue` is a free-text
column — the wedding's location, typed once on the project form, never linked to any vendor record.
This ADR does not touch that column's existing behavior; see "What this does not change" below.

## Decision

### Venue is a new table, still inside the `vendors` module
`venues` + `venue_photos` (one-to-many gallery) are new tables owned by the existing `vendors`
module — not a new top-level Go module. Venue and Vendor are conceptually siblings (both are
"external parties/places a WO works with"); splitting into a wholly separate module would add
constructor/contract wiring overhead with no isolation benefit, since nothing about Venue needs a
cross-module boundary Vendor doesn't already have. "Venue" is removed from `vendor_categories`'
seed list and from any tenant that already has it (deactivated, not deleted — see "Migration"
below): it is no longer a vendor category once this ships.

### Cardinality: exactly one venue per project, via a primitive cross-module reference
Unlike vendor engagements (many per project, tracked in `project_vendors` with its own
contract-value/DP/paid-amount fields), a project has at most one venue. `projects` gains a
nullable `venue_id` column — a primitive reference to `venues.id`, resolved through a module
contract (never a SQL foreign key, per the modular-monolith rule) — instead of a
`project_venues` engagement table. This ADR deliberately does **not** add per-project
negotiated-price/DP/paid-amount tracking for venues the way `project_vendors` has for other
vendors: the user's request didn't ask for it, and building it speculatively would be scope the
user didn't request. If "how much have we paid this venue for this wedding" tracking is wanted
later, it's a natural, isolated extension (a `project_venue` table mirroring `project_vendors`,
constrained to one row per project) — not assumed here.

### Facilities and social media: free-form lists the WO manages per venue, not a fixed enum
Both are stored as JSON columns (`facilities`, `social_media`) — arrays the WO adds to/removes
from directly on the venue's own form. No master "facility types" table, no fixed platform list:
this was a deliberate simplification after considering (and rejecting) a fixed checklist / fixed
per-platform-column design, since neither concept needs tenant-wide governance or is ever filtered/
reported on elsewhere — a JSON array is sufficient and avoids a master-data CRUD screen nobody
asked for.

### Rental price: mandatory on creation, nullable at the database
`rental_price` is `NULL`-able in the schema (required for the data migration below to succeed —
migrated venues have no historical rental price to carry over) but enforced as required in the
**create** form only. A venue with no rental price yet is not blocked from being attached to a
project — the UI flags it ("harga sewa belum diisi") without blocking the WO's work in progress.

### Two attachment kinds, two existing storage patterns — no new pattern needed
- **Photo gallery** (many): `venue_photos` side table, one row per file — the same shape as
  project evidence (ADR-0006/0010).
- **Contract document** (one): a single nullable `contract_document_path` column on `venues`
  itself — the same shape as the tenant logo (ADR-0012).

Both reuse the existing base64-JSON-upload / byte-proxy-download convention (`internal/shared/
storage`) — no new upload mechanism.

### Client Portal gets a Venue tab; only the public-safe fields
Commercial terms (`rental_price`, `charge`, `contract_document_path`, PIC contact) are a WO↔venue
negotiation, not the client couple's business — mirroring how vendor engagement's contract
value/DP/paid amounts are staff-only today. The Client Portal's new "Venue" tab, and the
cross-module contract method backing it, expose only name/address/capacity/facilities/social
media/photos. WO Console's own Project Detail gets a symmetric "Venue" tab with the full record
(commercial fields included) plus a way to attach/change the project's venue.

### A new, reciprocal module dependency: `projects` now reads from `vendors`
Today only `vendors → projects` exists (vendor engagement history). Showing venue info on a
project (both tabs above) means `projects` needs to resolve `venue_id` into venue details, i.e.
`projects → vendors`. This is not a cycle at the Go package level: `projects/application` defines
its own narrow consumer interface (`VenueResolver`, mirroring `ClientCleaner`'s existing shape)
returning a type from `vendors/contracts` (a leaf package with no dependency back on `projects`),
and main.go wires the concrete `vendors.Contracts()` value into `projects` via the same two-phase
setter pattern already used for `SetClientAccessResolver`/`SetClientCleaner`. No new architectural
mechanism — just one more application of the existing one.

### Migration of existing data
Tenants that already registered have real venue data sitting in `vendors` rows under the "Venue"
category (per the user's own screenshot). A one-time data migration copies those rows
(name/PIC/phone/email/address/notes) into `venues` — the new fields land empty, for the WO to fill
in — then deactivates (never deletes) the source `vendors` rows and the "Venue" `vendor_categories`
row itself, consistent with this module's existing no-hard-delete convention (`SetActive`, same as
project milestones). Existing `project_vendors` history rows already engaging a "Venue"-category
vendor are left untouched — not auto-linked to the new `venue_id`, since fuzzy-matching old
engagements to new venue rows risks linking the wrong one. Only newly attached venues (post-launch)
use the new mechanism; old projects keep showing their existing vendor-engagement history exactly
as today.

## What this does not change
- `projects.venue` (the free-text wedding-location string) is untouched — kept as a fallback for
  projects that never get a structured `venue_id`, per the migration-safety reasoning above.
- Login remains resolved by username/email; venue data has no bearing on ADR-0015's custom-domain
  branding.
- No CORS/deployment implications — this is entirely application-layer + one migration.

## Consequences
- New migrations `000020_create_venue_tables`, `000021_add_venue_id_to_projects`,
  `000022_migrate_venue_vendor_data` (see PLAN.md for exact SQL).
- New backend files inside `vendors` (domain/application/infrastructure/presentation for Venue +
  VenuePhoto), an extended `vendors/contracts` interface, and a new `projects/application`
  consumer interface + two-phase wiring call in `main.go`.
- New frontend module `apps/web/src/modules/venues/` (mirroring `modules/vendors/`), a new sidebar
  item, a new Project Detail tab (WO Console), and a new tab in Client Portal.
- `vendor_category_service.go`'s seed template drops the "Venue" entry for all future tenant
  registrations.

## Revisi — field-level, sebelum sempat diimplementasikan ke database nyata

Keputusan arsitektur di atas (Venue = tabel baru tapi tetap modul `vendors`, kardinalitas 1:1 ke
project via `venue_id`, dua-arah dependency `projects ↔ vendors` lewat two-phase wiring) **tidak
berubah**. Yang direvisi murni di level field, berdasarkan requirement lanjutan dari user sebelum
kode ini sempat di-commit/migrate ke database manapun (jadi ini revisi desain, bukan migrasi
susulan di atas data produksi):

- **Telepon dipecah berdasarkan kepemilikan, bukan prioritas**: `phone`/`phone_secondary` (utama
  vs tambahan, tanpa makna jelas) diganti `phone_pic` (wajib — nomor personal PIC yang bisa
  dihubungi langsung) dan `phone_venue` (opsional — nomor official/kantor venue).
- **Alamat jadi opsional** (sebelumnya wajib).
- **Fasilitas dan Sosial Media disederhanakan jadi teks bebas (textarea)**, bukan array/struktur
  bersarang — alasannya dua: (1) konsistensi UI (semua field venue jadi tipe sederhana), dan (2)
  memudahkan fitur bulk-import di bawah (kolom Excel tidak perlu dipecah untuk data terstruktur).
- **Kota** — field wajib baru, search-select, tervalidasi terhadap daftar tetap 128 kota+kabupaten
  resmi (Jawa: DKI Jakarta, Banten, Jawa Barat, Jawa Tengah, DI Yogyakarta, Jawa Timur — lengkap
  kota dan kabupaten, bukan cuma kota, supaya venue di area seperti Ubud/Kuta/Nusa Dua — yang
  semuanya kabupaten, bukan Kota Denpasar — punya lokasi yang akurat). Sama seperti
  `rental_price`, kolom ini **nullable di database** (supaya data migrasi lama yang tidak punya
  info kota tidak gagal insert) tapi wajib diisi di form pembuatan venue baru.
- **Lampiran disatukan jadi 1 slot file** (dokumen ATAU foto), menggantikan desain lama yang
  memisah galeri foto (tabel `venue_photos`, banyak file) dari dokumen kontrak (1 kolom). Tabel
  `venue_photos` dihapus dari rencana sepenuhnya.
- **Privasi lampiran di Client Portal ditentukan otomatis dari MIME type**, bukan dari mekanisme
  terpisah seperti sebelumnya: kalau lampirannya gambar, klien boleh lihat (dianggap foto venue);
  kalau PDF/dokumen, disembunyikan dari klien (dianggap dokumen internal). Ini butuh kolom baru
  `attachment_mime_type` disimpan saat upload, dan pengecekan tipe principal (`staff` vs `client`)
  di endpoint stream file itu sendiri — bukan cuma di frontend — supaya klien tidak bisa mengakali
  dengan memanggil endpoint langsung.
- **Fitur baru: bulk upload/import** — download template Excel kosong, upload Excel untuk insert
  massal, dengan partial success (baris valid tetap masuk, baris error dilaporkan balik nomor+
  alasan agar bisa diperbaiki dan diupload ulang khusus baris itu tanpa mengulang semuanya).

Rencana implementasi konkret hasil revisi ini ada di `PLAN.md`'s "PLAN — Venue Extraction" section
(ditulis ulang, bukan ditambah di atas rencana lama, supaya tidak ada dua versi yang membingungkan).

## What actually shipped

Matches the "Revisi" section above, not the original "Decision"/"Consequences" text further up
(kept as-is for historical context, not corrected retroactively) — most notably, **no
`venue_photos` table was ever created**; a venue has exactly one attachment slot
(`attachment_path`/`attachment_mime_type`), same shape as `vendors`' own single-slot attachment.
Migrations: `000020_create_venue_tables`, `000021_add_venue_id_to_projects`,
`000022_migrate_venue_vendor_data`. See `DB_SCHEMA.md`'s `venues` table and `API_CONTRACT.md`'s
`venues` module section for the exact final column/endpoint list.
