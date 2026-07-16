// Real backend types for the `vendors` module — see docs/API_CONTRACT.md.

export interface Vendor {
  id: string;
  name: string;
  categoryId: string;
  picName: string;
  phone: string;
  email: string;
  address: string;
  notes: string;
  isActive: boolean;
  createdAt: string;
}

// One row of a vendor's project engagement history — see
// `GET /vendors/{id}/project-history`.
export interface VendorProjectHistoryItem {
  projectId: string;
  projectName: string;
  eventDate: string;
  venue: string;
  engagementStatus: string;
}
