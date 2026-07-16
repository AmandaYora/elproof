import { z } from "zod";

export const activateSubscriptionSchema = z.object({
  planId: z.string().min(1, "Pilih paket langganan"),
});

export type ActivateSubscriptionFormValues = z.infer<typeof activateSubscriptionSchema>;
