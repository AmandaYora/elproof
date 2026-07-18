const FAQS = [
  {
    question: "Apa itu ElProof?",
    answer: `ElProof adalah platform berbasis langganan (Software as a Service) yang membantu wedding
    organizer (WO) mendokumentasikan persiapan project pernikahan — vendor, milestone, pembayaran, dan
    kendala — serta membagikan sebagian informasi tersebut kepada pasangan client terkait lewat Client
    Portal.`,
  },
  {
    question: "Siapa yang bisa menggunakan ElProof?",
    answer: `ElProof ditujukan untuk bisnis wedding organizer. Setiap bisnis WO mendaftar sebagai satu
    Tenant, dengan akun Owner yang dapat menambahkan Staff lain sesuai kebutuhan. Pasangan client
    mendapat akses baca terbatas melalui Client Portal, bukan akun berlangganan tersendiri.`,
  },
  {
    question: "Bagaimana cara mulai berlangganan?",
    answer: `Aktivasi akun Tenant dibantu langsung oleh tim kami. Hubungi kami melalui halaman Kontak untuk
    menceritakan kebutuhan bisnis Anda, dan tim kami akan membuatkan akun Owner setelah proses aktivasi
    selesai.`,
  },
  {
    question: "Metode pembayaran apa saja yang didukung?",
    answer: `Pembayaran langganan diproses melalui mitra payment gateway pihak ketiga. ElProof tidak
    menyimpan data kartu atau rekening pembayaran Anda — seluruh proses otorisasi pembayaran ditangani
    langsung oleh mitra tersebut.`,
  },
  {
    question: "Apa yang terjadi jika langganan saya berakhir?",
    answer: `Apabila langganan berakhir tanpa perpanjangan, akses Tenant terhadap ElProof dapat
    dinonaktifkan sementara hingga langganan diperpanjang kembali. Data yang sudah tercatat tidak dihapus
    akibat keterlambatan perpanjangan.`,
  },
  {
    question: "Apakah saya bisa mengajukan refund?",
    answer: `Ya, dalam kondisi tertentu — misalnya kesalahan penagihan sistem. Lihat halaman Kebijakan
    Refund untuk syarat, kondisi yang tidak dapat direfund, dan proses pengajuannya.`,
  },
  {
    question: "Apakah data project saya aman?",
    answer: `Data antar-Tenant terpisah secara penuh dan tidak dapat diakses lintas akun bisnis. ElProof
    menerapkan pemisahan akses berbasis peran (Staff, Client, admin platform) sehingga setiap pihak hanya
    melihat data yang menjadi haknya. Lihat Kebijakan Privasi untuk detail lengkap.`,
  },
  {
    question: "Bagaimana cara menghubungi dukungan pelanggan?",
    answer: `Tim kami dapat dihubungi melalui telepon, WhatsApp, atau email pada jam layanan yang tertera
    di halaman Kontak. Di luar jam tersebut, pesan Anda tetap akan kami balas sesegera mungkin.`,
  },
];

export default function FaqPage() {
  return (
    <div>
      <div className="border-b border-border bg-surface px-6 py-14">
        <div className="mx-auto max-w-3xl">
          <p className="text-[12.5px] font-semibold uppercase tracking-wide text-navy-900/70">Bantuan</p>
          <h1 className="mt-3 font-display text-[32px] font-semibold text-text-primary md:text-[38px]">
            Pertanyaan yang Sering Diajukan
          </h1>
          <p className="mt-3 max-w-xl text-[14.5px] leading-relaxed text-text-secondary">
            Belum menemukan jawaban yang Anda cari? Hubungi kami langsung melalui halaman Kontak.
          </p>
        </div>
      </div>

      <div className="px-6 py-14">
        <div className="mx-auto flex max-w-3xl flex-col gap-8">
          {FAQS.map((item) => (
            <div key={item.question}>
              <h2 className="text-[16px] font-semibold text-text-primary">{item.question}</h2>
              <p className="mt-2 text-[14.5px] leading-relaxed text-text-secondary">{item.answer}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
