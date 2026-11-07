import { useEffect, useState } from "react";
import QRCode from "qrcode";
import { cn } from "@/shared/lib/cn";

interface QrCodeProps {
  /** Text to encode (a URL, an otpauth:// link, ...). */
  value: string;
  /** Rendered edge length in px. */
  size?: number;
  /** Describes what the code opens, for screen readers. */
  label: string;
  className?: string;
}

/** Draws a QR code in the browser, so the encoded text never leaves the page. */
export function QrCode({ value, size = 176, label, className }: QrCodeProps) {
  const [src, setSrc] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let cancelled = false;
    QRCode.toDataURL(value, { width: size * 2, margin: 1, errorCorrectionLevel: "M", color: { dark: "#0f172a", light: "#ffffff" } })
      .then((url) => {
        if (!cancelled) {
          setSrc(url);
          setFailed(false);
        }
      })
      .catch(() => {
        if (!cancelled) setFailed(true);
      });
    return () => {
      cancelled = true;
    };
  }, [value, size]);

  if (failed) return null;
  return (
    <div className={cn("rounded-lg border border-border bg-white p-2", className)} style={{ width: size + 18, height: size + 18 }}>
      {src ? <img src={src} alt={label} width={size} height={size} className="block" /> : <div style={{ width: size, height: size }} className="animate-pulse rounded bg-surface-muted" />}
    </div>
  );
}
