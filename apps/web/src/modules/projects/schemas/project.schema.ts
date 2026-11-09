import { z } from "zod";

export const PROJECT_STATUS_OPTIONS = ["Draft", "Preparation", "Ready", "Completed", "Cancelled"] as const;

export const projectSchema = z.object({
  name: z.string().min(3, "Isi nama project, minimal 3 karakter"),
  brideName: z.string().min(2, "Isi nama mempelai wanita"),
  groomName: z.string().min(2, "Isi nama mempelai pria"),
  eventDate: z.string().min(1, "Pilih tanggal acara"),
  venue: z.string().min(2, "Isi lokasi atau venue"),
  // Field name stays prepStartDate (display-text-only rename to "Tanggal
  // Booking" — see DOMAIN_GLOSSARY.md and PLAN.md).
  prepStartDate: z.string().min(1, "Pilih tanggal booking"),
  packageName: z.string().min(2, "Isi paket atau layanan"),
  contractValue: z.coerce.number().min(0, "Nilai kontrak tidak boleh negatif"),
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
