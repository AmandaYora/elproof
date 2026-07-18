import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

const SECTIONS = [
  {
    title: "1. Data yang Kami Kumpulkan",
    body: [
      `Data akun: nama, email, dan nomor telepon Staff (Owner, Admin, Staff) serta kredensial akses Client
      yang dibuatkan oleh Tenant terkait.`,
      `Data operasional yang diinput Tenant: project pernikahan, vendor, milestone, status pembayaran, kendala,
      serta dokumen atau foto bukti kerja yang diunggah sebagai lampiran project.`,
      `Data langganan: riwayat status dan periode aktif paket langganan Tenant. ElProof tidak menyimpan data
      kartu atau rekening pembayaran — proses tersebut sepenuhnya ditangani oleh mitra payment gateway.`,
    ],
  },
  {
    title: "2. Bagaimana Kami Menggunakan Data",
    body: [
      `Untuk menjalankan layanan inti: mencatat data project dan menampilkan progress yang relevan kepada
      Staff dan Client sesuai peran masing-masing.`,
      `Untuk komunikasi terkait akun, aktivasi, dan status langganan Tenant.`,
      `Untuk pemeliharaan dan peningkatan keandalan layanan, termasuk penelusuran gangguan teknis.`,
    ],
  },
  {
    title: "3. Pembagian Data dengan Pihak Ketiga",
    body: [
      `ElProof tidak menjual atau membagikan data Tenant kepada pihak ketiga untuk kepentingan pemasaran.`,
      `Data pembayaran langganan diproses oleh mitra payment gateway pihak ketiga sebatas yang diperlukan
      untuk otorisasi transaksi. ElProof tidak menerima maupun menyimpan detail kartu atau rekening dari
      proses tersebut.`,
      `Data dapat diungkapkan apabila diwajibkan oleh ketentuan hukum yang berlaku.`,
    ],
  },
  {
    title: "4. Penyimpanan dan Retensi Data",
    body: [
      `Data project, vendor, dan dokumen bukti disimpan selama langganan Tenant aktif, untuk keperluan
      operasional layanan.`,
      `Apabila langganan berakhir tanpa perpanjangan, akses terhadap ElProof dapat dinonaktifkan sementara,
      namun data yang sudah tercatat tidak langsung dihapus akibat keterlambatan perpanjangan.`,
    ],
  },
  {
    title: "5. Pemisahan dan Keamanan Akses",
    body: [
      `Setiap Tenant hanya dapat mengakses data project miliknya sendiri. Data antar-Tenant terpisah secara
      penuh dan tidak dapat diakses lintas akun bisnis.`,
      `Client hanya dapat melihat data project yang menjadi tanggung jawab Tenant yang mengundangnya, terbatas
      pada status milestone, ringkasan pembayaran, dan kendala yang relevan — tidak mencakup data internal
      Tenant lainnya.`,
      `ElProof menerapkan pemisahan akses berbasis peran (Staff, Client, admin platform) sehingga setiap pihak
      hanya melihat data yang menjadi haknya.`,
    ],
  },
  {
    title: "6. Hak Anda atas Data",
    body: [
      `Staff dapat meminta akses, koreksi, atau penghapusan data akun melalui Owner Tenant-nya, atau langsung
      menghubungi tim ElProof lewat halaman Kontak.`,
      `Client dapat meminta koreksi data yang ditampilkan di Client Portal melalui Tenant yang mengelola
      project-nya, karena data tersebut diinput dan dimiliki oleh Tenant terkait.`,
    ],
  },
  {
    title: "7. Cookie dan Sesi Login",
    body: [
      `ElProof hanya menggunakan cookie atau penyimpanan sesi yang esensial untuk menjaga status login pada
      WO Console, Client Portal, dan Platform Console. Kami tidak menggunakan cookie iklan atau pelacakan
      pihak ketiga untuk kepentingan pemasaran.`,
    ],
  },
  {
    title: "8. Perubahan Kebijakan",
    body: [
      `Kebijakan ini dapat diperbarui sewaktu-waktu mengikuti perkembangan layanan. Perubahan berlaku sejak
      dipublikasikan pada halaman ini.`,
    ],
  },
];

export default function PrivacyPage() {
  return (
    <div>
      <div className="border-b border-border bg-surface px-6 py-14">
        <div className="mx-auto max-w-3xl">
          <p className="text-[12.5px] font-semibold uppercase tracking-wide text-navy-900/70">Dokumen Layanan</p>
          <h1 className="mt-3 font-display text-[32px] font-semibold text-text-primary md:text-[38px]">
            Kebijakan Privasi
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

          <div className="rounded-xl border border-border bg-surface-muted/60 p-6">
            <h3 className="text-[14.5px] font-semibold text-text-primary">Pertanyaan seputar privasi data?</h3>
            <p className="mt-1.5 text-[13.5px] leading-relaxed text-text-secondary">
              Hubungi tim kami melalui halaman{" "}
              <Link to={ROUTE_PATHS.homepageContact} className="font-medium text-navy-900 underline underline-offset-2">
                Kontak
              </Link>{" "}
              untuk pertanyaan atau permintaan terkait data Anda.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
