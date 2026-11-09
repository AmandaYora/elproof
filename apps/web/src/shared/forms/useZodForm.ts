import { useCallback, useMemo, useRef, useState, type FormEvent } from "react";
import type { ZodTypeAny, z } from "zod";
import { getApiFieldErrors, getApiErrorMessage } from "@/shared/lib/api-error";

export type FormErrors = Record<string, string>;

/** "terms.2.amount" -> "terms[2].amount", the key the API uses for a row. */
function pathKey(path: Array<string | number>): string {
  return path.reduce<string>((acc, part) => (typeof part === "number" ? `${acc}[${part}]` : acc ? `${acc}.${part}` : part), "");
}

function collect(schema: ZodTypeAny, values: unknown): { data?: unknown; errors: FormErrors } {
  const result = schema.safeParse(values);
  if (result.success) return { data: result.data, errors: {} };
  const errors: FormErrors = {};
  for (const issue of result.error.issues) {
    const key = pathKey(issue.path as Array<string | number>);
    errors[key] ??= issue.message;
  }
  return { errors };
}

/** Moves focus to a field by name (Field puts data-field on its wrapper). */
export function focusField(root: ParentNode | null, name: string): boolean {
  const el = root?.querySelector<HTMLElement>(`[data-field="${CSS.escape(name)}"] :is(input, textarea, select, button, [tabindex]):not([type="hidden"]):not([disabled])`);
  el?.focus();
  return Boolean(el);
}

type Input<S extends ZodTypeAny> = z.input<S>;
type Output<S extends ZodTypeAny> = z.output<S>;
/** Keys whose input may be missing but whose output is always there: the schema gave them a default. */
type DefaultedKeys<S extends ZodTypeAny> = {
  [K in keyof Input<S>]-?: undefined extends Output<S>[K] ? never : undefined extends Input<S>[K] ? K : never;
}[keyof Input<S>];

/**
 * The form's values: what the inputs hold (the schema's INPUT shape, so a field that coerces text to a number keeps
 * text), except that a field with a schema default is never undefined, and a truly optional one (an attached file)
 * stays optional.
 */
export type FormValues<S extends ZodTypeAny> = Pick<Input<S>, Exclude<keyof Input<S>, DefaultedKeys<S>>> & {
  [K in DefaultedKeys<S>]-?: Exclude<Input<S>[K], undefined>;
};

interface Options<S extends ZodTypeAny> {
  schema: S;
  initial: FormValues<S>;
}

/**
 * Form state with the hybrid validation DESIGN.md §6 asks for:
 *  1. nothing is shown while the person is still typing their first pass;
 *  2. pressing the submit button checks everything and shows every problem at once
 *     (focus goes to the summary when there are 3 or more, else to the first field);
 *  3. a field that was left (blur) is checked right then, and rechecked as it is
 *     corrected so the message disappears the moment it is fixed;
 *  4. the submit button is never disabled because the form is invalid, only while
 *     a request is running; typed input is never cleared.
 * A 422 from the server is mapped onto the same fields (row keys like
 * `terms[2].amount` included); anything it says about a field the form does not
 * have goes to `formError`.
 */
