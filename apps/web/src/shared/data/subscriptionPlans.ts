export interface SubscriptionPlan {
  id: string;
  name: string;
  durationMonths: number;
  price: number;
  features: string[];
  isActive: boolean;
  // appIds is populated by the backend only for a platform_admin caller —
  // [] for every other principal (D12). A non-empty value means this plan
  // is private to those App IDs (D3); empty means it's part of ElProof's
  // own public catalog.
  appIds: string[];
}
