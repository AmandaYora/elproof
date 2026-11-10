import type { BadgeTone } from "@/shared/components/ui/Badge";
import type { SubscriptionTransactionStatus, TenantSubscriptionStatus } from "@/modules/platform-admin/data/types";

export const TENANT_STATUS_TONE: Record<TenantSubscriptionStatus, BadgeTone> = {
  trialing: "info",
  active: "success",
  expiring_soon: "warning",
  past_due: "warning",
  read_only: "danger",
  suspended: "danger",
  pending_payment: "neutral",
  expired: "danger",
};

export const TENANT_STATUS_LABEL: Record<TenantSubscriptionStatus, string> = {
  trialing: "Masa Coba",
  active: "Aktif",
  expiring_soon: "Segera Berakhir",
  past_due: "Masa Tenggang",
  read_only: "Baca-saja",
  suspended: "Ditangguhkan",
  pending_payment: "Belum Berlangganan",
  expired: "Berakhir",
};

/** Statuses shown as filters on the tenant list, in lifecycle order. */
export const TENANT_STATUS_FILTERS: TenantSubscriptionStatus[] = [
  "trialing",
  "active",
  "expiring_soon",
  "past_due",
  "read_only",
  "suspended",
];

/** Statuses that need the platform team's attention (dashboard). */
export const TENANT_ATTENTION_STATUSES: TenantSubscriptionStatus[] = ["expiring_soon", "past_due", "read_only", "suspended"];

export const TRANSACTION_STATUS_TONE: Record<SubscriptionTransactionStatus, BadgeTone> = {
  unpaid: "warning",
  pending: "warning",
  paid: "success",
  expired: "danger",
  granted: "brand",
  cancelled: "neutral",
};

export const TRANSACTION_STATUS_LABEL: Record<SubscriptionTransactionStatus, string> = {
  unpaid: "Menunggu Pembayaran",
  pending: "Menunggu Konfirmasi",
  paid: "Berhasil",
  expired: "Kedaluwarsa",
  granted: "Diaktifkan Admin",
  cancelled: "Dibatalkan",
};

export const TRANSACTION_TYPE_LABEL = {
  new: "Baru",
  renewal: "Perpanjangan",
} as const;
