import { useEffect, useState, type ReactNode } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { Link2, MessageCircle, Plus } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Container } from "@/modules/homepage/components/Container";
import { ctaClass } from "@/modules/homepage/components/cta";
import { CONTACT } from "@/modules/homepage/data/contact";
import { cn } from "@/shared/lib/cn";
import { usePageMeta } from "@/shared/hooks/usePageMeta";

const WHATSAPP_URL = `https://wa.me/${CONTACT.phones[0].number}?text=${encodeURIComponent(
  "Halo ElProof, saya punya pertanyaan yang belum ada di halaman FAQ: ",
)}`;

const linkClass = "font-semibold text-foreground underline underline-offset-4 hover:text-primary-hover";

interface Faq {
  /** Stable anchor: /faq#<id> opens and scrolls to the question (shared over WhatsApp). Never rename. */
  id: string;
  question: string;
  answer: ReactNode;
}

const GROUPS: { id: string; title: string; items: Faq[] }[] = [
  {
    id: "akun-dan-harga",
    title: "Akun & harga",
    items: [
      {
        id: "apa-itu-elproof",
        question: "Apa itu ElProof?",
        answer: (
          <>
            ElProof adalah aplikasi berlangganan untuk wedding organizer (WO). Tim Anda mencatat persiapan setiap
            proyek pernikahan — vendor, jadwal, pembayaran, dan kendala — lalu sebagian informasinya bisa dilihat
            pasangan klien lewat Portal Klien.
          </>
        ),
      },
      {
        id: "siapa-pengguna",
        question: "Siapa yang bisa memakai ElProof?",
        answer: (
          <>
            ElProof dibuat untuk usaha wedding organizer. Setiap usaha punya satu workspace, dan pemiliknya bisa
            menambahkan anggota tim sesuai kebutuhan. Pasangan klien mendapat akses baca terbatas lewat Portal Klien.
          </>
        ),
      },
      {
        id: "cara-mulai",
        question: "Bagaimana cara mulai memakai ElProof?",
        answer: (
          <>
            Daftar sendiri dengan akun Google di{" "}
            <Link to={ROUTE_PATHS.signup} className={linkClass}>
              halaman Daftar
            </Link>
            . Workspace baru mendapat masa coba gratis 14 hari, tanpa kartu kredit dan tanpa tagihan otomatis. Lebih
            suka dibantu? Tim kami bisa menyiapkan workspace untuk Anda — hubungi kami lewat{" "}
            <Link to={ROUTE_PATHS.homepageContact} className={linkClass}>
              halaman Kontak
            </Link>
            .
          </>
        ),
      },
      {
        id: "metode-pembayaran",
        question: "Metode pembayaran apa saja yang didukung?",
        answer: (
          <>
            Pembayaran langganan diproses oleh mitra pembayaran pihak ketiga (payment gateway), dari menu Langganan di
            WO Console. ElProof tidak menyimpan data kartu atau rekening Anda — otorisasi pembayaran ditangani langsung
            oleh mitra tersebut.
          </>
        ),
      },
      {
        id: "langganan-berakhir",
        question: "Apa yang terjadi jika langganan saya berakhir?",
        answer: (
          <>
            Akses dikurangi bertahap, bukan langsung diputus: 7 hari pertama tetap penuh dengan pengingat, 23 hari
            berikutnya hanya baca dan ekspor data, setelah itu tim Anda hanya bisa membuka halaman Langganan untuk
            membayar. Portal Klien tetap bisa dibaca, dan data yang sudah tercatat tidak dihapus. Akses penuh kembali
            begitu pembayaran diterima.
          </>
        ),
      },
      {
        id: "refund",
        question: "Apakah saya bisa mengajukan refund?",
        answer: (
          <>
            Bisa, dalam kondisi tertentu — misalnya kesalahan penagihan sistem. Syarat, kondisi yang tidak bisa
            direfund, dan cara mengajukannya ada di{" "}
            <Link to={ROUTE_PATHS.homepageRefund} className={linkClass}>
              Kebijakan Refund
            </Link>
            .
          </>
        ),
      },
      {
        id: "menghubungi-tim",
        question: "Bagaimana cara menghubungi tim ElProof?",
        answer: (
          <>
            Lewat WhatsApp, telepon, atau email — semuanya ada di{" "}
            <Link to={ROUTE_PATHS.homepageContact} className={linkClass}>
              halaman Kontak
            </Link>
            . Jam layanan: {CONTACT.hours}. Di luar jam itu, pesan Anda tetap kami balas sesegera mungkin.
          </>
        ),
      },
    ],
  },
  {
    id: "data-dan-keamanan",
    title: "Data & keamanan",
    items: [
      {
        id: "keamanan-data",
        question: "Apakah data proyek saya aman?",
        answer: (
          <>
            Data setiap usaha terpisah penuh dan tidak bisa diakses dari akun usaha lain. Akses juga dibatasi per peran
            (tim, klien, admin ElProof), jadi setiap pihak hanya melihat data yang menjadi haknya. Rinciannya ada di{" "}
            <Link to={ROUTE_PATHS.homepagePrivacy} className={linkClass}>
              Kebijakan Privasi
            </Link>
            .
          </>
        ),
      },
      {
        id: "pemilik-data",
        question: "Siapa pemilik data yang saya masukkan?",
        answer: (
          <>
            Usaha Anda. Data proyek, vendor, klien, dan dokumen yang Anda unggah adalah milik Anda, dan tidak dihapus
            karena langganan berakhir. Anda bisa meminta ekspor atau penghapusan data lewat{" "}
            <Link to={ROUTE_PATHS.homepageContact} className={linkClass}>
              halaman Kontak
            </Link>
            .
          </>
        ),
      },
    ],
  },
  {
    id: "klien-dan-portal",
    title: "Klien & Portal",
    items: [
      {
        id: "portal-klien",
        question: "Bagaimana klien melihat perkembangan persiapannya?",
        answer: (
          <>
            Lewat Portal Klien. Klien masuk dengan nama pengguna dan kata sandi yang Anda berikan, lalu hanya bisa
            membaca data proyeknya sendiri — tidak bisa mengubah apa pun.
          </>
        ),
      },
      {
        id: "klien-berlangganan",
        question: "Apakah klien perlu berlangganan sendiri?",
        answer: (
          <>
            Tidak. Akses Portal Klien termasuk dalam langganan usaha Anda; klien tidak membuat akun berlangganan
            tersendiri.
          </>
        ),
      },
    ],
  },
];

