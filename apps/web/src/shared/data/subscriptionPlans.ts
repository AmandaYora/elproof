export interface SubscriptionPlan {
  id: string;
  /** The one App that sells this plan (ADR-0040). ELPROOF_APP_ID = ElProof's own catalog for WO tenants. */
  appId: string;
  name: string;
  durationMonths: number;
  price: number;
  features: string[];
  isActive: boolean;
}

/**
 * ElProof's own App in the payment registry (paymentcontracts.InternalAppBilling). Its plans are the only ones a WO
 * tenant can see or be subscribed to; the backend enforces that, this constant only lets the Console label it.
 */
export const ELPROOF_APP_ID = "platform-billing";

/** The name a person reads for an App's catalog: ElProof's own App is the WO tenants' catalog. */
export function appLabel(app: { appId: string; name: string } | undefined, appId: string): string {
  if (appId === ELPROOF_APP_ID) return "ElProof (tenant WO)";
  return app?.name ?? appId;
}