export function useZodForm<S extends ZodTypeAny>({ schema, initial }: Options<S>) {
  type Values = FormValues<S>;
  type Parsed = z.output<S>;
  const [values, setValuesState] = useState<Values>(initial);
  const [errors, setErrors] = useState<FormErrors>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const touched = useRef(new Set<string>());
  const attempted = useRef(false);
  // Messages that came from the server stay until the person edits that field (client rules cannot know them).
  const serverKeys = useRef(new Set<string>());
  const valuesRef = useRef(values);
  valuesRef.current = values;

  const recheck = useCallback(
    (next: Values, only?: string) => {
      const { errors: all } = collect(schema, next);
      setErrors((prev) => {
        const out: FormErrors = { ...prev };
        for (const key of Object.keys(out)) if (!(key in all) && !serverKeys.current.has(key)) delete out[key]; // fixed: drop its message
        if (only) {
          if (all[only]) out[only] = all[only];
          else delete out[only];
        } else Object.assign(out, all);
        return out;
      });
    },
    [schema],
  );

  const set = useCallback(
    <K extends keyof Values & string>(key: K, value: Values[K]) => {
      const next = { ...valuesRef.current, [key]: value } as Values;
      valuesRef.current = next;
      setValuesState(next);
      serverKeys.current.delete(key);
      // Already shown wrong (or the person tried to submit): keep it current as they type.
      if (attempted.current || touched.current.has(key)) recheck(next, attempted.current ? undefined : key);
      else setErrors((prev) => (key in prev ? Object.fromEntries(Object.entries(prev).filter(([k]) => k !== key)) : prev));
    },
    [recheck],
  );

  const setValues = useCallback((next: Values) => {
    valuesRef.current = next;
    setValuesState(next);
  }, []);

  const blur = useCallback(
    (key: string) => {
      touched.current.add(key);
      recheck(valuesRef.current, key);
    },
    [recheck],
  );

  const reset = useCallback(
    (next: Values) => {
      touched.current.clear();
      serverKeys.current.clear();
      attempted.current = false;
      valuesRef.current = next;
      setValuesState(next);
      setErrors({});
      setFormError(null);
    },
    [],
  );

  /** Errors the server reported for fields, merged over the client ones. */
  const applyServerError = useCallback((err: unknown, fallback: string, known?: (key: string) => boolean) => {
    const fields = getApiFieldErrors(err);
    const mine: FormErrors = {};
    const stray: string[] = [];
    for (const [key, message] of Object.entries(fields)) {
      if (!known || known(key)) mine[key] = message;
      else stray.push(message);
    }
    for (const key of Object.keys(mine)) serverKeys.current.add(key);
    setErrors((prev) => ({ ...prev, ...mine }));
    const message = getApiErrorMessage(err, fallback);
    // The overall message is shown when nothing can be pinned to a field, or when the server says more than the fields do.
    setFormError(Object.keys(mine).length > 0 && stray.length === 0 ? null : stray[0] ?? message);
    return Object.keys(mine);
  }, []);

  /**
   * The onSubmit handler. `run` gets the parsed values and may throw: a 422 is
   * mapped onto the fields, any other error becomes `formError`.
   */
  const handleSubmit = useCallback(
    (run: (parsed: Parsed) => Promise<void> | void, opts: { root?: () => ParentNode | null; fallback?: string; known?: (key: string) => boolean } = {}) =>
      async (e?: FormEvent) => {
        e?.preventDefault();
        if (submitting) return;
        attempted.current = true;
        setFormError(null);
        const { data, errors: found } = collect(schema, valuesRef.current);
        setErrors(found);
        const keys = Object.keys(found);
        if (keys.length > 0) {
          requestAnimationFrame(() => {
            if (keys.length >= 3) document.getElementById("form-error-summary")?.focus();
            // A problem on a field that is not on screen must never be silent: say it for the whole form.
            else if (!focusField(opts.root?.() ?? document, keys[0])) setFormError(found[keys[0]]);
          });
          return;
        }
        setSubmitting(true);
        try {
          await run(data as Parsed);
        } catch (err) {
          const pinned = applyServerError(err, opts.fallback ?? "Belum tersimpan. Periksa koneksi lalu coba lagi.", opts.known);
          requestAnimationFrame(() => {
            if (pinned.length >= 3) document.getElementById("form-error-summary")?.focus();
            else if (pinned.length > 0) focusField(opts.root?.() ?? document, pinned[0]);
          });
        } finally {
          setSubmitting(false);
        }
      },
    [schema, submitting, applyServerError],
  );

  const errorList = useMemo(() => Object.entries(errors).map(([key, message]) => ({ key, message })), [errors]);

  return { values, set, setValues, errors, errorList, formError, setFormError, submitting, blur, reset, handleSubmit, applyServerError };
}
