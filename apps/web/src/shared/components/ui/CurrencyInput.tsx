import { forwardRef, useLayoutEffect, useRef, type ChangeEvent } from "react";
import { Input } from "@/shared/components/ui/Input";

interface CurrencyInputProps {
  value: number;
  onChange: (value: number) => void;
  id?: string;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
}

function countDigits(s: string): number {
  let n = 0;
  for (const ch of s) if (ch >= "0" && ch <= "9") n++;
  return n;
}

// Character index in `formatted` right after the Nth digit (N =
// digitsBeforeCursor) -- "." separators inserted by grouping don't count, so
// this stays correct no matter how many separators shift around a keystroke.
function cursorForDigitCount(formatted: string, digitsBeforeCursor: number): number {
  if (digitsBeforeCursor <= 0) return 0;
  let seen = 0;
  for (let i = 0; i < formatted.length; i++) {
    if (formatted[i] >= "0" && formatted[i] <= "9") {
      seen++;
      if (seen === digitsBeforeCursor) return i + 1;
    }
  }
  return formatted.length;
}

// Rupiah input with live thousand-separator dots (e.g. "5.000.000"), like a
// calculator's own display -- a native <input type="number"> can't show a
// "." in its value at all, so this renders type="text" and formats/parses
// manually. `value`/`onChange` stay plain integers (no formatting on
// either side), so this is a drop-in replacement for
// `<Input type="number" value={n} onChange={(e) => set(Number(e.target.value))} />`
// at every call site.
export const CurrencyInput = forwardRef<HTMLInputElement, CurrencyInputProps>(function CurrencyInput(
  { value, onChange, className, ...props },
  forwardedRef
) {
  const innerRef = useRef<HTMLInputElement | null>(null);
  const pendingCursor = useRef<number | null>(null);

  // Always derived from the canonical numeric `value`, never from the last
  // raw keystroke -- self-corrects artifacts like a leading zero the moment
  // React re-renders, the same way a native number input already does.
  const digits = String(Math.max(0, Math.trunc(value)));
  const formatted = Number(digits).toLocaleString("id-ID");

  useLayoutEffect(() => {
    if (pendingCursor.current !== null && innerRef.current) {
      innerRef.current.setSelectionRange(pendingCursor.current, pendingCursor.current);
      pendingCursor.current = null;
    }
  });

  function handleChange(e: ChangeEvent<HTMLInputElement>) {
    const raw = e.target.value;
    const cursorPos = e.target.selectionStart ?? raw.length;
    const digitsBeforeCursor = countDigits(raw.slice(0, cursorPos));
    const newDigits = raw.replace(/\D/g, "");
    pendingCursor.current = cursorForDigitCount(Number(newDigits || "0").toLocaleString("id-ID"), digitsBeforeCursor);
    onChange(newDigits === "" ? 0 : Number(newDigits));
  }

  return (
    <Input
      ref={(node) => {
        innerRef.current = node;
        if (typeof forwardedRef === "function") forwardedRef(node);
        else if (forwardedRef) forwardedRef.current = node;
      }}
      type="text"
      inputMode="numeric"
      value={formatted}
      onChange={handleChange}
      className={className}
      {...props}
    />
  );
});
