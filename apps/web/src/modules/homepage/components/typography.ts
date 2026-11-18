/**
 * Heading scale of the public site (knowledge/DESIGN-PUBLIC.md › Huruf). One place, so the pages do not drift apart.
 * Bold (700), not extrabold, with modest negative tracking: the product screens are the loud part of the page, the
 * headings only lead into them. Ratio ≈ 1.25 between steps on desktop.
 */
export const TYPE = {
  /** The home hero h1 only. Two lines on desktop. */
  display: "text-[34px] leading-[1.12] font-bold tracking-[-0.025em] sm:text-[44px] lg:text-[52px]",
  /** h1 of every other public page (Tentang kami, Kontak, FAQ, legal documents). */
  pageTitle: "text-[30px] leading-[1.15] font-bold tracking-[-0.02em] text-foreground md:text-[40px]",
  /** h2 that opens a section of a page. */
  sectionTitle: "text-[26px] leading-[1.2] font-bold tracking-[-0.02em] md:text-[34px]",
  /** h2/h3 inside a section (a FAQ group, a numbered legal clause, a card heading). */
  subTitle: "text-[19px] leading-snug font-bold tracking-[-0.015em] text-foreground md:text-[21px]",
} as const;
