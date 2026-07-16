// Frontend half of ADR-0010's compression strategy — mirrors the backend's
// internal/shared/compress rules exactly (same constants, same scope): JPEG
// gets lossy re-encoding at a fixed quality, PNG gets a lossless re-encode,
// anything else (PDF, other formats) passes through untouched. No new npm
// dependency — uses the browser's native Canvas API only.
//
// The backend re-compresses independently and is the authoritative pass
// (defense in depth) — this module only reduces what actually goes over the
// wire, since the whole point of compressing here is a smaller upload.

const MAX_DIMENSION = 2000;
const JPEG_QUALITY = 0.82;

const COMPRESSIBLE_TYPES = new Set(["image/jpeg", "image/jpg", "image/png"]);

export interface CompressedFilePayload {
  fileName: string;
  mimeType: string;
  /** Raw base64 — no `data:...;base64,` prefix (ADR-0010). */
  base64Data: string;
}

/**
 * Compresses an image file (JPEG/PNG) via the Canvas API, downscaling if its
 * longest side exceeds MAX_DIMENSION, then returns it as base64 ready for the
 * JSON evidence-upload body. Non-image or unhandled types pass through
 * unchanged — this is a deliberate scope boundary (see ADR-0010), not a bug.
 */
export async function compressFileForUpload(file: File): Promise<CompressedFilePayload> {
  if (!COMPRESSIBLE_TYPES.has(file.type)) {
    return { fileName: file.name, mimeType: file.type, base64Data: await blobToBase64(file) };
  }

  try {
    const bitmap = await createImageBitmap(file);
    const { width, height } = scaledDimensions(bitmap.width, bitmap.height);

    const canvas = document.createElement("canvas");
    canvas.width = width;
    canvas.height = height;
    const ctx = canvas.getContext("2d");
    if (!ctx) throw new Error("Canvas 2D context tidak tersedia");
    ctx.drawImage(bitmap, 0, 0, width, height);
    bitmap.close();

    const quality = file.type === "image/png" ? undefined : JPEG_QUALITY;
    const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, file.type, quality));
    if (!blob) throw new Error("Gagal mengompres gambar");

    return { fileName: file.name, mimeType: file.type, base64Data: await blobToBase64(blob) };
  } catch {
    // Compression is a size optimization, not a correctness requirement — if
    // it fails for any reason, fall back to the original file rather than
    // blocking the upload. The backend's own re-compression pass still runs.
    return { fileName: file.name, mimeType: file.type, base64Data: await blobToBase64(file) };
  }
}

function scaledDimensions(width: number, height: number): { width: number; height: number } {
  if (width <= MAX_DIMENSION && height <= MAX_DIMENSION) {
    return { width, height };
  }
  if (width >= height) {
    return { width: MAX_DIMENSION, height: Math.round((height * MAX_DIMENSION) / width) };
  }
  return { width: Math.round((width * MAX_DIMENSION) / height), height: MAX_DIMENSION };
}

function blobToBase64(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result as string;
      const commaIndex = result.indexOf(",");
      resolve(commaIndex >= 0 ? result.slice(commaIndex + 1) : result);
    };
    reader.onerror = () => reject(reader.error ?? new Error("Gagal membaca file"));
    reader.readAsDataURL(blob);
  });
}
