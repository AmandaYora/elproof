// Class-name joining plus Tailwind conflict resolution ("px-2 px-4" keeps the
// last). `cn` is shadcn's drop-in replacement for clsx + tailwind-merge, the
// same one the shadcn components import.
export { cn } from "cn";
export type { ClassValue } from "cn";
