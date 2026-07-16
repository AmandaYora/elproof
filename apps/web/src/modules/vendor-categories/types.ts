// Real backend types for the `vendors` module's category sub-resource — see
// docs/API_CONTRACT.md.

export interface VendorCategory {
  id: string;
  name: string;
  description: string;
  isActive: boolean;
  createdAt: string;
}
