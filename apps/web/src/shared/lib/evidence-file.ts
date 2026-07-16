export type EvidenceFileKind = "pdf" | "image" | "file";

const IMAGE_EXTENSIONS = new Set(["jpg", "jpeg", "png", "gif", "webp", "bmp", "svg"]);

// The evidence type label (e.g. "Transfer Proof") doesn't reliably predict
// the actual file format — a proof can be a PDF or a photo — so the viewer
// kind is derived from the stored file's own extension instead.
export function getEvidenceFileKind(fileName: string): EvidenceFileKind {
  const ext = fileName.split(".").pop()?.toLowerCase() ?? "";
  if (ext === "pdf") return "pdf";
  if (IMAGE_EXTENSIONS.has(ext)) return "image";
  return "file";
}
