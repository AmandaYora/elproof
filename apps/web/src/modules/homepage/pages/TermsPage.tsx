const SECTIONS = [
  {
    title: "1. Definisi",
    body: [
      `"ElProof" adalah platform perangkat lunak berbasis langganan (Software as a Service) yang membantu
      wedding organizer ("WO") mendokumentasikan persiapan project pernikahan — termasuk vendor, milestone,
      pembayaran, dan kendala — serta membagikan sebagian informasi tersebut kepada pasangan client terkait.`,
      `"Tenant" adalah satu akun bisnis WO yang berlangganan ElProof. "Staff" adalah pengguna yang bekerja di
      bawah satu Tenant (Owner, Admin, atau Staff). "Client" adalah pasangan yang project pernikahannya
      dikelola oleh Tenant tersebut, dan diberi akses baca terbatas melalui Client Portal.`,
    ],
  },
  {
    title: "2. Ruang Lingkup Layanan",
    body: [
      `ElProof menyediakan pencatatan project, manajemen vendor dan kategori vendor, pelacakan milestone dan
      pembayaran, log aktivitas, serta portal khusus bagi Client untuk memantau progress project mereka
      sendiri secara read-only.`,
      `ElProof tidak menjadi pihak dalam perjanjian antara Tenant dan vendor atau Client-nya. Seluruh
      kesepakatan bisnis, kontrak kerja, dan transaksi antara Tenant dengan pihak ketiga sepenuhnya menjadi
      tanggung jawab Tenant.`,
    ],
  },
  {
    title: "3. Akun dan Peran Pengguna",
    body: [
      `Akun Tenant beserta akun Staff pertamanya (Owner) dibuat oleh tim ElProof setelah proses aktivasi —
      lihat halaman Kontak. Owner selanjutnya dapat menambah Staff lain sesuai kebutuhan bisnisnya.`,
      `Setiap Tenant hanya dapat mengakses data project miliknya sendiri. Data antar-Tenant terpisah secara
      penuh dan tidak dapat diakses lintas akun bisnis.`,
      `Client menerima kredensial akses Client Portal dari Tenant yang menangani project pernikahannya, dan
      hanya dapat melihat data project tersebut — tidak project lain milik Tenant yang sama.`,
    ],
  },
  {
    title: "4. Langganan dan Pembayaran",
    body: [
      `Akses ElProof diberikan per paket langganan dengan masa aktif tertentu. Perpanjangan langganan
      dilakukan oleh Owner Tenant melalui menu Langganan di WO Console.`,
      `Pembayaran diproses melalui mitra payment gateway pihak ketiga. ElProof tidak menyimpan data kartu
      atau rekening pembayaran Tenant — seluruh proses otorisasi pembayaran ditangani oleh mitra tersebut.`,
      `Apabila langganan berakhir tanpa perpanjangan, akses Tenant terhadap ElProof dapat dinonaktifkan
      sementara hingga langganan diperpanjang kembali. Data yang sudah tercatat tidak dihapus akibat
      keterlambatan perpanjangan.`,
    ],
  },
  {
    title: "5. Data dan Kerahasiaan",
    body: [
      `Data project, vendor, dan dokumen bukti yang diunggah Tenant adalah milik Tenant tersebut. ElProof
      menyimpan data ini selama langganan aktif untuk keperluan operasional layanan.`,
      `Informasi yang ditampilkan kepada Client di Client Portal terbatas pada data project miliknya sendiri
      — status milestone, ringkasan pembayaran, dan kendala yang relevan — dan tidak mencakup data internal
      lain milik Tenant.`,
      `ElProof menerapkan pemisahan akses berbasis peran (Staff, Client, admin platform) untuk memastikan
      setiap pihak hanya melihat data yang menjadi haknya.`,
    ],
  },
  {
    title: "6. Tanggung Jawab Pengguna",
    body: [
      `Tenant bertanggung jawab atas keakuratan data yang dimasukkan ke dalam ElProof, termasuk status
      project, nilai kontrak, dan informasi vendor.`,
      `Setiap pengguna bertanggung jawab menjaga kerahasiaan kredensial akunnya sendiri dan segera melaporkan
      apabila terjadi dugaan akses tidak sah.`,
    ],
  },
  {
    title: "7. Batasan Tanggung Jawab",
    body: [
      `ElProof disediakan sebagai alat bantu dokumentasi dan transparansi, bukan sebagai penjamin
      terlaksananya perjanjian antara Tenant dengan vendor atau Client-nya. Perselisihan yang timbul dari
      hubungan bisnis tersebut diselesaikan langsung oleh pihak-pihak terkait.`,
    ],
  },
  {
    title: "8. Perubahan Ketentuan",
    body: [
      `Ketentuan ini dapat diperbarui sewaktu-waktu mengikuti perkembangan layanan. Perubahan berlaku sejak
      dipublikasikan pada halaman ini.`,
    ],
  },
];

export default function TermsPage() {
  return (
    <div>
      <div className="border-b border-border bg-surface px-6 py-14">
        <div className="mx-auto max-w-3xl">
          <p className="text-[12.5px] font-semibold uppercase tracking-wide text-navy-900/70">Dokumen Layanan</p>
          <h1 className="mt-3 font-display text-[32px] font-semibold text-text-primary md:text-[38px]">
            Syarat &amp; Ketentuan
          </h1>
          <p className="mt-3 text-[13.5px] text-text-secondary">Terakhir diperbarui: 17 Juli 2026</p>
        </div>
      </div>

      <div className="px-6 py-14">
        <div className="mx-auto flex max-w-3xl flex-col gap-10">
          {SECTIONS.map((section) => (
            <div key={section.title}>
              <h2 className="text-[17px] font-semibold text-text-primary">{section.title}</h2>
              <div className="mt-3 flex flex-col gap-3">
                {section.body.map((paragraph, i) => (
                  <p key={i} className="text-[14.5px] leading-relaxed text-text-secondary">
                    {paragraph}
                  </p>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
