import { create } from "zustand";
import { ifMatch } from "@/shared/lib/concurrency";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { uploadFile } from "@/shared/lib/direct-upload";
import { renameApiFieldErrors } from "@/shared/lib/api-error";
import { toSupplierPayment, useProjectStore, type RawSupplierPayment } from "@/modules/projects/stores/useProjectStore";
import type { SupplierBill, SupplierBillTerm, SupplierKind } from "@/modules/projects/types";
import type { SupplierBillTermFormValues } from "@/modules/projects/schemas/supplier-bill.schema";
import type { SubmitProofValues } from "@/modules/client-portal/schemas/submit-proof.schema";

// "Tagihan vendor" (Paket WO Tahap 4): the bills of a vendor or venue the client pays directly, the couple's proofs
// of paying them from the portal, and the WO checking those proofs. None of it is WO cash: the client pays the
// vendor, so Keuangan and the WO's own totals never move with it.

interface RawSupplierBillTerm {
  id: number;
  projectSupplierId: number;
  label: string;
  amount: number;
  dueDate: string;
  paid: number;
  remaining: number;
  pending: number;
  status: SupplierBillTerm["status"];
  updatedAt?: string;
}

interface RawSupplierBill {
  projectSupplierId: number;
  name: string;
  kind: SupplierKind;
  total: number;
  paid: number;
  remaining: number;
  pending: number;
  terms?: RawSupplierBillTerm[] | null;
  payments?: RawSupplierPayment[] | null;
}

function toTerm(raw: RawSupplierBillTerm): SupplierBillTerm {
  return {
    updatedAt: raw.updatedAt ?? "",
    id: String(raw.id),
    projectSupplierId: String(raw.projectSupplierId),
    label: raw.label,
    amount: raw.amount,
    dueDate: raw.dueDate,
    paid: raw.paid,
    remaining: raw.remaining,
    pending: raw.pending,
    status: raw.status,
  };
}

function toBill(raw: RawSupplierBill): SupplierBill {
  return {
    projectSupplierId: String(raw.projectSupplierId),
    name: raw.name,
    kind: raw.kind,
    total: raw.total,
    paid: raw.paid,
    remaining: raw.remaining,
    pending: raw.pending,
    terms: (raw.terms ?? []).map(toTerm),
    payments: (raw.payments ?? []).map(toSupplierPayment),
  };
}

type LoadStatus = "idle" | "loading" | "ready" | "error";

interface SupplierBillState {
  /** The project the list below belongs to; a screen of another project shows its skeleton until its own list lands. */
  projectId: string | null;
  bills: SupplierBill[];
  status: LoadStatus;
  fetchBills: (projectId: string) => Promise<void>;
  createTerm: (projectId: string, values: SupplierBillTermFormValues, idempotencyKey: string) => Promise<void>;
  updateTerm: (projectId: string, termId: string, values: Omit<SupplierBillTermFormValues, "projectSupplierId">, version: string) => Promise<void>;
  /** Deletes a bill and answers how to bring it back (for `deleteWithUndo`). */
  deleteTerm: (projectId: string, termId: string) => Promise<() => Promise<void>>;
  /** Marks a proof the couple sent as matching (it then counts as paid to the vendor). */
  verifySubmission: (projectId: string, paymentId: string) => Promise<void>;
  /** Refuses a proof the couple sent; the reason is shown to them. */
  rejectSubmission: (projectId: string, paymentId: string, reason: string) => Promise<void>;
  /** Portal: the couple sends proof of paying one bill (upload first, then the submission, safe to retry). */
  submitVendorProof: (projectId: string, billTermId: string, values: SubmitProofValues, idempotencyKey: string) => Promise<void>;
}

/** Refreshes what the WO screens around the bills show: the bills, the payments ledger and Vendor & venue. */
async function refreshStaff(projectId: string, fetchBills: (id: string) => Promise<void>): Promise<void> {
  const project = useProjectStore.getState();
  await Promise.all([fetchBills(projectId), project.fetchSupplierPayments(projectId), project.fetchSuppliers(projectId)]);
}

export const useSupplierBillStore = create<SupplierBillState>((set, get) => ({
  projectId: null,
  bills: [],
  status: "idle",

  fetchBills: async (projectId) => {
    const same = get().projectId === projectId;
    set({ projectId, status: same && get().status === "ready" ? "ready" : "loading", ...(same ? {} : { bills: [] }) });
    try {
      const res = await httpClient.get(API.projects.supplierBills(projectId));
      if (get().projectId !== projectId) return;
      set({ bills: ((res.data.data ?? []) as RawSupplierBill[]).map(toBill), status: "ready" });
    } catch (err) {
      if (get().projectId === projectId) set({ status: "error" });
      throw err;
    }
  },

  createTerm: async (projectId, values, idempotencyKey) => {
    await httpClient.post(
      API.projects.supplierBillTerms(projectId),
      { projectSupplierId: Number(values.projectSupplierId), label: values.label, amount: values.amount, dueDate: values.dueDate },
      { headers: { "Idempotency-Key": idempotencyKey } },
    );
    await refreshStaff(projectId, get().fetchBills);
  },

  updateTerm: async (projectId, termId, values, version) => {
    await httpClient.patch(
      API.projects.supplierBillTerm(projectId, termId),
      { label: values.label, amount: values.amount, dueDate: values.dueDate },
      { headers: ifMatch(version) },
    );
    await get().fetchBills(projectId);
  },

  deleteTerm: async (projectId, termId) => {
    const before = get().bills;
    set({ bills: before.map((b) => ({ ...b, terms: b.terms.filter((t) => t.id !== termId) })) });
    try {
      await httpClient.delete(API.projects.supplierBillTerm(projectId, termId));
    } catch (err) {
      set({ bills: before });
      throw err;
    }
    void refreshStaff(projectId, get().fetchBills).catch(() => undefined);
    return async () => {
      await httpClient.post(API.projects.supplierBillTermRestore(projectId, termId));
      await refreshStaff(projectId, get().fetchBills);
    };
  },

  verifySubmission: async (projectId, paymentId) => {
    await httpClient.post(API.projects.supplierPaymentVerify(projectId, paymentId));
    await refreshStaff(projectId, get().fetchBills);
  },

  rejectSubmission: async (projectId, paymentId, reason) => {
    await httpClient.post(API.projects.supplierPaymentReject(projectId, paymentId), { reason });
    await refreshStaff(projectId, get().fetchBills);
  },

  submitVendorProof: async (projectId, billTermId, values, idempotencyKey) => {
    if (!values.proofFile) throw new Error("proofFile missing");
    let uploadToken: string;
    try {
      uploadToken = await uploadFile(projectId, values.proofFile);
    } catch (err) {
      throw renameApiFieldErrors(err, () => "proofFile");
    }
    try {
      await httpClient.post(
        API.projects.supplierBillSubmissions(projectId),
        {
          billTermId: Number(billTermId),
          amount: values.amount,
          paymentDate: values.paymentDate,
          method: values.method,
          referenceNumber: values.referenceNumber || undefined,
          notes: values.notes || undefined,
          evidence: [{ uploadToken, type: "Transfer Proof" }],
        },
        { headers: { "Idempotency-Key": idempotencyKey } },
      );
    } catch (err) {
      throw renameApiFieldErrors(err, (key) => (/^evidence(\[0\])?(\.\w+)?$/.test(key) ? "proofFile" : undefined));
    }
    await get().fetchBills(projectId).catch(() => undefined);
  },
}));
