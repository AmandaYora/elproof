import * as React from "react"
import { cn } from "cn"

function Textarea({ className, ...props }: React.ComponentProps<"textarea">) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        "flex min-h-20 w-full rounded-md border border-input bg-card px-3 py-2 text-base text-foreground transition-colors outline-none placeholder:text-subtle-foreground focus-visible:border-ring focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-offset-0 outline-ring disabled:cursor-not-allowed disabled:bg-muted disabled:text-muted-foreground aria-invalid:border-danger aria-invalid:outline-danger md:text-sm",
        className
      )}
      {...props}
    />
  )
}

export { Textarea }
