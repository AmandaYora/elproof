import type { LucideIcon } from "lucide-react";

export interface NavItem {
  to: string;
  label: string;
  icon: LucideIcon;
}

/**
 * A block of the menu. With `collapsible` the items sit under one expandable entry ("Direktori", "Pengaturan");
 * with a `label` they sit under a small heading ("Pelanggan"); with neither they are plain entries.
 */
export interface NavSection {
  id: string;
  label?: string;
  collapsible?: { label: string; icon: LucideIcon };
  items: NavItem[];
}
