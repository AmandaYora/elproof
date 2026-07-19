import { Suspense, useState, useEffect } from "react";
import { NavLink, Outlet, Link, useLocation } from "react-router-dom";
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
  const [scrolled, setScrolled] = useState(false);
  const location = useLocation();

  useEffect(() => {
    const handleScroll = () => {
      setScrolled(window.scrollY > 10);
    };
    window.addEventListener("scroll", handleScroll);
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  // Close menu on route change
  useEffect(() => {
    setMenuOpen(false);
  }, [location.pathname]);

  return (
    <div className="flex min-h-screen flex-col bg-background font-sans text-text-primary selection:bg-navy-900/10">
      <header
        className={cn(
          "fixed inset-x-0 top-0 z-50 transition-all duration-300",
          scrolled
            ? "border-b border-navy-900/5 bg-white/80 py-2.5 backdrop-blur-lg shadow-[0_1px_3px_0_rgba(0,0,0,0.02)]"
            : "border-b border-transparent bg-transparent py-4"
        )}
      >
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6">
          <Link to={ROUTE_PATHS.homepage} className="group flex items-center gap-2.5 outline-none">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-tr from-navy-900 to-navy-800 text-[13px] font-bold text-white shadow-sm transition-transform duration-300 group-hover:scale-105 group-hover:shadow-md">
              EP
            </span>
            <span className="font-display text-[19px] font-semibold tracking-tight text-navy-950 transition-colors group-hover:text-navy-900">
              {APP_NAME}
            </span>
          </Link>

          <nav className="hidden items-center gap-1 md:flex">
            {NAV_LINKS.map((link) => (
              <NavLink
                key={link.to}
                to={link.to}
                end
                className={({ isActive }) =>
                  cn(
                    "rounded-full px-4 py-2 text-[13.5px] font-medium transition-all duration-200 outline-none focus-visible:ring-2 focus-visible:ring-navy-900/20",
                    isActive
                      ? "bg-navy-900/5 text-navy-950"
                      : "text-text-secondary hover:bg-navy-900/5 hover:text-navy-950"
                  )
                }
              >
                {link.label}
              </NavLink>
            ))}
            <Link
              to={ROUTE_PATHS.login}
              className="ml-3 inline-flex h-9 items-center rounded-full bg-navy-900 px-5 text-[13.5px] font-semibold text-white shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:bg-navy-800 hover:shadow-md outline-none focus-visible:ring-2 focus-visible:ring-navy-900/20 focus-visible:ring-offset-2"
            >
              Masuk
            </Link>
          </nav>

          <button
            type="button"
            onClick={() => setMenuOpen((v) => !v)}
            aria-label={menuOpen ? "Tutup menu" : "Buka menu"}
            className="flex h-10 w-10 items-center justify-center rounded-full text-navy-950 transition-colors hover:bg-navy-900/5 md:hidden"
          >
            {menuOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
          </button>
        </div>

        {menuOpen && (
          <nav className="absolute inset-x-0 top-full flex flex-col gap-1 border-b border-navy-900/10 bg-white/95 px-6 pb-6 pt-3 shadow-lg backdrop-blur-xl md:hidden">
            {NAV_LINKS.map((link) => (
              <NavLink
                key={link.to}
                to={link.to}
                end
                className={({ isActive }) =>
                  cn(
                    "rounded-xl px-4 py-3 text-[14.5px] font-medium transition-colors",
                    isActive ? "bg-navy-900/5 text-navy-950" : "text-text-secondary"
                  )
                }
              >
                {link.label}
              </NavLink>
            ))}
            <Link
              to={ROUTE_PATHS.login}
              className="mt-3 flex h-11 items-center justify-center rounded-xl bg-navy-900 text-[14.5px] font-semibold text-white shadow-sm"
            >
              Masuk ke {APP_NAME}
            </Link>
          </nav>
        )}
      </header>

      {/* spacer to prevent content from jumping because header is fixed */}
      <div className="h-16 md:h-20" />

      <main className="flex-1">
        <Suspense fallback={<div className="flex min-h-[50vh] items-center justify-center text-[14px] text-text-secondary">Memuat halaman...</div>}>
          <Outlet />
        </Suspense>
      </main>

      <footer className="border-t border-border bg-white pt-16 pb-8 md:pt-24">
        <div className="mx-auto grid max-w-6xl gap-12 px-6 md:grid-cols-[1.5fr_1fr_1fr_1.5fr]">
          <div>
            <Link to={ROUTE_PATHS.homepage} className="flex items-center gap-2.5 outline-none">
              <ProofSeal size={36} className="text-navy-900" />
              <span className="font-display text-[20px] font-semibold tracking-tight text-navy-950">{APP_NAME}</span>
            </Link>
            <p className="mt-5 text-[14px] leading-relaxed text-text-secondary">
              Satu portal transparansi untuk wedding organizer, vendor, dan pasangan client — setiap progress,
              pembayaran, dan kendala tercatat secara profesional dan modern.
            </p>
          </div>

          <div>
            <h3 className="text-[13px] font-semibold uppercase tracking-wider text-navy-950">Navigasi</h3>
            <ul className="mt-5 flex flex-col gap-3.5 text-[14px]">
              {NAV_LINKS.map((link) => (
                <li key={link.to}>
                  <Link to={link.to} className="text-text-secondary transition-colors hover:text-navy-900 font-medium">
                    {link.label}
                  </Link>
                </li>
              ))}
              <li>
                <Link to={ROUTE_PATHS.login} className="text-navy-900 transition-colors hover:text-navy-800 font-semibold">
                  Masuk ke Portal
                </Link>
              </li>
            </ul>
          </div>

          <div>
            <h3 className="text-[13px] font-semibold uppercase tracking-wider text-navy-950">Legal</h3>
            <ul className="mt-5 flex flex-col gap-3.5 text-[14px]">
              {LEGAL_LINKS.map((link) => (
                <li key={link.to}>
                  <Link to={link.to} className="text-text-secondary transition-colors hover:text-navy-900 font-medium">
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
          </div>

          <div>
            <h3 className="text-[13px] font-semibold uppercase tracking-wider text-navy-950">Hubungi Kami</h3>
            <ul className="mt-5 flex flex-col gap-3.5 text-[14px] text-text-secondary">
              <li>
                <a href={`mailto:${CONTACT.email}`} className="transition-colors hover:text-navy-900 font-medium">
                  {CONTACT.email}
                </a>
              </li>
              {CONTACT.phones.map((phone) => (
                <li key={phone.number}>
                  <a href={`tel:+${phone.number}`} className="transition-colors hover:text-navy-900 font-medium">
                    {phone.display}
                  </a>
                </li>
              ))}
              <li className="leading-relaxed">{CONTACT.address}</li>
            </ul>
          </div>
        </div>

        <div className="mx-auto mt-16 max-w-6xl border-t border-border px-6 pt-8 text-center md:flex md:items-center md:justify-between md:text-left">
          <p className="text-[13px] font-medium text-text-secondary">
            © {new Date().getFullYear()} {APP_NAME}. Seluruh hak cipta dilindungi.
          </p>
          <div className="mt-4 flex items-center justify-center gap-4 md:mt-0">
            <span className="text-[12px] text-text-secondary/60">Didesain dengan indah untuk Anda.</span>
          </div>
        </div>
      </footer>
    </div>
  );
}
