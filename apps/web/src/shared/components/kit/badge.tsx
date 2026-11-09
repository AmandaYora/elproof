import { mergeProps } from "@base-ui/react/merge-props"
import { useRender } from "@base-ui/react/use-render"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "cn"

// Status pills (DESIGN.md §4.2): the text always uses a -text token (>= 4.5:1 on
// the -soft background), never the bright status color. Callers add an icon
// next to the text so status is never conveyed by color alone.
const badgeVariants = cva(
  "inline-flex w-fit shrink-0 items-center justify-center gap-1.5 rounded-full border border-transparent px-2.5 py-1 text-xs leading-none font-semibold whitespace-nowrap [&>svg]:pointer-events-none [&>svg]:size-3.5 [&>svg]:shrink-0",
  {
    variants: {
      variant: {
        neutral: "bg-neutral-soft text-muted-foreground",
        success: "bg-success-soft text-success-text",
        warning: "bg-warning-soft text-warning-text",
        danger: "bg-danger-soft text-danger-text",
        info: "bg-info-soft text-info-text",
        brand: "bg-accent text-accent-foreground",
        outline: "border-border bg-card text-foreground",
      },
    },
    defaultVariants: {
      variant: "neutral",
    },
  }
)

function Badge({
  className,
  variant = "neutral",
  render,
  ...props
}: useRender.ComponentProps<"span"> & VariantProps<typeof badgeVariants>) {
  return useRender({
    defaultTagName: "span",
    props: mergeProps<"span">(
      {
        className: cn(badgeVariants({ variant }), className),
      },
      props
    ),
    render,
    state: {
      slot: "badge",
      variant,
    },
  })
}

export { Badge, badgeVariants }
