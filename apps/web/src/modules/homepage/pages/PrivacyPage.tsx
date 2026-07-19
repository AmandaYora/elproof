import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { ShieldAlert } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Reveal } from "@/modules/homepage/components/Reveal";
import { cn } from "@/shared/lib/cn";

const SECTIONS = [
  {
    id: "data-dikumpulkan",
    title: "1. Data yang Kami Kumpulkan",
    body: [
      `Data akun: nama, email, dan nomor telepon Staff (Owner, Admin, Staff) serta kredensial akses Client yang dibuatkan oleh Tenant terkait.`,
      `Data operasional yang diinput Tenant: project pernikahan, vendor, milestone, status pembayaran, kendala, serta dokumen atau foto bukti kerja yang diunggah sebagai lampiran project.`,
      `Data langganan: riwayat status dan periode aktif paket langganan Tenant. ElProof tidak menyimpan data kartu atau rekening pembayaran — proses tersebut sepenuhnya ditangani oleh mitra payment gateway.`,
    ],
  },
  {
    id: "penggunaan-data",
    title: "2. Bagaimana Kami Menggunakan Data",
    body: [
      `Untuk menjalankan layanan inti: mencatat data project dan menampilkan progress yang relevan kepada Staff dan Client sesuai peran masing-masing.`,
      `Untuk komunikasi terkait akun, aktivasi, dan status langganan Tenant.`,
      `Untuk pemeliharaan dan peningkatan keandalan layanan, termasuk penelusuran gangguan teknis.`,
    ],
  },
  {
    id: "pembagian-data",
    title: "3. Pembagian Data dengan Pihak Ketiga",
    body: [
      `ElProof tidak menjual atau membagikan data Tenant kepada pihak ketiga untuk kepentingan pemasaran.`,
      `Data pembayaran langganan diproses oleh mitra payment gateway pihak ketiga sebatas yang diperlukan untuk otorisasi transaksi. ElProof tidak menerima maupun menyimpan detail kartu atau rekening dari proses tersebut.`,
      `Data dapat diungkapkan apabila diwajibkan oleh ketentuan hukum yang berlaku.`,
    ],
  },
  {
    id: "penyimpanan-data",
    title: "4. Penyimpanan dan Retensi Data",
    body: [
      `Data project, vendor, dan dokumen bukti disimpan selama langganan Tenant aktif, untuk keperluan operasional layanan.`,
      `Apabila langganan berakhir tanpa perpanjangan, akses terhadap ElProof dapat dinonaktifkan sementara, namun data yang sudah tercatat tidak langsung dihapus akibat keterlambatan perpanjangan.`,
    ],
  },
  {
    id: "keamanan-akses",
    title: "5. Pemisahan dan Keamanan Akses",
    body: [
      `Setiap Tenant hanya dapat mengakses data project miliknya sendiri. Data antar-Tenant terpisah secara penuh dan tidak dapat diakses lintas akun bisnis.`,
      `Client hanya dapat melihat data project yang menjadi tanggung jawab Tenant yang mengundangnya, terbatas pada status milestone, ringkasan pembayaran, dan kendala yang relevan — tidak mencakup data internal Tenant lainnya.`,
      `ElProof menerapkan pemisahan akses berbasis peran (Staff, Client, admin platform) sehingga setiap pihak hanya melihat data yang menjadi haknya.`,
    ],
  },
  {
    id: "hak-atas-data",
    title: "6. Hak Anda atas Data",
    body: [
      `Staff dapat meminta akses, koreksi, atau penghapusan data akun melalui Owner Tenant-nya, atau langsung menghubungi tim ElProof lewat halaman Kontak.`,
      `Client dapat meminta koreksi data yang ditampilkan di Client Portal melalui Tenant yang mengelola project-nya, karena data tersebut diinput dan dimiliki oleh Tenant terkait.`,
    ],
  },
  {
    id: "cookie",
    title: "7. Cookie dan Sesi Login",
    body: [
      `ElProof hanya menggunakan cookie atau penyimpanan sesi yang esensial untuk menjaga status login pada WO Console, Client Portal, dan Platform Console. Kami tidak menggunakan cookie iklan atau pelacakan pihak ketiga untuk kepentingan pemasaran.`,
    ],
  },
  {
    id: "perubahan-kebijakan",
    title: "8. Perubahan Kebijakan",
    body: [
      `Kebijakan ini dapat diperbarui sewaktu-waktu mengikuti perkembangan layanan. Perubahan berlaku sejak dipublikasikan pada halaman ini.`,
    ],
  },
];

export default function PrivacyPage() {
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
              <ShieldAlert className="h-4 w-4" /> Dokumen Layanan
            </div>
            <h1 className="mt-6 font-display text-[36px] font-semibold tracking-tight text-navy-950 md:text-[48px]">
              Kebijakan Privasi
            </h1>
            <p className="mt-4 text-[15px] font-medium text-text-secondary">
              Terakhir diperbarui: 18 Juli 2026
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

              <div className="mt-8 rounded-2xl border border-border bg-surface-muted/60 p-8">
                <h3 className="text-[16px] font-bold text-navy-950">Pertanyaan seputar privasi data?</h3>
                <p className="mt-3 text-[14.5px] leading-relaxed text-text-secondary">
                  Hubungi tim kami melalui halaman{" "}
                  <Link to={ROUTE_PATHS.homepageContact} className="font-semibold text-navy-900 hover:underline">
                    Kontak
                  </Link>{" "}
                  untuk pertanyaan atau permintaan terkait data Anda.
                </p>
              </div>
            </Reveal>
          </div>
        </div>
      </div>
    </div>
  );
}
