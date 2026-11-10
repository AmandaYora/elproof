import { Suspense, useEffect, useId, useState } from "react";
import { Link, NavLink, Outlet, useLocation } from "react-router-dom";
import { Menu, X } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { APP_NAME } from "@/shared/constants/brand";
import { useDocumentSurface } from "@/shared/hooks/useDocumentSurface";
import { BrandMark } from "@/modules/homepage/components/BrandMark";
import { Container } from "@/modules/homepage/components/Container";
import { ctaClass } from "@/modules/homepage/components/cta";
import { CONTACT } from "@/modules/homepage/data/contact";
import { cn } from "@/shared/lib/cn";

// Sections of the Beranda the header can jump to (ids set by HomePage).
const NAV_LINKS = [
  { to: `${ROUTE_PATHS.homepage}#cara-kerja`, label: "Cara kerja" },
  { to: `${ROUTE_PATHS.homepage}#harga`, label: "Harga" },
  { to: ROUTE_PATHS.homepageAbout, label: "Tentang Kami" },
  { to: ROUTE_PATHS.homepageFaq, label: "FAQ" },
  { to: ROUTE_PATHS.homepageContact, label: "Kontak" },
];

const FOOTER_GROUPS = [
  {
    title: "Produk",
    links: [
      { to: `${ROUTE_PATHS.homepage}#cara-kerja`, label: "Cara kerja" },
      { to: `${ROUTE_PATHS.homepage}#harga`, label: "Harga" },
      { to: ROUTE_PATHS.homepageFaq, label: "FAQ" },
      { to: ROUTE_PATHS.login, label: "Masuk" },
    ],
  },
  {
    title: "Perusahaan",
    links: [
      { to: ROUTE_PATHS.homepageAbout, label: "Tentang Kami" },
      { to: ROUTE_PATHS.homepageContact, label: "Kontak" },
    ],
  },
  {
    title: "Legal",
    links: [
      { to: ROUTE_PATHS.homepageTerms, label: "Syarat & Ketentuan" },
      { to: ROUTE_PATHS.homepagePrivacy, label: "Kebijakan Privasi" },
      { to: ROUTE_PATHS.homepageDataProcessing, label: "Pemrosesan Data" },
      { to: ROUTE_PATHS.homepageRefund, label: "Kebijakan Refund" },
    ],
  },
];

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    "rounded-full px-3.5 py-2 text-[14.5px] font-medium transition-colors outline-none focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-offset-2 focus-visible:outline-ring",
    isActive ? "text-foreground" : "text-muted-foreground hover:bg-muted hover:text-foreground",
  );

/** A header link; section links (#…) are never shown as the current page. */
function NavItem({ to, label, className }: { to: string; label: string; className?: string }) {
  if (to.includes("#")) {
    return (
      <Link to={to} className={cn(navLinkClass({ isActive: false }), className)}>
        {label}
      </Link>
    );
  }
  return (
    <NavLink to={to} end className={({ isActive }) => cn(navLinkClass({ isActive }), className)}>
      {label}
    </NavLink>
  );
}

/**
 * Scrolls to #id after navigation (react-router does not), or to the top on a new page. Keyed on location.key so a
 * second click on the same link (react-router replaces the entry with a new key) scrolls again. The page and its
 * lazy sections may still be loading: look for the target for up to 5 s, then scroll once more after the layout
 * has settled, since a lazy section above the target can change its position.
 */
function useScrollOnNavigate() {
  const { pathname, hash, key } = useLocation();
  useEffect(() => {
    if (!hash) {
      window.scrollTo(0, 0);
      return;
    }
    const id = decodeURIComponent(hash.slice(1));
    let settle: number | undefined;
    let tries = 0;
    const poll = window.setInterval(() => {
      const el = document.getElementById(id);
      if (el || ++tries > 100) {
        window.clearInterval(poll);
        if (!el) return;
        el.scrollIntoView({ block: "start" });
        settle = window.setTimeout(() => document.getElementById(id)?.scrollIntoView({ block: "start" }), 700);
      }
    }, 50);
    return () => {
      window.clearInterval(poll);
      window.clearTimeout(settle);
    };
  }, [pathname, hash, key]);
}

