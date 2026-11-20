import { z } from "zod";

/**
 * One bill of a vendor or venue the client pays directly ("Tagihan vendor", Paket WO Tahap 4). The WO writes it down
 * so the couple sees what to pay the vendor and by when; it is never WO cash and never part of the contract.
 */
export const supplierBillTermSchema = z.object({
  projectSupplierId: z.string().min(1, "Pilih vendor atau venue"),
  label: z.string().trim().min(1, "Isi nama tagihan, mis. DP atau Pelunasan").max(100, "Paling banyak 100 karakter"),
  amount: z.coerce.number().int("Nominal dalam rupiah penuh").positive("Nominal harus lebih dari 0"),
  dueDate: z.string().min(1, "Isi tanggal jatuh tempo"),
});

export type SupplierBillTermFormValues = z.infer<typeof supplierBillTermSchema>;

/** "Tolak dengan alasan": the couple reads the reason, so it is required. */
export const supplierProofRejectSchema = z.object({
  reason: z.string().trim().min(1, "Tulis alasannya agar pasangan tahu apa yang perlu diperbaiki").max(500, "Paling banyak 500 karakter"),
});

export type SupplierProofRejectValues = z.infer<typeof supplierProofRejectSchema>;
