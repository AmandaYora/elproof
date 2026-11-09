import { z } from "zod";

/**
 * Field names match the API's (so a 422 lands on the right field). There are no
 * client-side rules: a blank API or private key means "keep what is stored",
 * and what a provider needs is decided by the server.
 */
export const gatewayConfigSchema = z.object({
  activeProvider: z.string(),
  isSandbox: z.boolean(),
  tripayMerchantCode: z.string(),
  tripayApiKey: z.string(),
  tripayPrivateKey: z.string(),
});

export type GatewayConfigFormValues = z.infer<typeof gatewayConfigSchema>;