export default function MarketingLayout() {
  useDocumentSurface("marketing");
  useScrollOnNavigate();
  const [menuOpen, setMenuOpen] = useState(false);
  const location = useLocation();
  const menuId = useId();

  // Any navigation closes the phone menu, also a tap on the page that is already open (same path, new key).
  useEffect(() => {
    setMenuOpen(false);
  }, [location.key]);

  return (
    <div className="flex min-h-screen flex-col bg-background font-sans text-[16px] leading-relaxed text-foreground">
      <a
        href="#isi"
        className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-3 focus:z-(--z-banner) focus:rounded-md focus:bg-card focus:px-4 focus:py-2 focus:text-[14px] focus:font-semibold"
      >
        Lewati ke isi
      </a>
      <header className="sticky top-0 z-(--z-sticky) border-b border-border bg-background">
        <Container className="flex h-16 items-center justify-between gap-4 md:h-[68px]">
          <BrandMark />

          <nav aria-label="Utama" className="hidden items-center gap-1 lg:flex">
            {NAV_LINKS.map((link) => (
              <NavItem key={link.to} to={link.to} label={link.label} />
            ))}
          </nav>

          <div className="flex items-center gap-2">
            <Link to={ROUTE_PATHS.login} className={cn(navLinkClass({ isActive: false }), "hidden sm:inline-flex")}>
              Masuk
            </Link>
            <Link to={ROUTE_PATHS.signup} className={ctaClass("primary", "sm")}>
              Coba gratis
            </Link>
            <button
              type="button"
              onClick={() => setMenuOpen((v) => !v)}
              aria-expanded={menuOpen}
              aria-controls={menuId}
              aria-label={menuOpen ? "Tutup menu" : "Buka menu"}
              className="grid size-11 place-items-center rounded-full text-foreground outline-none hover:bg-muted focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-ring lg:hidden"
            >
              {menuOpen ? <X className="size-5" aria-hidden /> : <Menu className="size-5" aria-hidden />}
            </button>
          </div>
        </Container>

        <nav
          id={menuId}
          aria-label="Menu"
          hidden={!menuOpen}
          className="border-t border-border bg-background pb-5 lg:hidden"
        >
          <Container className="flex flex-col gap-1 pt-3">
            {NAV_LINKS.map((link) => (
              <NavItem key={link.to} to={link.to} label={link.label} className="rounded-lg px-3 py-3" />
            ))}
            <Link to={ROUTE_PATHS.login} className={cn(ctaClass("outline"), "mt-3 w-full")}>
              Masuk ke {APP_NAME}
            </Link>
          </Container>
        </nav>
      </header>

      <main id="isi" className="flex-1">
        <Suspense fallback={<div className="min-h-[60vh]" aria-busy="true" />}>
          <Outlet />
        </Suspense>
      </main>

      <footer className="border-t border-border bg-background pt-14 pb-8 text-[14.5px] text-muted-foreground md:pt-16">
        <Container>
          <div className="grid grid-cols-2 gap-x-6 gap-y-10 md:grid-cols-[1.6fr_1fr_1fr_1.2fr_1.4fr]">
            <div className="col-span-2 md:col-span-1">
              <BrandMark />
              <p className="mt-4 max-w-[30ch]">Aplikasi kerja untuk wedding organizer Indonesia.</p>
            </div>
            {FOOTER_GROUPS.map((group) => (
              <div key={group.title}>
                <h2 className="text-[14px] font-semibold text-foreground">{group.title}</h2>
                <ul className="mt-3 flex flex-col gap-1">
                  {group.links.map((link) => (
                    <li key={link.to}>
                      <Link to={link.to} className="inline-block py-1.5 hover:text-foreground">
                        {link.label}
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
            <div className="col-span-2 md:col-span-1">
              <h2 className="text-[14px] font-semibold text-foreground">Hubungi kami</h2>
              <ul className="mt-3 flex flex-col gap-1">
                <li>
                  <a href={`mailto:${CONTACT.email}`} className="inline-block py-1.5 hover:text-foreground">
                    {CONTACT.email}
                  </a>
                </li>
                {CONTACT.phones.map((phone) => (
                  <li key={phone.number}>
                    <a href={`https://wa.me/${phone.number}`} className="inline-block py-1.5 hover:text-foreground">
                      WhatsApp {phone.display}
                    </a>
                  </li>
                ))}
                <li className="pt-1.5 leading-relaxed">{CONTACT.address}</li>
              </ul>
            </div>
          </div>
          <p className="mt-12 border-t border-border pt-6 text-[13.5px]">
            © {new Date().getFullYear()} {APP_NAME}. Seluruh hak cipta dilindungi.
          </p>
        </Container>
      </footer>
    </div>
  );
}
