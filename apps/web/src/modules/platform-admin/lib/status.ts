import type { BadgeTone } from "@/shared/components/ui/Badge";
import type { SubscriptionTransactionStatus, TenantSubscriptionStatus } from "@/modules/platform-admin/data/types";

export const TENANT_STATUS_TONE: Record<TenantSubscriptionStatus, BadgeTone> = {
  active: "success",
  expiring_soon: "warning",
  expired: "danger",
  pending_payment: "neutral",
};

export const TENANT_STATUS_LABEL: Record<TenantSubscriptionStatus, string> = {
  active: "Aktif",
  expiring_soon: "Segera Berakhir",
  expired: "Berakhir",
  pending_payment: "Menunggu Pembayaran",
};

export const TRANSACTION_STATUS_TONE: Record<SubscriptionTransactionStatus, BadgeTone> = {
  unpaid: "warning",
  pending: "warning",
  paid: "success",
  expired: "danger",
  granted: "navy",
};

export const TRANSACTION_STATUS_LABEL: Record<SubscriptionTransactionStatus, string> = {
  unpaid: "Menunggu Pembayaran",
  pending: "Menunggu Konfirmasi",
  paid: "Berhasil",
  expired: "Kedaluwarsa",
  granted: "Diaktifkan Admin",
};

export const TRANSACTION_TYPE_LABEL = {
  new: "Baru",
  renewal: "Perpanjangan",
} as const;
