import { CircleAlert } from "lucide-react";
import { focusField } from "@/shared/forms/useZodForm";

interface Props {
  /** One entry per failed field; shown only from 3 entries up (fewer are clear from the fields themselves). */
  errors: Array<{ key: string; message: string }>;
  /** Human labels by field key, so the list reads "Nama vendor: wajib diisi". */
  labels?: Record<string, string>;
  /** A message about the whole form (a failed save, a rule that is not about one field). */
  formError?: string | null;
  /** Where the fields live, for the jump links (default: the whole page). */
  root?: () => ParentNode | null;
}

/**
 * The list of problems, focused when the person presses submit and 3 or more
 * fields are wrong (DESIGN.md §6), each item jumping to its field. A single
 * message about the whole form (formError) is always shown.
 */
export function FormErrorSummary({ errors, labels, formError, root }: Props) {
  const showList = errors.length >= 3;
  if (!showList && !formError) return null;
  return (
    <div id="form-error-summary" tabIndex={-1} role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-4 py-3 text-[13px] text-danger-text outline-none focus-visible:ring-2 focus-visible:ring-danger-text/40">
      {formError && (
        <p className="flex items-start gap-2 font-medium">
          <CircleAlert aria-hidden className="mt-px size-4 shrink-0" />
          {formError}
        </p>
      )}
      {showList && (
        <>
          <p className={`flex items-start gap-2 font-semibold ${formError ? "mt-2" : ""}`}>
            {!formError && <CircleAlert aria-hidden className="mt-px size-4 shrink-0" />}
            Ada {errors.length} isian yang perlu diperbaiki
          </p>
          <ul className="mt-1.5 list-disc space-y-0.5 pl-9">
            {errors.map((e) => (
              <li key={e.key}>
                <button type="button" onClick={() => focusField(root?.() ?? document, e.key)} className="text-left underline underline-offset-2 hover:no-underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-current">
                  {labels?.[e.key] ? `${labels[e.key]}: ` : ""}
                  {e.message}
                </button>
              </li>
            ))}
          </ul>
        </>
      )}
    </div>
  );
}
