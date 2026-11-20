import type { LineDraft, Rincian, RincianLine } from "@/modules/rincian/types";

/**
 * One line as the editor holds it while it is being changed. Every row has a local `key` (a saved line's is "l" + its
 * id, so a preview can be matched back); a component points at its package by `parentKey`. Prices are never worked
 * out here: what a row comes to is read from the latest preview or the saved rincian (PLAN §2.2, one formula on the
 * server).
 */
export interface EditorRow {
  key: string;
  /** "" while the line is not saved yet. */
  id: string;
  kind: "Package" | "Component" | "Item" | "Discount";
  parentKey: string;
  /** A package added from the catalog in this session (its components come from the catalog when it is saved). */
  packageId: string;
  description: string;
  eventId: string;
  categoryId: string;
  supplierRef: string;
  quantity: number;
  unit: string;
  unitPrice: number;
  amount: number;
  isBonus: boolean;
}

/** Which of the three groups of the table a row is drawn in (DESAIN §5.3: Bonus and Potongan are their own groups). */
export type RowGroup = "main" | "bonus" | "discount";

export function groupOf(row: Pick<EditorRow, "kind" | "isBonus">): RowGroup {
  if (row.kind === "Discount") return "discount";
  if (row.kind === "Item" && row.isBonus) return "bonus";
  return "main";
}

export const savedKey = (id: string) => `l${id}`;

let counter = 0;
/** A key for a row added in this session. */
export function newKey(): string {
  counter += 1;
  return `n${Date.now().toString(36)}${counter}`;
}

/** The rows of a saved rincian (the computed "Tambahan tamu" lines are the server's and are left out). */
export function rowsFromRincian(r: Rincian): EditorRow[] {
  const rows = r.lines
    .filter((l) => l.kind !== "ExtraGuests")
    .map(
      (l): EditorRow => ({
        key: savedKey(l.id),
        id: l.id,
        kind: l.kind as EditorRow["kind"],
        parentKey: l.parentId ? savedKey(l.parentId) : "",
        packageId: "",
        description: l.description,
        eventId: l.eventId,
        categoryId: l.categoryId,
        supplierRef: l.supplierRef ?? "",
        quantity: l.quantity,
        unit: l.unit,
        unitPrice: l.unitPrice ?? 0,
        amount: l.amount ?? 0,
        isBonus: l.isBonus,
      }),
    );
  return normalize(rows);
}

/**
 * The rows in the order the table and the save use: the main group (each package followed by its components, items
 * in between), then Bonus, then Potongan. Order inside a group is kept.
 */
export function normalize(rows: EditorRow[]): EditorRow[] {
  const components = new Map<string, EditorRow[]>();
  for (const r of rows) {
    if (r.kind === "Component") components.set(r.parentKey, [...(components.get(r.parentKey) ?? []), r]);
  }
  const out: EditorRow[] = [];
  for (const group of ["main", "bonus", "discount"] as RowGroup[]) {
    for (const r of rows) {
      if (r.kind === "Component" || groupOf(r) !== group) continue;
      out.push(r);
      if (r.kind === "Package") out.push(...(components.get(r.key) ?? []));
    }
  }
  return out;
}

/** The top-level rows of one group, in order (what "Pindah ke atas/bawah" moves between). */
export function topLevel(rows: EditorRow[], group: RowGroup): EditorRow[] {
  return rows.filter((r) => r.kind !== "Component" && groupOf(r) === group);
}

/** Moves a row one place up or down among its siblings (a component inside its package, anything else in its group). */
export function moveRow(rows: EditorRow[], key: string, direction: -1 | 1): { rows: EditorRow[]; position: number; count: number } | null {
  const row = rows.find((r) => r.key === key);
  if (!row) return null;
  const siblings =
    row.kind === "Component" ? rows.filter((r) => r.kind === "Component" && r.parentKey === row.parentKey) : topLevel(rows, groupOf(row));
  const at = siblings.findIndex((r) => r.key === key);
  const to = at + direction;
  if (to < 0 || to >= siblings.length) return null;
  const other = siblings[to];
  // Swap the two rows (with a package's components travelling along, normalize puts them back after it).
  const next = rows.map((r) => (r.key === key ? other : r.key === other.key ? row : r));
  return { rows: normalize(next), position: to + 1, count: siblings.length };
}

/** Removes a row (a package takes its components along); returns what was removed so it can be put back. */
export function removeRow(rows: EditorRow[], key: string): { rows: EditorRow[]; removed: EditorRow[]; index: number } {
  const index = rows.findIndex((r) => r.key === key);
  const removed = rows.filter((r) => r.key === key || r.parentKey === key);
  return { rows: rows.filter((r) => r.key !== key && r.parentKey !== key), removed, index };
}

/** Puts removed rows back where they were ("Urungkan"). */
export function restoreRows(rows: EditorRow[], removed: EditorRow[], index: number): EditorRow[] {
  const keys = new Set(rows.map((r) => r.key));
  const back = removed.filter((r) => !keys.has(r.key));
  if (back.length === 0) return rows;
  const at = Math.min(Math.max(index, 0), rows.length);
  return normalize([...rows.slice(0, at), ...back, ...rows.slice(at)]);
}

