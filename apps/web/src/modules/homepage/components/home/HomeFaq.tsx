import { Link } from "react-router-dom";
import { ChevronDown } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

// The questions a WO weighing ElProof asks first. Answers are taken from the FAQ page (FaqPage.tsx) and worded with
// the glossary (usaha Anda, tim, klien); change both together.
const QUESTIONS = [
  {
    question: "Bagaimana cara mencoba ElProof?",
    answer:
      "Daftar dengan akun Google, lalu siapkan workspace usaha Anda. Workspace baru mendapat masa coba gratis 14 hari, tanpa kartu kredit dan tanpa tagihan otomatis.",
  },
  {
    question: "Bagaimana cara membayar langganan?",
    answer:
      "Pemilik usaha membayar dari menu Langganan di WO Console: pilih paket, lalu pindai kode QRIS yang muncul dengan aplikasi m-banking atau e-wallet. Langganan aktif begitu pembayaran diterima.",
  },
  {
    question: "Apa yang terjadi jika langganan saya berakhir?",
    answer:
      "Akses dikurangi bertahap, bukan langsung diputus: 7 hari pertama tetap penuh dengan pengingat, 23 hari berikutnya hanya baca dan ekspor data, setelah itu tim Anda hanya bisa membuka halaman Langganan untuk membayar. Portal Klien tetap bisa dibaca, dan data yang sudah tercatat tidak dihapus. Akses penuh kembali begitu pembayaran diterima.",
  },
  {
    question: "Apakah data proyek saya aman?",
    answer:
      "Data setiap usaha terpisah penuh dan tidak bisa diakses dari akun usaha lain. Akses juga dibatasi per peran (tim, klien, admin ElProof), jadi setiap pihak hanya melihat data yang menjadi haknya.",
  },
  {
    question: "Bagaimana klien melihat perkembangan persiapannya?",
    answer:
      "Lewat Portal Klien, yang dibuka dari tautan WhatsApp di HP tanpa memasang aplikasi. Klien hanya melihat yang Anda pilih untuk ditampilkan dan tidak bisa mengubah apa pun.",
  },
];

/** A short FAQ on the Beranda: native disclosure, works without JavaScript. */
export function HomeFaq() {
  return (
    <div>
      <div className="border-t border-foreground">
        {QUESTIONS.map((item) => (
          <details key={item.question} className="group border-b border-border">
            <summary className="flex min-h-14 cursor-pointer list-none items-center justify-between gap-4 rounded-md py-4 text-[17px] font-semibold focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring [&::-webkit-details-marker]:hidden">
              {item.question}
              <ChevronDown aria-hidden className="size-5 shrink-0 text-muted-foreground motion-safe:transition-transform group-open:rotate-180" />
            </summary>
            <p className="max-w-[68ch] pb-5 text-muted-foreground">{item.answer}</p>
          </details>
        ))}
      </div>
      <Link
        to={ROUTE_PATHS.homepageFaq}
        className="mt-6 inline-flex min-h-11 items-center rounded-md font-semibold text-foreground underline decoration-border underline-offset-4 hover:decoration-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
      >
        Lihat semua pertanyaan
      </Link>
    </div>
  );
}
