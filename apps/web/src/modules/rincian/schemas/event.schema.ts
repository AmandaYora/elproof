import { z } from "zod";

export const EVENT_LAYOUT_OPTIONS = ["Standing", "Seated"] as const;

const time = z
  .string()
  .regex(/^$|^([01]\d|2[0-3]):[0-5]\d$/, "Gunakan jam JJ:MM")
  .optional()
  .default("");

export const eventSessionSchema = z.object({
  label: z.string().optional().default(""),
  startTime: time,
  endTime: time,
  guests: z.coerce.number().int("Isi jumlah tamu tanpa koma").min(0, "Jumlah tamu tidak boleh negatif"),
});

// One acara with its sessions, as the drawer edits it. The capacity rule
// (tamu per sesi ≤ kapasitas venue untuk tata letak itu) is the server's: the
// form only shows what is left, the save is what refuses.
export const eventSchema = z.object({
  eventKindId: z.string().min(1, "Pilih jenis acara"),
  eventDate: z.string().min(1, "Pilih tanggal acara"),
  startTime: time,
  endTime: time,
  venueId: z.string().optional().default(""),
  layout: z.enum(EVENT_LAYOUT_OPTIONS),
  sessions: z.array(eventSessionSchema).min(1, "Isi minimal satu sesi"),
});

export type EventFormValues = z.infer<typeof eventSchema>;
export type EventSessionFormValues = z.infer<typeof eventSessionSchema>;