/** The drafts a save or a preview sends, in screen order. */
export function toDrafts(rows: EditorRow[]): LineDraft[] {
  const byKey = new Map(rows.map((r) => [r.key, r]));
  return normalize(rows).map((r): LineDraft => {
    const base = {
      key: r.key,
      description: r.description.trim(),
      eventId: r.eventId || undefined,
      categoryId: r.categoryId || undefined,
      supplierRef: r.supplierRef || undefined,
    };
    switch (r.kind) {
      case "Package":
        return r.id ? { ...base, id: r.id, kind: "Package" } : { ...base, kind: "Package", packageId: r.packageId };
      case "Component": {
        const parent = byKey.get(r.parentKey);
        return {
          ...base,
          ...(r.id ? { id: r.id } : {}),
          kind: "Component",
          ...(parent?.id ? { parentId: parent.id } : { parentKey: r.parentKey }),
        };
      }
      case "Item":
        return {
          ...base,
          ...(r.id ? { id: r.id } : {}),
          kind: "Item",
          quantity: r.quantity,
          unit: r.unit.trim(),
          unitPrice: r.isBonus ? 0 : r.unitPrice,
          isBonus: r.isBonus,
        };
      case "Discount":
        return { ...base, ...(r.id ? { id: r.id } : {}), kind: "Discount", amount: r.amount };
    }
  });
}

/** True when the rows say the same as the saved rincian (nothing to save). */
export function sameRows(a: EditorRow[], b: EditorRow[]): boolean {
  return JSON.stringify(toDrafts(a)) === JSON.stringify(toDrafts(b));
}

/** How the computed lines of a rincian are found again from the editor's rows. */
export interface ComputedIndex {
  /** A line by the row key it belongs to (preview: the echoed key; saved: "l" + id). */
  byKey: Map<string, RincianLine>;
  /** The components of a package by the package's row key (a new package's come from the catalog). */
  componentsOf: Map<string, RincianLine[]>;
  /** The "Tambahan tamu" line of a package by the package's row key. */
  extraOf: Map<string, RincianLine>;
}

export function indexRincian(r: Rincian | null): ComputedIndex {
  const index: ComputedIndex = { byKey: new Map(), componentsOf: new Map(), extraOf: new Map() };
  if (!r) return index;
  const keyOf = (l: RincianLine) => l.key || (l.id ? savedKey(l.id) : "");
  const parentKeyOf = (l: RincianLine) => l.parentKey || (l.parentId ? savedKey(l.parentId) : "");
  for (const l of r.lines) {
    const key = keyOf(l);
    if (key) index.byKey.set(key, l);
    if (l.kind === "Component") index.componentsOf.set(parentKeyOf(l), [...(index.componentsOf.get(parentKeyOf(l)) ?? []), l]);
    if (l.kind === "ExtraGuests") index.extraOf.set(parentKeyOf(l), l);
  }
  return index;
}

/** The sums of the "Ringkasan biaya" panel per kind of line, from the computed lines (null when amounts are hidden). */
export interface KindSums {
  packages: number;
  extraGuests: number;
  items: number;
  discount: number;
}

export function kindSums(r: Rincian | null): KindSums | null {
  if (!r || !r.totals) return null;
  const sums: KindSums = { packages: 0, extraGuests: 0, items: 0, discount: 0 };
  for (const l of r.lines) {
    const amount = l.amount ?? 0;
    if (l.kind === "Package") sums.packages += amount;
    else if (l.kind === "ExtraGuests") sums.extraGuests += amount;
    else if (l.kind === "Item") sums.items += amount;
    else if (l.kind === "Discount") sums.discount += amount;
  }
  return sums;
}

/** What all "Tambahan tamu" lines come to (to say when guests moved the price). */
export function extraGuestsTotal(r: Rincian | null): number {
  return (r?.lines ?? []).reduce((sum, l) => (l.kind === "ExtraGuests" ? sum + (l.amount ?? 0) : sum), 0);
}

/**
 * The row and field a server error points at: "lines[2].amount" or "lines.2.amount" is the third draft sent, which
 * the editor knows by key. Anything else is a message for the whole rincian.
 */
export function rowErrors(fields: Record<string, string>, sent: LineDraft[]): { byRow: Record<string, Record<string, string>>; general: string[] } {
  const byRow: Record<string, Record<string, string>> = {};
  const general: string[] = [];
  for (const [field, message] of Object.entries(fields)) {
    const m = /^lines(?:\[(\d+)\]|\.(\d+))\.(\w+)$/.exec(field);
    const draft = m ? sent[Number(m[1] ?? m[2])] : undefined;
    if (m && draft?.key) {
      byRow[draft.key] = { ...byRow[draft.key], [m[3]]: message };
    } else if (!field.startsWith("_")) {
      general.push(message);
    }
  }
  return { byRow, general };
}
