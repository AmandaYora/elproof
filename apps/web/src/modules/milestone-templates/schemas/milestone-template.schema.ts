import { z } from "zod";

// Internal form state only (MilestoneTemplateFormModal's own useState) --
// never sent to onSubmit/store/API directly. daysOffset is always
// non-negative; daysDirection ("before"/"after") carries the sign instead
// of asking the user to type a negative number under a field labeled "H-"
// (PLAN.md "Timeline Default: dukung H+"). See toDaysBeforeEvent/
// fromDaysBeforeEvent below for the conversion to/from the API's signed
// daysBeforeEvent.
export const milestoneTemplateSchema = z.object({
  name: z.string().min(3, "Nama timeline minimal 3 karakter"),
  daysOffset: z.number().int("Harus berupa angka bulat").min(0, "Tidak boleh negatif"),
  daysDirection: z.enum(["before", "after"]),
});

export type MilestoneTemplateFormValues = z.infer<typeof milestoneTemplateSchema>;

// What actually flows to onSubmit -> MilestoneTemplateListPage.handleSubmit
// -> useMilestoneTemplateStore's createTemplate/updateTemplate -> the API
// body -- unchanged shape from before this feature existed. Kept as its own
// named type (not reusing MilestoneTemplateFormValues) so the store's own
// type annotation can never silently drift into sending the wrong shape --
// see PLAN.md's "KOREKSI" note for exactly why this split matters.
export interface MilestoneTemplateSubmitValues {
  name: string;
  daysBeforeEvent: number;
}

export function toDaysBeforeEvent(daysOffset: number, direction: "before" | "after"): number {
  return direction === "after" ? -daysOffset : daysOffset;
}

export function fromDaysBeforeEvent(daysBeforeEvent: number): { daysOffset: number; daysDirection: "before" | "after" } {
  return { daysOffset: Math.abs(daysBeforeEvent), daysDirection: daysBeforeEvent < 0 ? "after" : "before" };
}
