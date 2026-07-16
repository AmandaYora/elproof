import { z } from "zod";

export const ENGAGEMENT_STATUS_OPTIONS = [
  "Planned",
  "Negotiation",
  "Booked",
  "DP Paid",
  "In Progress",
  "Fully Paid",
  "Ready",
  "Completed",
  "Cancelled",
] as const;

export const projectVendorSchema = z.object({
  vendorId: z.string().min(1, "Vendor wajib dipilih"),
  scope: z.string().min(3, "Scope pekerjaan wajib diisi"),
  contractValue: z.coerce.number().min(0, "Nilai kerja sama tidak valid"),
  engagementStatus: z.enum(ENGAGEMENT_STATUS_OPTIONS),
  bookingDate: z.string().optional().default(""),
  dpAmount: z.coerce.number().min(0, "Jumlah DP tidak valid"),
  paidAmount: z.coerce.number().min(0, "Jumlah pembayaran tidak valid"),
  dueDate: z.string().optional().default(""),
  picStaffId: z.string().min(1, "PIC internal WO wajib dipilih"),
  notes: z.string().optional().default(""),
});

export type ProjectVendorFormValues = z.infer<typeof projectVendorSchema>;
