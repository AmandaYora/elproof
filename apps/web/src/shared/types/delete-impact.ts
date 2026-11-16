// Shared shape for the hard-delete "impact" endpoints (PLAN.md) — Vendor,
// Venue, and Staff each expose a `GET .../{id}/delete-impact` returning this
// same shape, naming exactly which projects would lose a data source so the
// confirmation dialog can say something concrete instead of a generic
// warning. Vendor Category has no equivalent (its delete is a hard block,
// not a dialog — see PLAN.md's carve-out reasoning).
export interface AffectedProject {
  id: string;
  name: string;
  /** How many projects the entry stands for: 1, or the hidden ones counted together ("2 proyek lain…"). */
  count?: number;
}

/** "2 proyek: Citra & Bima, Dewi & Arga" — how many projects use it and which, for a delete that is refused (QA FQ-02). */
export function describeAffectedProjects(list: AffectedProject[]): string {
  const total = list.reduce((n, p) => n + (p.count ?? 1), 0);
  return `${total} proyek: ${list.map((p) => p.name).join(", ")}`;
}

export interface RawAffectedProject {
  id: string;
  name: string;
}

export interface RawDeleteImpact {
  affectedProjects: RawAffectedProject[];
  /** Affected projects outside the caller's project scope: counted, never named (ADR-0026). */
  hiddenProjectCount?: number;
}

/**
 * The projects named by a refused delete (409 `vendor_in_use` / `venue_in_use`, QA FQ-02: its `data` has the same shape as
 * delete-impact), or null when the error is something else.
 */
export function inUseProjectsOf(err: unknown): AffectedProject[] | null {
  const res = (err as { response?: { status?: number; data?: { data?: RawDeleteImpact | null } } } | undefined)?.response;
  const data = res?.status === 409 ? res.data?.data : null;
  if (!data || !Array.isArray(data.affectedProjects)) return null;
  const list = toAffectedProjects(data);
  return list.length > 0 ? list : null;
}

export function toAffectedProjects(raw: RawDeleteImpact): AffectedProject[] {
  const named = raw.affectedProjects.map((p) => ({ id: p.id, name: p.name }));
  const hidden = raw.hiddenProjectCount ?? 0;
  // The warning lists names joined by commas: the projects one may not open are one more entry, as a count.
  return hidden > 0 ? [...named, { id: "lainnya", name: `${hidden} proyek lain yang tidak Anda pegang`, count: hidden }] : named;
}
