import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { MilestoneTemplate } from "@/modules/milestone-templates/types";
import type { MilestoneTemplateFormValues } from "@/modules/milestone-templates/schemas/milestone-template.schema";

interface RawMilestoneTemplate {
  id: number;
  name: string;
  daysBeforeEvent: number;
  sortOrder: number;
}

function toMilestoneTemplate(raw: RawMilestoneTemplate): MilestoneTemplate {
  return { ...raw, id: String(raw.id) };
}

interface MilestoneTemplateState {
  templates: MilestoneTemplate[];
  fetchTemplates: () => Promise<void>;
  createTemplate: (values: MilestoneTemplateFormValues) => Promise<void>;
  updateTemplate: (id: string, values: MilestoneTemplateFormValues) => Promise<void>;
  deleteTemplate: (id: string) => Promise<void>;
  reorderTemplates: (orderedIds: string[]) => Promise<void>;
}

// Backed by the real `projects` module (Timeline Default Template, PLAN.md) --
// tenant-scoped, Owner-only. Un-paginated: expected to stay a short list, the
// same as a single project's own Timeline tab.
export const useMilestoneTemplateStore = create<MilestoneTemplateState>((set, get) => ({
  templates: [],

  fetchTemplates: async () => {
    const res = await httpClient.get(API.milestoneTemplates.base);
    set({ templates: (res.data.data as RawMilestoneTemplate[]).map(toMilestoneTemplate) });
  },

  createTemplate: async (values) => {
    await httpClient.post(API.milestoneTemplates.base, values);
    await get().fetchTemplates();
  },

  updateTemplate: async (id, values) => {
    await httpClient.patch(API.milestoneTemplates.item(id), values);
    await get().fetchTemplates();
  },

  deleteTemplate: async (id) => {
    await httpClient.delete(API.milestoneTemplates.item(id));
    await get().fetchTemplates();
  },

  reorderTemplates: async (orderedIds) => {
    await httpClient.patch(API.milestoneTemplates.base, { orderedIds: orderedIds.map(Number) });
    await get().fetchTemplates();
  },
}));
