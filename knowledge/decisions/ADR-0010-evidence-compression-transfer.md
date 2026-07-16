# ADR-0010: Evidence upload — compression strategy and base64 transfer

## Status
Accepted

## Context
Following ADR-0006 (IDCloudHost object storage), the user explicitly requested: (a) compression on
**both** frontend and backend, "best-practice, ideal, and non-destructive to the file", and (b) a
base64 transfer mechanism for the HTTP request from frontend to backend (not multipart/form-data,
which was the original placeholder in `docs/API_CONTRACT.md`).

## Decision

### Scope of compression: images only, format-preserving
"Non-destructive" is interpreted literally: only apply compression where it is both safe and
actually effective.

- **JPEG (`image/jpeg`)**: re-encoded at a fixed quality setting (0.82 / quality 82). This is lossy
  but visually near-identical at this quality for photographic evidence — standard practice for web
  uploads, not a corruption risk (the file remains a valid, fully-openable JPEG).
- **PNG (`image/png`)**: re-encoded at maximum deflate compression level. This is **lossless** —
  pixel data is byte-for-byte unchanged, only the container's internal compression improves. Format
  is never converted (PNG stays PNG) because PNG's alpha channel (transparency) would be silently
  destroyed by converting to JPEG — that would be actual data loss, not "ideal" compression.
- **Any other file** (PDF, DOCX, other image formats, etc.): **stored byte-identical, no
  compression attempted.** PDFs and Office documents are already internally compressed containers
  (Flate streams / zip); wrapping them in another compression pass typically saves 1–5% while adding
  real complexity (tracking whether a file is wrapped, setting `Content-Encoding` correctly on
  download) and correctness risk. This is a deliberate scope boundary, not an oversight.
- **Oversized images are downscaled** before re-encoding: longest side capped at 2000px (a WO's
  evidence photos never need to exceed typical screen/print resolution). Aspect ratio is always
  preserved — no cropping, no stretching.

### Where compression happens: both sides, backend is authoritative
- **Frontend** compresses first (Canvas API — `HTMLCanvasElement.toBlob`, no new npm dependency):
  draws the image at the capped dimension, re-encodes at the quality above, *before* base64-encoding
  it for transfer. This keeps upload payloads small on slow connections.
  Implemented in `shared/lib/image-compression.ts`.
- **Backend re-compresses independently** (Go standard library `image`/`image/jpeg`/`image/png`,
  plus `golang.org/x/image/draw` for the resize step — official Go extended package, not a
  third-party imaging library) using the exact same rules. This is defense in depth: the API is
  never trusted to only be called through the frontend, so the backend must enforce the same
  size/quality bound regardless of what a direct API caller sends. Implemented in
  `internal/shared/compress`.
- Whichever side's output is smaller is not the point — **both always run**; the backend's
  re-compression is what actually gets uploaded to object storage, since it is the last, trusted
  step before persistence.

### Transfer encoding: base64 over JSON, not multipart
`POST /api/v1/projects/{id}/evidence` (Fase 4) accepts a JSON body:
```json
{ "name": "...", "type": "photo", "fileName": "kwitansi-dp.jpg", "mimeType": "image/jpeg",
  "base64Data": "<base64, no data: URI prefix>", "relatedKind": "payment", "relatedId": "..." }
```
This replaces the multipart/form-data placeholder in the original `docs/API_CONTRACT.md` draft.
Rationale: the whole rest of the API is JSON-in/JSON-out with the standard success/error envelope;
base64-in-JSON keeps evidence upload consistent with that, at the cost of ~33% payload inflation
from base64 itself — an acceptable trade given compression already ran before encoding.

- A **decoded-size cap** (15 MB) is enforced server-side before any processing — an oversized
  payload is rejected with a validation error, never partially processed.
- The frontend strips the `data:<mime>;base64,` prefix before sending (`FileReader.readAsDataURL`
  produces that prefix); the backend expects raw base64 only, no URI scheme.

## Consequences
- Two independent implementations of the same compression rules (frontend Canvas, backend Go) must
  be kept in sync if the quality/dimension constants ever change — documented here as the single
  source of truth for both (2000px longest side, JPEG quality 82, PNG max compression) so a future
  change updates both intentionally, not just one.
- Base64-in-JSON means the whole file sits in memory (both as base64 string and decoded bytes)
  during a single request — acceptable at the 15 MB decoded cap (≈20 MB base64), not appropriate if
  evidence files ever need to grow far beyond that (out of scope for this project's actual usage:
  photos and scanned documents, not video).
- Object storage always holds the *compressed* bytes (post backend re-compression), never the raw
  upload — `evidence.storage_path` points at what was actually compressed and stored, and download
  always serves exactly those bytes back (no on-the-fly re-processing on read).