const ALL_IDS = new Set(GROUPS.flatMap((g) => g.items.map((i) => i.id)));

/** Copies the shareable link of one question; falls back to putting it in the address bar. */
function CopyLink({ id }: { id: string }) {
  const navigate = useNavigate();
  const [note, setNote] = useState("");

  const copy = async () => {
    const url = `${window.location.origin}${ROUTE_PATHS.homepageFaq}#${id}`;
    try {
      await navigator.clipboard.writeText(url);
      setNote("Tautan disalin.");
    } catch {
      navigate(`#${id}`, { replace: true });
      setNote("Salin tautan dari bilah alamat.");
    }
  };

  return (
    <div className="mt-3 flex flex-wrap items-center gap-x-3">
      <button
        type="button"
        onClick={copy}
        className="-ml-2 inline-flex min-h-11 items-center gap-2 rounded-md px-2 text-[14px] font-medium text-muted-foreground outline-hidden hover:text-foreground focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-ring"
      >
        <Link2 aria-hidden className="h-4 w-4" /> Salin tautan pertanyaan
      </button>
      <span aria-live="polite" className="text-[14px] text-muted-foreground">
        {note}
      </span>
    </div>
  );
}

function FaqItem({ item, open, onToggle }: { item: Faq; open: boolean; onToggle: () => void }) {
  const buttonId = `${item.id}-tombol`;
  const panelId = `${item.id}-jawaban`;
  return (
    <div id={item.id} className="scroll-mt-24 border-b border-border">
      <h3>
        <button
          type="button"
          id={buttonId}
          aria-expanded={open}
          aria-controls={panelId}
          onClick={onToggle}
          className="flex min-h-14 w-full items-start justify-between gap-6 py-4 text-left text-[17px] font-semibold leading-snug tracking-[-0.01em] text-foreground outline-hidden focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-offset-2 focus-visible:outline-ring"
        >
          {item.question}
          <Plus
            aria-hidden
            className={cn(
              "mt-0.5 h-5 w-5 shrink-0 text-muted-foreground motion-safe:transition-transform motion-safe:duration-200",
              open && "rotate-45 text-foreground",
            )}
          />
        </button>
      </h3>
      <div id={panelId} role="region" aria-labelledby={buttonId} hidden={!open} className="pb-5">
        <p className="max-w-[32rem] text-[16.5px] leading-[1.7] text-foreground">{item.answer}</p>
        <CopyLink id={item.id} />
      </div>
    </div>
  );
}

