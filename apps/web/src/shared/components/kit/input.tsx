import * as React from "react"
import { Input as InputPrimitive } from "@base-ui/react/input"
import { cn } from "cn"

// The control's boundary uses --input (3.7:1 on white, WCAG 1.4.11); text is 16px
// on small screens so iOS Safari does not zoom the page on focus.
function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <InputPrimitive
      type={type}
      data-slot="input"
      className={cn(
        "h-(--control-h) w-full min-w-0 rounded-md border border-input bg-card px-3 text-base text-foreground transition-colors outline-none file:inline-flex file:h-6 file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-subtle-foreground focus-visible:border-ring focus-visible:outline-2 focus-visible:outline-offset-0 focus-visible:outline-ring disabled:cursor-not-allowed disabled:bg-muted disabled:text-muted-foreground aria-invalid:border-danger aria-invalid:outline-danger md:text-sm pointer-coarse:h-11",
        className
      )}
      {...props}
    />
  )
}

export { Input }
