import { Button as ButtonPrimitive } from "@base-ui/react/button"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "cn"

// shadcn/ui Button on Base UI, tuned to ElProof (DESIGN.md §4/§6): solid
// plum primary, a visible 2px focus outline (the stock 50%-opacity ring does
// not reach 3:1; outline-solid is required because Tailwind 4's outline-none
// also sets --tw-outline-style: none, which outline-2 reads), control height from the density token (36px console, 44px
// portal) and 44px on coarse pointers.
const buttonVariants = cva(
  "inline-flex shrink-0 items-center justify-center rounded-md border border-transparent text-sm font-semibold whitespace-nowrap transition-colors outline-none select-none focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-offset-2 focus-visible:outline-ring disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-60 aria-invalid:border-danger [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:bg-primary-hover",
        outline: "border-input bg-card text-foreground hover:bg-muted aria-expanded:bg-muted",
        secondary: "border-border bg-card text-foreground hover:bg-muted aria-expanded:bg-muted",
        ghost: "bg-transparent text-muted-foreground hover:bg-muted hover:text-foreground aria-expanded:bg-muted",
        destructive: "bg-danger text-white hover:bg-danger/90",
        link: "h-auto px-0 text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-(--control-h) gap-2 px-4 pointer-coarse:h-11",
        sm: "h-8 gap-1.5 px-3 text-[13px] pointer-coarse:h-11",
        xs: "h-7 gap-1 px-2 text-xs pointer-coarse:h-11",
        lg: "h-11 gap-2 px-5 text-base",
        icon: "size-(--control-h) pointer-coarse:size-11",
        "icon-sm": "size-8 pointer-coarse:size-11",
        "icon-xs": "size-6 pointer-coarse:size-11",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

function Button({
  className,
  variant = "default",
  size = "default",
  ...props
}: ButtonPrimitive.Props & VariantProps<typeof buttonVariants>) {
  return (
    <ButtonPrimitive
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export { Button, buttonVariants }
