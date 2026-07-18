import { Suspense, useState } from "react";
import { NavLink, Outlet, Link } from "react-router-dom";
import { Menu, X } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { APP_NAME } from "@/shared/constants/brand";
import { ProofSeal } from "@/modules/homepage/components/ProofSeal";
import { CONTACT } from "@/modules/homepage/data/contact";
import { cn } from "@/shared/lib/cn";

const NAV_LINKS = [
  { to: ROUTE_PATHS.homepage, label: "Beranda" },
  { to: ROUTE_PATHS.homepageAbout, label: "Tentang Kami" },
  { to: ROUTE_PATHS.homepageFaq, label: "FAQ" },
  { to: ROUTE_PATHS.homepageContact, label: "Kontak" },
];

const LEGAL_LINKS = [
  { to: ROUTE_PATHS.homepageTerms, label: "Syarat & Ketentuan" },
  { to: ROUTE_PATHS.homepagePrivacy, label: "Kebijakan Privasi" },
  { to: ROUTE_PATHS.homepageRefund, label: "Kebijakan Refund" },
];

export default function MarketingLayout() {
  const [menuOpen, setMenuOpen] = useState(false);

  return (
    <div className="flex min-h-screen flex-col bg-background">
      <header className="sticky top-0 z-30 border-b border-border/70 bg-surface/90 backdrop-blur">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-6">
          <Link to={ROUTE_PATHS.homepage} className="flex items-center gap-2.5">
            <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-navy-900 text-[13px] font-bold text-white">
              EP
            </span>
            <span className="font-display text-[19px] font-semibold text-navy-950">{APP_NAME}</span>
          </Link>

          <nav className="hidden items-center gap-1 md:flex">
            {NAV_LINKS.map((link) => (
              <NavLink
                key={link.to}
                to={link.to}
                end
                className={({ isActive }) =>
                  cn(
                    "rounded-md px-3.5 py-2 text-[13.5px] font-medium transition-colors",
                    isActive ? "text-navy-900" : "text-text-secondary hover:text-navy-900"
                  )
                }
              >
                {link.label}
              </NavLink>
            ))}
            <Link
              to={ROUTE_PATHS.login}
              className="ml-2 inline-flex h-9 items-center rounded-md bg-navy-900 px-4 text-[13.5px] font-semibold text-white transition-colors hover:bg-navy-800"
            >
              Masuk
            </Link>
          </nav>

          <button
            type="button"
            onClick={() => setMenuOpen((v) => !v)}
            aria-label={menuOpen ? "Tutup menu" : "Buka menu"}
            className="flex h-9 w-9 items-center justify-center rounded-md text-navy-950 md:hidden"
          >
            {menuOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
          </button>
        </div>

        {menuOpen && (
          <nav className="flex flex-col gap-0.5 border-t border-border/70 px-6 py-3 md:hidden">
            {NAV_LINKS.map((link) => (
              <NavLink
                key={link.to}
                to={link.to}
                end
                onClick={() => setMenuOpen(false)}
                className={({ isActive }) =>
                  cn(
                    "rounded-md px-3 py-2.5 text-[14px] font-medium",
                    isActive ? "bg-surface-muted text-navy-900" : "text-text-secondary"
                  )
                }
              >
                {link.label}
              </NavLink>
            ))}
            <Link
              to={ROUTE_PATHS.login}
              className="mt-1 inline-flex h-10 items-center justify-center rounded-md bg-navy-900 text-[14px] font-semibold text-white"
            >
              Masuk
            </Link>
          </nav>
        )}
      </header>

      <main className="flex-1">
        <Suspense fallback={<div className="py-24 text-center text-sm text-text-secondary">Memuat halaman...</div>}>
          <Outlet />
        </Suspense>
      </main>

      <footer className="border-t border-white/10 bg-navy-950 text-white/70">
        <div className="mx-auto grid max-w-6xl gap-10 px-6 py-14 md:grid-cols-[1.2fr_0.8fr_0.9fr_1.1fr_1fr]">
          <div>
            <div className="flex items-center gap-2.5">
              <ProofSeal size={40} className="text-warning" />
              <span className="font-display text-[19px] font-semibold text-white">{APP_NAME}</span>
            </div>
            <p className="mt-4 max-w-sm text-[13.5px] leading-relaxed text-white/60">
              Satu portal transparansi untuk wedding organizer, vendor, dan pasangan client — setiap progress,
              pembayaran, dan kendala tercatat dan bisa dilihat pihak yang berhak, kapan saja.
            </p>
          </div>

          <div>
            <p className="text-[12.5px] font-semibold uppercase tracking-wide text-white/40">Navigasi</p>
            <ul className="mt-3.5 flex flex-col gap-2.5 text-[13.5px]">
              {NAV_LINKS.map((link) => (
                <li key={link.to}>
                  <Link to={link.to} className="hover:text-white">
                    {link.label}
                  </Link>
                </li>
              ))}
              <li>
                <Link to={ROUTE_PATHS.login} className="hover:text-white">
                  Masuk ke ElProof
                </Link>
              </li>
            </ul>
          </div>

          <div>
            <p className="text-[12.5px] font-semibold uppercase tracking-wide text-white/40">Legal</p>
            <ul className="mt-3.5 flex flex-col gap-2.5 text-[13.5px]">
              {LEGAL_LINKS.map((link) => (
                <li key={link.to}>
                  <Link to={link.to} className="hover:text-white">
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
          </div>

          <div>
            <p className="text-[12.5px] font-semibold uppercase tracking-wide text-white/40">Untuk Wedding Organizer</p>
            <p className="mt-3.5 text-[13.5px] leading-relaxed text-white/60">
              Tim kami membantu proses aktivasi akun bisnis Anda secara langsung —{" "}
              <Link to={ROUTE_PATHS.homepageContact} className="font-medium text-white underline underline-offset-2">
                hubungi kami
              </Link>{" "}
              untuk mulai.
            </p>
          </div>

          <div>
            <p className="text-[12.5px] font-semibold uppercase tracking-wide text-white/40">Kontak</p>
            <ul className="mt-3.5 flex flex-col gap-2.5 text-[13.5px] text-white/60">
              <li>
                <a href={`mailto:${CONTACT.email}`} className="hover:text-white">
                  {CONTACT.email}
                </a>
              </li>
              {CONTACT.phones.map((phone) => (
                <li key={phone.number}>
                  <a href={`tel:+${phone.number}`} className="hover:text-white">
                    {phone.display}
                  </a>
                </li>
              ))}
              <li className="leading-relaxed">{CONTACT.address}</li>
            </ul>
          </div>
        </div>

        <div className="border-t border-white/10 px-6 py-5">
          <p className="mx-auto max-w-6xl text-[12.5px] text-white/40">
            © {new Date().getFullYear()} {APP_NAME}. Seluruh hak cipta dilindungi.
          </p>
        </div>
      </footer>
    </div>
  );
}
