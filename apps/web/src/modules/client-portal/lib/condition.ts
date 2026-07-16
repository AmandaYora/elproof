import type { ProjectCondition } from "@/modules/projects/types";

export const CLIENT_CONDITION_COPY: Record<ProjectCondition, { label: string; tone: "success" | "warning" | "danger" }> = {
  "On Track": { label: "Berjalan sesuai rencana", tone: "success" },
  Attention: { label: "Ada beberapa hal yang sedang kami tangani", tone: "warning" },
  "At Risk": { label: "Membutuhkan perhatian lebih dari tim kami", tone: "danger" },
};
