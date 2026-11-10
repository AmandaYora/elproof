import { useRef } from "react";
import { Button } from "@/shared/components/ui/Button";
import { AlertDialog, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/shared/components/kit/alert-dialog";
import { revalidateAll, useConflictStore } from "@/shared/stores/useConflictStore";

/**
 * Shown when a save was refused because someone saved the same item first. Nothing was overwritten. Offers the one safe way
 * forward (load the latest, then make the change again) and a way to stay put, for a person who wants to copy what they typed.
 * Loaded on demand: it is only ever needed after a refusal.
 */
export default function ConflictDialog() {
  const open = useConflictStore((s) => s.open);
  const dismiss = useConflictStore((s) => s.dismiss);
  const stayRef = useRef<HTMLButtonElement>(null);

  return (
    <AlertDialog open={open} onOpenChange={(next) => !next && dismiss()}>
      <AlertDialogContent initialFocus={stayRef}>
        <AlertDialogHeader>
          <AlertDialogTitle>Data diubah orang lain</AlertDialogTitle>
          <AlertDialogDescription className="text-left text-[13.5px] leading-relaxed">
            Seseorang menyimpan perubahan pada data ini setelah Anda membukanya, jadi perubahan Anda belum tersimpan. Muat ulang untuk melihat versi terbaru, lalu ulangi perubahan Anda.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <Button ref={stayRef} variant="secondary" onClick={dismiss}>
            Tetap di sini
          </Button>
          <Button
            variant="primary"
            onClick={() => {
              dismiss();
              revalidateAll();
            }}
          >
            Muat ulang
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
