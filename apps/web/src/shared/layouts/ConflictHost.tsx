import { lazy, Suspense, useEffect, useState } from "react";
import { useConflictStore } from "@/shared/stores/useConflictStore";

const ConflictDialog = lazy(() => import("@/shared/layouts/ConflictDialog"));

/** Mounts the "data diubah orang lain" dialog the first time a save is refused; until then it costs nothing. */
export function ConflictHost() {
  const open = useConflictStore((s) => s.open);
  const [wanted, setWanted] = useState(false);
  useEffect(() => {
    if (open) setWanted(true);
  }, [open]);
  if (!wanted && !open) return null;
  return (
    <Suspense fallback={null}>
      <ConflictDialog />
    </Suspense>
  );
}
