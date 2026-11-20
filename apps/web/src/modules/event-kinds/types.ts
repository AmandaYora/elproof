// The `offers` module's event kinds (docs/API_CONTRACT.md, ADR-0049): six built-in kinds every WO starts with, plus its own.

export interface EventKind {
  id: string;
  name: string;
  /** One of the six kinds every workspace starts with: it can be renamed or deactivated like any other. */
  isDefault: boolean;
  isActive: boolean;
  /** How many of the workspace's packages cover this kind: a kind in use cannot be deleted. */
  packageCount: number;
  /** How many quotations have an event of this kind: like a package, it keeps the kind from being deleted. */
  quotationCount: number;
  /** The version a rename sends back as If-Match (ADR-0036). */
  updatedAt: string;
}
