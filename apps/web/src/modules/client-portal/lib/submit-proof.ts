import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { uploadFile } from "@/shared/lib/direct-upload";
import { renameApiFieldErrors } from "@/shared/lib/api-error";
import type { SubmitProofValues } from "@/modules/client-portal/schemas/submit-proof.schema";

/**
 * Sends the file through the signed upload (POST …/uploads → PUT), then the payment with its upload token in one
 * request that is safe to retry (Idempotency-Key). Whatever the server says about the file lands on "proofFile".
 */
export async function submitTransferProof(projectId: string, values: SubmitProofValues, idempotencyKey: string): Promise<void> {
  if (!values.proofFile) throw new Error("proofFile missing");
  let uploadToken: string;
  try {
    uploadToken = await uploadFile(projectId, values.proofFile);
  } catch (err) {
    throw renameApiFieldErrors(err, () => "proofFile");
  }
  try {
    await httpClient.post(
      API.projects.clientPaymentSubmissions(projectId),
      {
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
}
