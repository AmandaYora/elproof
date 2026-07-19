import { useEffect, useState } from "react";
import { Scale } from "lucide-react";
import { Reveal } from "@/modules/homepage/components/Reveal";
import { cn } from "@/shared/lib/cn";

const SECTIONS = [
  {
    id: "definisi",
    title: "1. Definisi",
    body: [
      `"ElProof" adalah platform perangkat lunak berbasis langganan (Software as a Service) yang membantu wedding organizer ("WO") mendokumentasikan persiapan project pernikahan — termasuk vendor, milestone, pembayaran, dan kendala — serta membagikan sebagian informasi tersebut kepada pasangan client terkait.`,
      `"Tenant" adalah satu akun bisnis WO yang berlangganan ElProof. "Staff" adalah pengguna yang bekerja di bawah satu Tenant (Owner, Admin, atau Staff). "Client" adalah pasangan yang project pernikahannya dikelola oleh Tenant tersebut, dan diberi akses baca terbatas melalui Client Portal.`,
    ],
  },
  {
    id: "ruang-lingkup",
    title: "2. Ruang Lingkup Layanan",
    body: [
      `ElProof menyediakan pencatatan project, manajemen vendor dan kategori vendor, pelacakan milestone dan pembayaran, log aktivitas, serta portal khusus bagi Client untuk memantau progress project mereka sendiri secara read-only.`,
      `ElProof tidak menjadi pihak dalam perjanjian antara Tenant dan vendor atau Client-nya. Seluruh kesepakatan bisnis, kontrak kerja, dan transaksi antara Tenant dengan pihak ketiga sepenuhnya menjadi tanggung jawab Tenant.`,
    ],
  },
  {
    id: "akun-peran",
    title: "3. Akun dan Peran Pengguna",
    body: [
      `Akun Tenant beserta akun Staff pertamanya (Owner) dibuat oleh tim ElProof setelah proses aktivasi — lihat halaman Kontak. Owner selanjutnya dapat menambah Staff lain sesuai kebutuhan bisnisnya.`,
      `Setiap Tenant hanya dapat mengakses data project miliknya sendiri. Data antar-Tenant terpisah secara penuh dan tidak dapat diakses lintas akun bisnis.`,
      `Client menerima kredensial akses Client Portal dari Tenant yang menangani project pernikahannya, dan hanya dapat melihat data project tersebut — tidak project lain milik Tenant yang sama.`,
    ],
  },
  {
    id: "langganan-pembayaran",
    title: "4. Langganan dan Pembayaran",
    body: [
      `Akses ElProof diberikan per paket langganan dengan masa aktif tertentu. Perpanjangan langganan dilakukan oleh Owner Tenant melalui menu Langganan di WO Console.`,
      `Pembayaran diproses melalui mitra payment gateway pihak ketiga. ElProof tidak menyimpan data kartu atau rekening pembayaran Tenant — seluruh proses otorisasi pembayaran ditangani oleh mitra tersebut.`,
      `Apabila langganan berakhir tanpa perpanjangan, akses Tenant terhadap ElProof dapat dinonaktifkan sementara hingga langganan diperpanjang kembali. Data yang sudah tercatat tidak dihapus akibat keterlambatan perpanjangan.`,
    ],
  },
  {
    id: "data-kerahasiaan",
    title: "5. Data dan Kerahasiaan",
    body: [
      `Data project, vendor, dan dokumen bukti yang diunggah Tenant adalah milik Tenant tersebut. ElProof menyimpan data ini selama langganan aktif untuk keperluan operasional layanan.`,
      `Informasi yang ditampilkan kepada Client di Client Portal terbatas pada data project miliknya sendiri — status milestone, ringkasan pembayaran, dan kendala yang relevan — dan tidak mencakup data internal lain milik Tenant.`,
      `ElProof menerapkan pemisahan akses berbasis peran (Staff, Client, admin platform) untuk memastikan setiap pihak hanya melihat data yang menjadi haknya.`,
    ],
  },
  {
    id: "tanggung-jawab",
    title: "6. Tanggung Jawab Pengguna",
    body: [
      `Tenant bertanggung jawab atas keakuratan data yang dimasukkan ke dalam ElProof, termasuk status project, nilai kontrak, dan informasi vendor.`,
      `Setiap pengguna bertanggung jawab menjaga kerahasiaan kredensial akunnya sendiri dan segera melaporkan apabila terjadi dugaan akses tidak sah.`,
    ],
  },
  {
    id: "batasan-tanggung-jawab",
    title: "7. Batasan Tanggung Jawab",
    body: [
      `ElProof disediakan sebagai alat bantu dokumentasi dan transparansi, bukan sebagai penjamin terlaksananya perjanjian antara Tenant dengan vendor atau Client-nya. Perselisihan yang timbul dari hubungan bisnis tersebut diselesaikan langsung oleh pihak-pihak terkait.`,
    ],
  },
  {
    id: "perubahan-ketentuan",
    title: "8. Perubahan Ketentuan",
    body: [
      `Ketentuan ini dapat diperbarui sewaktu-waktu mengikuti perkembangan layanan. Perubahan berlaku sejak dipublikasikan pada halaman ini.`,
    ],
  },
];

