export interface SubscriptionPlan {
  id: string;
  name: string;
  durationMonths: number;
  price: number;
  features: string[];
  isActive: boolean;
}
