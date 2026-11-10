// Shared shape for the hard-delete "impact" endpoints (PLAN.md) — Vendor,
// Venue, and Staff each expose a `GET .../{id}/delete-impact` returning this
// same shape, naming exactly which projects would lose a data source so the
// confirmation dialog can say something concrete instead of a generic
// warning. Vendor Category has no equivalent (its delete is a hard block,
// not a dialog — see PLAN.md's carve-out reasoning).
export interface AffectedProject {
  id: string;
  name: string;
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

export function toAffectedProjects(raw: RawDeleteImpact): AffectedProject[] {
  const named = raw.affectedProjects.map((p) => ({ id: p.id, name: p.name }));
  const hidden = raw.hiddenProjectCount ?? 0;
  // The warning lists names joined by commas: the projects one may not open are one more entry, as a count.
  return hidden > 0 ? [...named, { id: "lainnya", name: `${hidden} proyek lain yang tidak Anda pegang` }] : named;
}
