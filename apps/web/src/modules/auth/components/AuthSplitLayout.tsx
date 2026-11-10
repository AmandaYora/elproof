import type { ReactNode } from "react";
import { LoginPhotoSlider } from "@/modules/auth/components/LoginPhotoSlider";
import { BrandMark } from "@/modules/homepage/components/BrandMark";
import { ProofSeal } from "@/modules/homepage/components/ProofSeal";
import { useDocumentSurface } from "@/shared/hooks/useDocumentSurface";

/**
 * Two-panel frame for every screen before sign-in (Masuk, Daftar, Siapkan
 * workspace, Undangan), on the public site's surface (ADR-0038): white page,
 * same plum and seal gold as every console (ADR-0039).
 *
 * Left (large screens): the photos chosen in Platform Console, under a plum
 * scrim so the words stay readable; without photos the plum panel carries the
 * seal instead of an empty gradient. Right: the form, which comes first on a
 * phone so the keyboard never covers it. `aside` is decorative text (not a
 * heading); each page puts its own h1 in the form column.
 */
export function AuthSplitLayout({ aside, children }: { aside: ReactNode; children: ReactNode }) {
  useDocumentSurface("marketing");
  return (
    <div className="flex min-h-screen bg-background text-foreground">
      <div className="relative hidden overflow-hidden bg-primary lg:flex lg:w-[46%] lg:flex-col lg:justify-between lg:p-14">
        <div className="pointer-events-none absolute -top-20 -right-20 opacity-30" aria-hidden>
          <ProofSeal size={360} className="text-seal" />
        </div>
        <LoginPhotoSlider />
        <div className="pointer-events-none absolute inset-0 bg-gradient-to-t from-primary via-primary/55 to-primary/25" aria-hidden />
        <BrandMark inverted className="relative z-10 self-start" />
        <div className="relative z-10 max-w-md text-primary-foreground">{aside}</div>
      </div>
      <main className="flex w-full flex-col items-center px-4 py-8 sm:px-6 lg:w-[54%] lg:justify-center lg:py-12">
        <BrandMark className="mb-8 self-start sm:self-center lg:hidden" />
        <div className="w-full max-w-[440px]">{children}</div>
      </main>
    </div>
  );
}

/** The quiet panel text: a large statement and one sentence under it. */
export function AuthAside({ title, body }: { title: string; body: string }) {
  return (
    <>
      <p className="text-[36px] leading-[1.08] font-extrabold tracking-[-0.03em]">{title}</p>
      <div className="mt-5 h-px w-12 bg-seal" aria-hidden />
      <p className="mt-5 text-[16px] leading-relaxed text-on-primary-muted">{body}</p>
    </>
  );
}

/** The form surface on the right. */
export function AuthCard({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <div className={`rounded-2xl border border-border bg-card p-6 sm:p-9 ${className}`}>{children}</div>;
}
