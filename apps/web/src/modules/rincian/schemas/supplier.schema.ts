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

export const SUPPLIER_PAYER_OPTIONS = ["WO", "Client"] as const;

// A vendor or venue of the project ("Vendor & venue"). vendorId is read only
// when a vendor is added (a venue comes in through an event). agreedCost
// ("Biaya rekanan") is shown and sent only to finance.margin.view; empty = not
// set yet (the margin is then incomplete).
export const supplierSchema = z.object({
  vendorId: z.string().optional().default(""),
  paidBy: z.enum(SUPPLIER_PAYER_OPTIONS),
  showName: z.boolean(),
  agreedCost: z.union([z.literal(""), z.coerce.number().min(0, "Biaya rekanan tidak boleh negatif")]).optional().default(""),
  engagementStatus: z.enum(ENGAGEMENT_STATUS_OPTIONS),
  bookingDate: z.string().optional().default(""),
  dueDate: z.string().optional().default(""),
  picStaffId: z.string().min(1, "Pilih PIC internal WO"),
  scope: z.string().optional().default(""),
  notes: z.string().optional().default(""),
});

export type SupplierFormValues = z.infer<typeof supplierSchema>;
