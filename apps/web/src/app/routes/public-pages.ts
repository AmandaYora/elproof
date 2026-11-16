import { ROUTE_PATHS } from "@/app/routes/route-paths";

/** What a public page says about itself in search results and link previews. */
export interface PublicPageMeta {
  /** The page's path, also its canonical address (with SITE_URL in front). */
  path: string;
  /** Page name; the tab reads "<title> — ElProof". */
  title: string;
  /** One or two sentences for search results and link previews. */
  description: string;
  /** Kept out of search results and out of sitemap.xml (sign-in). */
  noindex?: boolean;
}

/**
 * The pages a visitor can open before signing in that are written to HTML at build time (ADR-0043). Each page passes its
 * entry to usePageMeta, and scripts/prerender.mjs writes the same title, description, canonical and share tags into the
 * page's HTML, so a crawler that runs no JavaScript reads what the browser shows. Siapkan workspace and Undangan are not
 * here: they need a Google credential or a token and are never indexed.
 */
export const PUBLIC_PAGES = {
  home: {
    path: ROUTE_PATHS.homepage,
    title: "Aplikasi wedding organizer: kerja tim lebih ringan, klien makin percaya",
    description:
      "Catat tahapan, vendor, dan pembayaran klien di satu tempat. Pasangan memantau persiapan sendiri lewat Portal Klien, tim Anda punya waktu untuk lebih banyak acara.",
  },
  about: {
    path: ROUTE_PATHS.homepageAbout,
    title: "Tentang kami",
    description:
      "Kenapa ElProof dibuat: agar wedding organizer bisa menunjukkan pembayaran, vendor, dan progres persiapan kepada klien, lengkap dengan buktinya.",
  },
  faq: {
    path: ROUTE_PATHS.homepageFaq,
    title: "Pertanyaan umum",
    description: "Jawaban singkat soal ElProof: untuk siapa, cara mulai dan coba gratis, harga dan cara membayar, keamanan data, dan Portal Klien.",
  },
  contact: {
    path: ROUTE_PATHS.homepageContact,
    title: "Hubungi kami",
    description: "Hubungi tim ElProof untuk pertanyaan, bantuan, atau permintaan terkait data Anda.",
  },
  terms: {
    path: ROUTE_PATHS.homepageTerms,
    title: "Syarat dan ketentuan",
    description: "Syarat dan ketentuan penggunaan layanan ElProof untuk wedding organizer dan klien mereka.",
  },
  privacy: {
    path: ROUTE_PATHS.homepagePrivacy,
    title: "Kebijakan privasi",
    description: "Cara ElProof mengumpulkan, memakai, dan melindungi data pribadi sesuai UU Pelindungan Data Pribadi.",
  },
  dataProcessing: {
    path: ROUTE_PATHS.homepageDataProcessing,
    title: "Perjanjian pemrosesan data",
    description: "Perjanjian pemrosesan data antara wedding organizer sebagai pengendali data dan ElProof sebagai prosesor.",
  },
  refund: {
    path: ROUTE_PATHS.homepageRefund,
    title: "Kebijakan refund",
    description: "Ketentuan pengembalian dana untuk langganan ElProof.",
  },
  signup: {
    path: ROUTE_PATHS.signup,
    title: "Daftar gratis",
    description: "Daftarkan wedding organizer Anda dan coba ElProof gratis selama masa percobaan, tanpa kartu kredit.",
  },
  login: {
    path: ROUTE_PATHS.login,
    title: "Masuk",
    description: "Masuk ke ElProof sebagai tim WO, klien, atau admin platform.",
    noindex: true,
  },
} as const satisfies Record<string, PublicPageMeta>;