export default function TermsPage() {
  const [activeId, setActiveId] = useState("");

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            setActiveId(entry.target.id);
          }
        });
      },
      { rootMargin: "-20% 0px -80% 0px" }
    );

    SECTIONS.forEach((s) => {
      const el = document.getElementById(s.id);
      if (el) observer.observe(el);
    });

    return () => observer.disconnect();
  }, []);

  return (
    <div className="bg-background min-h-screen">
      {/* HEADER */}
      <section className="relative overflow-hidden border-b border-border bg-white px-6 py-20 md:py-24">
        <div className="absolute inset-0 bg-[linear-gradient(to_right,#f1f5f9_1px,transparent_1px),linear-gradient(to_bottom,#f1f5f9_1px,transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)] opacity-60"></div>
        <div className="mx-auto max-w-4xl text-center relative z-10">
          <Reveal>
            <div className="inline-flex items-center gap-2 rounded-full bg-blue-50 px-3.5 py-1.5 text-[13px] font-bold uppercase tracking-widest text-blue-600">
              <Scale className="h-4 w-4" /> Dokumen Layanan
            </div>
            <h1 className="mt-6 font-display text-[36px] font-semibold tracking-tight text-navy-950 md:text-[48px]">
              Syarat &amp; Ketentuan
            </h1>
            <p className="mt-4 text-[15px] font-medium text-text-secondary">
              Terakhir diperbarui: 17 Juli 2026
            </p>
          </Reveal>
        </div>
      </section>

      {/* CONTENT */}
      <div className="mx-auto max-w-6xl px-6 py-16 md:py-24">
        <div className="flex flex-col md:flex-row gap-12 lg:gap-20">
          
          {/* SIDEBAR NAVIGATION */}
          <div className="hidden md:block w-64 shrink-0">
            <div className="sticky top-32 flex flex-col gap-2">
              <h3 className="text-[12px] font-bold uppercase tracking-wider text-text-secondary mb-4">Daftar Isi</h3>
              {SECTIONS.map((section) => (
                <a
                  key={section.id}
                  href={`#${section.id}`}
                  className={cn(
                    "text-[14px] font-medium transition-all duration-200 py-1.5 border-l-2 pl-4",
                    activeId === section.id
                      ? "border-navy-900 text-navy-950 bg-navy-50/50 rounded-r-md"
                      : "border-border text-text-secondary hover:border-navy-400 hover:text-navy-900"
                  )}
                >
                  {section.title}
                </a>
              ))}
            </div>
          </div>

          {/* DOCUMENT BODY */}
          <div className="flex-1 max-w-3xl rounded-3xl bg-white p-8 md:p-12 shadow-sm border border-border">
            <Reveal className="flex flex-col gap-12">
              {SECTIONS.map((section) => (
                <div key={section.id} id={section.id} className="scroll-mt-32">
                  <h2 className="text-[20px] font-bold text-navy-950 tracking-tight">{section.title}</h2>
                  <div className="mt-4 flex flex-col gap-4">
                    {section.body.map((paragraph, i) => (
                      <p key={i} className="text-[15.5px] leading-relaxed text-text-secondary">
                        {paragraph}
                      </p>
                    ))}
                  </div>
                </div>
              ))}
            </Reveal>
          </div>
        </div>
      </div>
    </div>
  );
}
