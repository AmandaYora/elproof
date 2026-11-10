import { z } from "zod";

/** One plan of ElProof's public catalog, as GET /api/v1/public/plans returns it (active plans only). */
export const publicPlanSchema = z.object({
  id: z.number(),
  name: z.string(),
  durationMonths: z.number().int().positive(),
  price: z.number().int().nonnegative(),
  features: z.array(z.string()),
});

export const publicPlanListSchema = z.array(publicPlanSchema);

export type PublicPlan = z.infer<typeof publicPlanSchema>;
