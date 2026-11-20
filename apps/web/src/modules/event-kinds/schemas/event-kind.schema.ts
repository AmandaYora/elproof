import { z } from "zod";

export const eventKindSchema = z.object({
  name: z.string().trim().min(1, "Isi nama jenis acara").max(60, "Nama jenis acara maksimal 60 karakter"),
});

export type EventKindFormValues = z.infer<typeof eventKindSchema>;
