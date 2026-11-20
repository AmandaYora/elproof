import { Outlet, useOutletContext } from "react-router-dom";
import { TabNav } from "@/shared/components/ui/TabNav";
import { useCan } from "@/shared/stores/usePermissionStore";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

/**
 * Keuangan tab: the project's three books, one per address. Money in from the client, what the WO pays its vendors and
 * venues (only with finance.margin.view: the "Biaya rekanan" is the margin's other half), and what the client paid
 * vendors and venues directly.
 */
export default function ProjectFinanceTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  const canMargin = useCan("finance.margin.view");
  const views = [
    { to: "klien", label: "Klien" },
    ...(canMargin ? [{ to: "vendor-venue", label: "Vendor & venue dibayar WO" }] : []),
    { to: "dibayar-klien", label: "Dibayar klien langsung" },
  ];
  return (
    <div className="flex flex-col gap-5">
      <TabNav items={views} variant="pills" aria-label="Keuangan proyek" />
      <Outlet context={{ projectId } satisfies ProjectDetailContext} />
    </div>
  );
}
