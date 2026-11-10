/**
 * The "Cara kerja" story on the Beranda: one wedding from contract to the day
 * itself. Shared by the static fallback (page chunk) and the interactive
 * story (lazy chunk), so the step text is there before the chunk arrives.
 */
export interface StoryStep {
  when: string;
  title: string;
  body: string;
}

export const STORY_STEPS: StoryStep[] = [
  {
    when: "H-180",
    title: "Kontrak dan termin dibuat sekali",
    body: "Isi data pasangan, paket, dan jadwal termin. Jadwal persiapan terisi dari templat milik Anda, lalu disesuaikan.",
  },
  {
    when: "H-120",
    title: "Vendor dan venue tercatat per proyek",
    body: "Kontrak dan pembayaran vendor tersimpan di proyeknya. Biaya vendor tidak pernah tampil di Portal Klien.",
  },
  {
    when: "H-32",
    title: "Setiap pembayaran punya bukti",
    body: "Klien mengunggah bukti transfer, tim memverifikasi, dan status yang sama muncul di kedua sisi.",
  },
  {
    when: "Hari H",
    title: "Hari H tanpa kejutan",
    body: "Tahapan, dokumen, dan pembayaran sudah lengkap, dan pasangan sudah tahu itu sejak jauh hari.",
  },
];
