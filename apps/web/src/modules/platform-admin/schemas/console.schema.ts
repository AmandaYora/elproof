import { z } from "zod";

const day = (label: string, min: number, max: number) =>
  z.coerce
    .number({ invalid_type_error: `${label} harus berupa angka` })
    .int(`${label} harus bilangan bulat`)
    .min(min, `${label} minimal ${min} hari`)
    .max(max, `${label} maksimal ${max} hari`);

/** Mirrors backend domain.SubscriptionPolicy validation. */
export const subscriptionPolicySchema = z.object({
  trialDays: day("Masa coba", 1, 90),
  expiringSoonDays: day("Pengingat sebelum berakhir", 0, 30),
  graceDays: day("Masa tenggang", 0, 60),
  readOnlyDays: day("Masa baca-saja", 0, 180),
});

export type SubscriptionPolicyFormValues = z.infer<typeof subscriptionPolicySchema>;

export const ANNOUNCEMENT_KINDS = [
  { value: "info", label: "Informasi" },
  { value: "warning", label: "Peringatan" },
  { value: "maintenance", label: "Pemeliharaan" },
] as const;

export const ANNOUNCEMENT_AUDIENCES = [
  { value: "all", label: "Semua tenant" },
  { value: "status", label: "Tenant dengan status langganan tertentu" },
  { value: "plan", label: "Tenant dengan paket tertentu" },
] as const;

export const announcementSchema = z
  .object({
    title: z.string().trim().min(3, "Judul minimal 3 karakter").max(120, "Judul maksimal 120 karakter"),
    body: z.string().trim().min(5, "Isi pengumuman minimal 5 karakter").max(1000, "Isi maksimal 1000 karakter"),
    kind: z.enum(["info", "warning", "maintenance"], { errorMap: () => ({ message: "Pilih jenis pengumuman" }) }),
    audience: z.enum(["all", "plan", "status"], { errorMap: () => ({ message: "Pilih siapa yang dituju" }) }),
    audienceValue: z.string(),
    startsAt: z.string().min(1, "Isi waktu mulai tayang"),
    endsAt: z.string(),
  })
  .refine((v) => v.audience === "all" || v.audienceValue !== "", { path: ["audienceValue"], message: "Pilih tujuan pengumuman" })
  .refine((v) => v.endsAt === "" || v.endsAt > v.startsAt, { path: ["endsAt"], message: "Waktu selesai harus setelah waktu mulai" });

export type AnnouncementFormValues = z.infer<typeof announcementSchema>;

export const noteSchema = z.object({ body: z.string().trim().min(3, "Catatan minimal 3 karakter").max(1000, "Catatan maksimal 1000 karakter") });

export const dataRequestSchema = z.object({
  kind: z.enum(["export", "delete"], { errorMap: () => ({ message: "Pilih jenis permintaan" }) }),
  requestedBy: z.string().trim().min(3, "Isi siapa yang meminta, misalnya email Owner").max(150, "Maksimal 150 karakter"),
  note: z.string().trim().max(500, "Catatan maksimal 500 karakter"),
});

export const completeDataRequestSchema = z.object({
  status: z.enum(["done", "rejected"]),
  note: z.string().trim().min(5, "Isi bukti penyelesaian atau alasan penolakan (minimal 5 karakter)").max(500, "Maksimal 500 karakter"),
});

export const supportReasonSchema = z.object({
  reason: z.string().trim().min(10, "Jelaskan alasan melihat workspace ini (minimal 10 karakter)").max(255, "Alasan maksimal 255 karakter"),
});

/** `datetime-local` value ("2026-11-07T10:00") to the API's timestamp, in WIB. */
export function localToApiTime(local: string): string {
  return `${local}:00+07:00`;
}

/** The API's timestamp to a `datetime-local` value. */
export function apiTimeToLocal(iso: string | null): string {
  return iso ? iso.slice(0, 16) : "";
}
