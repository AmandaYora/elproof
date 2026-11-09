import { Toaster as Sonner, type ToasterProps } from "sonner"
import { CircleCheckIcon, InfoIcon, OctagonXIcon, TriangleAlertIcon } from "lucide-react"

/**
 * Sonner on the app's tokens. The app has no dark mode, so the theme is fixed
 * (next-themes from the shadcn template is not used). Status icons use the
 * `-text` tokens so they meet contrast on the card background.
 */
const Toaster = ({ ...props }: ToasterProps) => {
  return (
    <Sonner
      theme="light"
      position="bottom-right"
      containerAriaLabel="Notifikasi"
      className="toaster group"
      icons={{
        success: <CircleCheckIcon className="size-4 text-success-text" />,
        info: <InfoIcon className="size-4 text-info-text" />,
        warning: <TriangleAlertIcon className="size-4 text-warning-text" />,
        error: <OctagonXIcon className="size-4 text-danger-text" />,
      }}
      style={
        {
          "--normal-bg": "var(--popover)",
          "--normal-text": "var(--popover-foreground)",
          "--normal-border": "var(--border)",
          "--border-radius": "var(--radius-lg, 0.5rem)",
          "--width": "min(24rem, calc(100vw - 2rem))",
        } as React.CSSProperties
      }
      toastOptions={{
        classNames: {
          toast: "!shadow-lg !text-[13.5px] !gap-3",
          actionButton: "!bg-primary !text-primary-foreground !font-semibold !h-8 !px-3 !rounded-md",
          closeButton: "!border-border",
        },
      }}
      {...props}
    />
  )
}

export { Toaster }