export default function FaqPage() {
  usePageMeta({ title: "Pertanyaan umum", description: "Jawaban atas pertanyaan yang sering diajukan tentang ElProof, langganan, dan keamanan data." });
  const { hash } = useLocation();
  const [openIds, setOpenIds] = useState<Set<string>>(() => new Set());

  // /faq#<id>: open that question; MarketingLayout scrolls to it.
  useEffect(() => {
    const id = decodeURIComponent(hash.slice(1));
    if (ALL_IDS.has(id)) setOpenIds((prev) => (prev.has(id) ? prev : new Set(prev).add(id)));
  }, [hash]);

  const toggle = (id: string) =>
    setOpenIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  return (
    <div className="bg-background">
      <header className="border-b border-border bg-muted">
        <Container className="py-12 md:py-16">
          <div className="max-w-[60ch]">
            <h1 className="text-[34px] font-extrabold leading-[1.04] tracking-[-0.035em] text-foreground md:text-[48px]">
              Pertanyaan umum
            </h1>
            <p className="mt-4 text-[17px] leading-relaxed text-muted-foreground">
              Jawaban singkat soal langganan, keamanan data, dan Portal Klien.
            </p>
            <a href={WHATSAPP_URL} target="_blank" rel="noreferrer" className={ctaClass("outline", "md", "mt-6 bg-background focus-visible:outline-solid")}>
              <MessageCircle aria-hidden className="h-[18px] w-[18px]" /> Tanya lewat WhatsApp
            </a>
          </div>
        </Container>
      </header>

      <Container className="py-10 md:py-16">
        <div className="flex flex-col gap-12 md:gap-16">
          {GROUPS.map((group) => (
            <section
              key={group.id}
              aria-labelledby={`${group.id}-judul`}
              className="md:grid md:grid-cols-[220px_minmax(0,1fr)] md:gap-12 lg:grid-cols-[260px_minmax(0,1fr)] lg:gap-20"
            >
              <h2
                id={`${group.id}-judul`}
                className="mb-2 text-[22px] font-extrabold tracking-[-0.03em] text-foreground md:mb-0 md:pt-4"
              >
                {group.title}
              </h2>
              <div className="max-w-[44rem] border-t border-border">
                {group.items.map((item) => (
                  <FaqItem key={item.id} item={item} open={openIds.has(item.id)} onToggle={() => toggle(item.id)} />
                ))}
              </div>
            </section>
          ))}

          <aside className="md:grid md:grid-cols-[220px_minmax(0,1fr)] md:gap-12 lg:grid-cols-[260px_minmax(0,1fr)] lg:gap-20">
            <div className="max-w-[32rem] border-t-2 border-seal pt-6 md:col-start-2">
              <h2 className="text-[18px] font-bold tracking-[-0.02em] text-foreground">Pertanyaan Anda belum terjawab?</h2>
              <p className="mt-2 text-[16px] leading-relaxed text-muted-foreground">
                Kirim pesan lewat{" "}
                <a href={WHATSAPP_URL} target="_blank" rel="noreferrer" className={linkClass}>
                  WhatsApp
                </a>{" "}
                atau lihat cara lain menghubungi kami di{" "}
                <Link to={ROUTE_PATHS.homepageContact} className={linkClass}>
                  halaman Kontak
                </Link>
                .
              </p>
            </div>
          </aside>
        </div>
      </Container>
    </div>
  );
}
