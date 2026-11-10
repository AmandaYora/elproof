import { Link } from "react-router-dom";
import { ChevronDown } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

// Answers taken from the FAQ page, worded with the glossary (usaha Anda, tim, klien).
const QUESTIONS = [
  {
    question: "Bagaimana cara membayar langganan?",
    answer:
      "Pembayaran langganan diproses lewat mitra payment gateway pihak ketiga. ElProof tidak menyimpan data kartu atau rekening Anda; otorisasi pembayaran ditangani langsung oleh mitra tersebut.",
  },
  {
    question: "Apa yang terjadi jika masa langganan habis?",
    answer:
      "Akses dikurangi bertahap, bukan langsung diputus: 7 hari pertama tetap penuh dengan pengingat, 23 hari berikutnya hanya baca dan ekspor data, setelah itu tim Anda hanya bisa membuka halaman Langganan untuk membayar. Portal Klien tetap bisa dibaca, dan data yang sudah tercatat tidak dihapus. Akses penuh kembali begitu pembayaran diterima.",
  },
  {
    question: "Apakah data proyek saya aman?",
    answer:
      "Data setiap usaha terpisah penuh dan tidak bisa dibuka dari akun usaha lain. Akses dibagi menurut peran (tim, klien, admin platform), jadi setiap pihak hanya melihat data yang menjadi haknya.",
  },
  {
    question: "Apakah saya bisa mengajukan refund?",
    answer:
      "Bisa, dalam kondisi tertentu, misalnya kesalahan penagihan sistem. Syarat, kondisi yang tidak dapat direfund, dan cara mengajukannya ada di halaman Kebijakan Refund.",
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
