const SECTIONS = [
  {
    title: "1. Ketentuan Umum",
    body: [
      `ElProof adalah layanan berbasis langganan (Software as a Service), bukan barang fisik. Kebijakan
      refund ini mengatur kondisi di mana Tenant berhak mengajukan pengembalian dana atas pembayaran
      langganan yang telah diproses melalui mitra payment gateway.`,
    ],
  },
  {
    title: "2. Kondisi yang Berhak Direfund",
    body: [
      `Kesalahan penagihan sistem — misalnya Tenant tertagih dua kali (duplikat) untuk paket langganan
      yang sama pada periode yang sama.`,
      `Layanan tidak dapat diakses sama sekali akibat gangguan di pihak ElProof selama lebih dari 3 (tiga)
      hari kerja berturut-turut sejak pembayaran diterima, dan belum sempat digunakan sama sekali oleh
      Tenant.`,
      `Pembayaran berhasil diproses namun aktivasi akun Tenant tidak dilakukan dalam waktu wajar akibat
      kesalahan di pihak ElProof.`,
    ],
  },
  {
    title: "3. Kondisi yang Tidak Dapat Direfund",
    body: [
      `Langganan yang sudah digunakan sebagian atau seluruhnya untuk mengelola project, sejak Owner atau
      Staff pertama kali login ke akun tersebut.`,
      `Perubahan pikiran (change of mind) setelah pembayaran berhasil, tanpa adanya kegagalan layanan di
      pihak ElProof.`,
      `Akses yang dinonaktifkan sementara karena keterlambatan perpanjangan langganan — ini bukan
      pemutusan layanan, dan dapat diaktifkan kembali kapan saja setelah perpanjangan dilakukan.`,
      `Kesalahan input data, kesalahan pemilihan paket, atau kendala penggunaan yang berasal dari pihak
      Tenant sendiri.`,
    ],
  },
  {
    title: "4. Cara Mengajukan Refund",
    body: [
      `Ajukan permintaan refund melalui salah satu kanal pada halaman Kontak (telepon, WhatsApp, atau
      email), selambat-lambatnya 7 (tujuh) hari kalender sejak tanggal pembayaran, dengan menyertakan
      bukti pembayaran dan alasan pengajuan.`,
      `Tim kami akan memverifikasi kelayakan pengajuan berdasarkan kondisi pada Bagian 2 dan 3 di atas,
      dan memberikan keputusan selambat-lambatnya 5 (lima) hari kerja setelah pengajuan diterima lengkap.`,
    ],
  },
  {
    title: "5. Proses dan Waktu Pengembalian Dana",
    body: [
      `Refund yang disetujui diproses melalui mitra payment gateway yang sama dengan metode pembayaran
      awal, dan diteruskan ke rekening atau instrumen pembayaran Tenant selambat-lambatnya 14 (empat
      belas) hari kerja sejak persetujuan — mengikuti waktu proses mitra pembayaran terkait.`,
      `ElProof tidak memungut biaya tambahan atas proses refund yang disetujui; biaya administrasi yang
      mungkin dikenakan oleh mitra payment gateway di luar kendali ElProof.`,
    ],
  },
  {
    title: "6. Perubahan Kebijakan",
    body: [
      `Kebijakan ini dapat diperbarui sewaktu-waktu mengikuti perkembangan layanan. Perubahan berlaku
      sejak dipublikasikan pada halaman ini dan tidak berlaku surut terhadap pengajuan refund yang sudah
      diputuskan sebelumnya.`,
    ],
  },
];

export default function RefundPolicyPage() {
  return (
    <div>
      <div className="border-b border-border bg-surface px-6 py-14">
        <div className="mx-auto max-w-3xl">
          <p className="text-[12.5px] font-semibold uppercase tracking-wide text-navy-900/70">Dokumen Layanan</p>
          <h1 className="mt-3 font-display text-[32px] font-semibold text-text-primary md:text-[38px]">
            Kebijakan Refund
          </h1>
          <p className="mt-3 text-[13.5px] text-text-secondary">Terakhir diperbarui: 18 Juli 2026</p>
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
