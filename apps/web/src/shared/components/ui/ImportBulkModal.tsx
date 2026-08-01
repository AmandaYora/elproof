import type { DragEvent, KeyboardEvent } from "react";
import { useEffect, useRef, useState } from "react";
import { Download, UploadCloud, FileSpreadsheet, X } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { cn } from "@/shared/lib/cn";
import { getApiErrorMessage } from "@/shared/lib/api-error";

export interface BulkImportResult {
  insertedCount: number;
  updatedCount: number;
  errors: { row: number; message: string }[];
}

interface ImportBulkModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  description: string;
  templateFileName: string;
  // Used only to compose the result summary sentence (e.g. "3 vendor baru
  // ditambahkan, 1 vendor diperbarui (nama+kota+kategori sudah ada
  // sebelumnya).") — kept as plain strings rather than a render-prop so this
  // component stays domain-agnostic while every caller keeps its own exact
  // wording.
  entityLabel: string;
  matchCriteriaLabel: string;
  onDownloadTemplate: () => Promise<Blob>;
  onImport: (file: File) => Promise<BulkImportResult>;
}

type Stage = "idle" | "selected" | "importing" | "result";

const ACCEPTED_EXTENSION = ".xlsx";

// Consolidates the "Download Template" + "Import Excel" button pair (used
// identically by Vendor and Venue list pages) into one modal: download the
// template, then drag-and-drop or browse for the filled-in file, confirm,
// and see the result — all in one place, instead of two separate header
// buttons and a results card that used to appear elsewhere on the page.
export function ImportBulkModal({
  open,
  onClose,
  title,
  description,
  templateFileName,
  entityLabel,
  matchCriteriaLabel,
  onDownloadTemplate,
  onImport,
}: ImportBulkModalProps) {
  const [stage, setStage] = useState<Stage>("idle");
  const [file, setFile] = useState<File | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [importError, setImportError] = useState<string | null>(null);
  const [result, setResult] = useState<BulkImportResult | null>(null);
  const [downloadingTemplate, setDownloadingTemplate] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  // A drop anywhere on the page that isn't handled (e.g. just outside the
  // dropzone but still inside the modal, or on the backdrop) defaults to the
  // browser navigating to/opening the dropped file, abandoning the whole
  // app — first drag-and-drop surface in this codebase, so nothing already
  // guards against this. Swallow it globally for as long as this modal is open.
  useEffect(() => {
    if (!open) return;
    const preventDefault = (e: Event) => e.preventDefault();
    window.addEventListener("dragover", preventDefault);
    window.addEventListener("drop", preventDefault);
    return () => {
      window.removeEventListener("dragover", preventDefault);
      window.removeEventListener("drop", preventDefault);
    };
  }, [open]);

  function reset() {
    setStage("idle");
    setFile(null);
    setFileError(null);
    setImportError(null);
    setResult(null);
    setIsDragging(false);
  }

  function handleClose() {
    // The footer's own cancel button is already disabled while importing —
    // Escape and the header's X button must respect that too, otherwise the
    // modal unmounts mid-flight and the eventual result/error from `onImport`
    // resolves into a component that's no longer there to show it.
    if (stage === "importing") return;
    reset();
    onClose();
  }

  function handleChangeFile() {
    setFile(null);
    setFileError(null);
    setImportError(null);
    setStage("idle");
    // Browsers don't fire `onChange` again for a file with the same name/path
    // as what's already the input's value — clear it so re-picking the exact
    // same file (e.g. after fixing errors in Excel and re-uploading) still
    // registers.
    if (inputRef.current) inputRef.current.value = "";
  }

  function pickFile(candidate: File | undefined | null) {
    if (!candidate) return;
    if (!candidate.name.toLowerCase().endsWith(ACCEPTED_EXTENSION)) {
      setFileError("Berkas harus berformat .xlsx");
      return;
    }
    setFileError(null);
    setFile(candidate);
    setStage("selected");
    if (inputRef.current) inputRef.current.value = "";
  }

  async function handleDownloadTemplate() {
    setDownloadingTemplate(true);
    setFileError(null);
    try {
      const blob = await onDownloadTemplate();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = templateFileName;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      setFileError(getApiErrorMessage(err, "Gagal mengunduh template"));
    } finally {
      setDownloadingTemplate(false);
    }
  }

  async function handleStartImport() {
    if (!file) return;
    setStage("importing");
    setImportError(null);
    try {
      const res = await onImport(file);
      setResult(res);
      setStage("result");
    } catch (err) {
      setImportError(getApiErrorMessage(err, "Gagal mengimpor berkas"));
      setStage("selected");
    }
  }

  function handleDrop(e: DragEvent<HTMLDivElement>) {
    e.preventDefault();
    setIsDragging(false);
    pickFile(e.dataTransfer.files?.[0]);
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={title}
      description={description}
      footer={
        stage === "result" ? (
          <>
            <Button variant="secondary" onClick={reset}>Import Berkas Lain</Button>
            <Button onClick={handleClose}>Selesai</Button>
          </>
        ) : stage === "selected" ? (
          <>
            <Button variant="secondary" onClick={handleChangeFile}>Ganti Berkas</Button>
            <Button onClick={() => void handleStartImport()}>Mulai Import</Button>
          </>
        ) : stage === "importing" ? (
          <Button variant="secondary" disabled>Mengimpor...</Button>
        ) : (
          <Button variant="secondary" onClick={handleClose}>Batal</Button>
        )
      }
    >
      <div className="flex flex-col gap-4">
        <button
          type="button"
          onClick={() => void handleDownloadTemplate()}
          disabled={downloadingTemplate}
          className="flex items-center gap-1.5 self-start text-[13px] font-semibold text-navy-900 hover:underline disabled:opacity-60"
        >
          <Download className="h-3.5 w-3.5" />
          {downloadingTemplate ? "Mengunduh..." : "Download Template"}
        </button>

        {stage === "idle" && (
          <div
            role="button"
            tabIndex={0}
            onDragOver={(e) => {
              e.preventDefault();
              setIsDragging(true);
            }}
            onDragLeave={() => setIsDragging(false)}
            onDrop={handleDrop}
            onClick={() => inputRef.current?.click()}
            onKeyDown={(e: KeyboardEvent<HTMLDivElement>) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                inputRef.current?.click();
              }
            }}
            className={cn(
              "flex cursor-pointer flex-col items-center gap-2 rounded-lg border-2 border-dashed px-6 py-10 text-center transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy-900",
              isDragging ? "border-navy-900 bg-navy-900/5" : "border-border hover:border-navy-900/40"
            )}
          >
            <UploadCloud className="h-8 w-8 text-text-secondary" />
            <p className="text-[13.5px] font-medium text-text-primary">
              Tarik &amp; lepas berkas Excel di sini, atau klik untuk memilih
            </p>
            <p className="text-[12px] text-text-secondary">Format .xlsx</p>
            <input
              ref={inputRef}
              type="file"
              accept=".xlsx"
              className="hidden"
              onChange={(e) => pickFile(e.target.files?.[0])}
            />
          </div>
        )}
        {fileError && <p className="text-[12.5px] font-medium text-danger">{fileError}</p>}

        {(stage === "selected" || stage === "importing") && file && (
          <div className="flex items-center gap-3 rounded-lg border border-border bg-surface-muted/40 px-4 py-3">
            <FileSpreadsheet className="h-5 w-5 shrink-0 text-navy-900" />
            <span className="flex-1 truncate text-[13px] font-medium text-text-primary">{file.name}</span>
            {stage === "selected" && (
              <button
                type="button"
                onClick={handleChangeFile}
                aria-label="Hapus berkas"
                className="text-text-secondary hover:text-danger"
              >
                <X className="h-4 w-4" />
              </button>
            )}
            {stage === "importing" && <span className="text-[12.5px] text-text-secondary">Mengimpor...</span>}
          </div>
        )}
        {importError && <p className="text-[12.5px] font-medium text-danger">{importError}</p>}

        {stage === "result" && result && (
          <div className="flex flex-col gap-2 rounded-lg border border-border bg-surface-muted/40 px-4 py-3">
            <p className="text-[13.5px] font-semibold text-text-primary">
              {result.insertedCount} {entityLabel} baru ditambahkan, {result.updatedCount} {entityLabel} diperbarui (
              {matchCriteriaLabel} sudah ada sebelumnya).
            </p>
            {result.errors.length > 0 && (
              <div className="mt-1 flex flex-col gap-1">
                <p className="text-[12.5px] font-semibold text-danger">{result.errors.length} baris gagal diproses:</p>
                <ul className="flex flex-col gap-0.5 text-[12.5px] text-text-secondary">
                  {result.errors.map((e, idx) => (
                    <li key={idx}>
                      {e.row > 0 ? `Baris ${e.row}: ` : ""}
                      {e.message}
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        )}
      </div>
    </Modal>
  );
}
