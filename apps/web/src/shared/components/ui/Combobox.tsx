import {
  Children,
  isValidElement,
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
  type ReactElement,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";
import { Check, ChevronDown, Search } from "lucide-react";
import { cn } from "@/shared/lib/cn";

interface ComboboxOption {
  value: string;
  label: string;
  disabled?: boolean;
}

interface OptionElementProps {
  value?: string;
  children?: ReactNode;
  disabled?: boolean;
}

function getNodeText(node: ReactNode): string {
  if (node === null || node === undefined || typeof node === "boolean") return "";
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(getNodeText).join("");
  if (isValidElement(node)) return getNodeText((node.props as OptionElementProps).children);
  return "";
}

function optionsFromChildren(children: ReactNode): ComboboxOption[] {
  return Children.toArray(children)
    .filter((child): child is ReactElement<OptionElementProps> => isValidElement(child))
    .map((child) => ({
      value: String(child.props.value ?? ""),
      label: getNodeText(child.props.children),
      disabled: child.props.disabled,
    }));
}

interface ComboboxProps {
  value: string;
  onChange: (e: { target: { value: string } }) => void;
  children: ReactNode;
  className?: string;
  disabled?: boolean;
  placeholder?: string;
}

export function Combobox({ value, onChange, children, className, disabled, placeholder = "Pilih..." }: ComboboxProps) {
  const options = useMemo(() => optionsFromChildren(children), [children]);
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [highlighted, setHighlighted] = useState(0);
  const [rect, setRect] = useState<{ top: number; left: number; width: number } | null>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);

  const selected = options.find((o) => o.value === value);
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (q === "") return options;
    return options.filter((o) => o.label.toLowerCase().includes(q));
  }, [options, query]);

  function openDropdown() {
    if (disabled) return;
    const r = triggerRef.current?.getBoundingClientRect();
    if (r) setRect({ top: r.bottom + 4, left: r.left, width: r.width });
    setQuery("");
    setHighlighted(Math.max(0, options.findIndex((o) => o.value === value)));
    setOpen(true);
  }

  function closeDropdown() {
    setOpen(false);
  }

  function selectOption(opt: ComboboxOption) {
    if (opt.disabled) return;
    onChange({ target: { value: opt.value } });
    closeDropdown();
    triggerRef.current?.focus();
  }

  useEffect(() => {
    if (!open) return;
    requestAnimationFrame(() => searchRef.current?.focus());

    function onMouseDown(e: MouseEvent) {
      const target = e.target as Node;
      if (triggerRef.current?.contains(target) || panelRef.current?.contains(target)) return;
      closeDropdown();
    }
    function onViewportChange() {
      closeDropdown();
    }

    document.addEventListener("mousedown", onMouseDown);
    window.addEventListener("scroll", onViewportChange, true);
    window.addEventListener("resize", onViewportChange);
    return () => {
      document.removeEventListener("mousedown", onMouseDown);
      window.removeEventListener("scroll", onViewportChange, true);
      window.removeEventListener("resize", onViewportChange);
    };
  }, [open]);

  function handleTriggerKeyDown(e: KeyboardEvent<HTMLButtonElement>) {
    if (!open && (e.key === "Enter" || e.key === " " || e.key === "ArrowDown" || e.key === "ArrowUp")) {
      e.preventDefault();
      openDropdown();
    }
  }

  function handleSearchKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setHighlighted((h) => Math.min(filtered.length - 1, h + 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setHighlighted((h) => Math.max(0, h - 1));
    } else if (e.key === "Enter") {
      e.preventDefault();
      const opt = filtered[highlighted];
      if (opt) selectOption(opt);
    } else if (e.key === "Escape") {
      e.preventDefault();
      closeDropdown();
      triggerRef.current?.focus();
    }
  }

  return (
    <div className={cn("relative", className)}>
      <button
        ref={triggerRef}
        type="button"
        disabled={disabled}
        onClick={() => (open ? closeDropdown() : openDropdown())}
        onKeyDown={handleTriggerKeyDown}
        className={cn(
          "flex h-9 w-full items-center justify-between gap-2 rounded-md border border-border bg-white px-3 text-left text-sm text-text-primary transition-colors",
          "focus:outline-none focus:ring-2 focus:ring-navy-900/20 focus:border-navy-900",
          "disabled:cursor-not-allowed disabled:bg-surface-muted disabled:text-text-secondary",
          open && "border-navy-900 ring-2 ring-navy-900/20"
        )}
      >
        <span className={cn("truncate", !selected && "text-text-secondary/70")}>{selected ? selected.label : placeholder}</span>
        <ChevronDown className={cn("h-4 w-4 shrink-0 text-text-secondary transition-transform", open && "rotate-180")} />
      </button>

      {open &&
        rect &&
        createPortal(
          <div
            ref={panelRef}
            style={{ position: "fixed", top: rect.top, left: rect.left, width: rect.width }}
            className="z-50 overflow-hidden rounded-md border border-border bg-white shadow-lg"
          >
            <div className="relative border-b border-border-light p-1.5">
              <Search className="pointer-events-none absolute left-4 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-text-secondary" />
              <input
                ref={searchRef}
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value);
                  setHighlighted(0);
                }}
                onKeyDown={handleSearchKeyDown}
                placeholder="Cari..."
                className="h-8 w-full rounded-md border-0 bg-surface-muted pl-8 pr-2 text-[13px] text-text-primary placeholder:text-text-secondary/70 focus:outline-none"
              />
            </div>
            <ul role="listbox" className="max-h-52 overflow-y-auto py-1">
              {filtered.length === 0 ? (
                <li className="px-3 py-2.5 text-center text-[13px] text-text-secondary">Tidak ditemukan</li>
              ) : (
                filtered.map((opt, idx) => (
                  <li
                    key={opt.value}
                    role="option"
                    aria-selected={opt.value === value}
                    onMouseEnter={() => setHighlighted(idx)}
                    onClick={() => selectOption(opt)}
                    className={cn(
                      "flex cursor-pointer items-center justify-between gap-2 px-3 py-2 text-[13px]",
                      idx === highlighted && "bg-surface-muted",
                      opt.disabled && "pointer-events-none text-text-secondary/50"
                    )}
                  >
                    <span className={cn("truncate", opt.value === value && "font-semibold text-navy-900")}>{opt.label}</span>
                    {opt.value === value && <Check className="h-3.5 w-3.5 shrink-0 text-navy-900" />}
                  </li>
                ))
              )}
            </ul>
          </div>,
          document.body
        )}
    </div>
  );
}
