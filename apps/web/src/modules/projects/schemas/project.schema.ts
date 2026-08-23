import { z } from "zod";

export const PROJECT_STATUS_OPTIONS = ["Draft", "Preparation", "Ready", "Completed", "Cancelled"] as const;

export const projectSchema = z.object({
  name: z.string().min(3, "Nama project minimal 3 karakter"),
  brideName: z.string().min(2, "Nama mempelai wanita wajib diisi"),
  groomName: z.string().min(2, "Nama mempelai pria wajib diisi"),
  eventDate: z.string().min(1, "Tanggal acara wajib diisi"),
  venue: z.string().min(2, "Venue wajib diisi"),
  // Field name stays prepStartDate (display-text-only rename to "Tanggal
  // Booking" — see DOMAIN_GLOSSARY.md and PLAN.md).
  prepStartDate: z.string().min(1, "Tanggal booking wajib diisi"),
  packageName: z.string().min(2, "Paket/layanan wajib diisi"),
  contractValue: z.coerce.number().min(0, "Nilai kontrak tidak valid"),
  status: z.enum(PROJECT_STATUS_OPTIONS),
  // Optional now (PLAN.md revisi-timeline-vendor-role-sales) — a project
  // created by Sales may have no Wedding Planner assigned yet, until
  // handover.
  picStaffId: z.string().optional().default(""),
  // PIC Sales — "" means "belum ditugaskan" (0 on the wire), same sentinel
  // convention as picStaffId now uses.
  picSalesStaffId: z.string().optional().default(""),
  description: z.string().optional().default(""),
});

export type ProjectFormValues = z.infer<typeof projectSchema>;
