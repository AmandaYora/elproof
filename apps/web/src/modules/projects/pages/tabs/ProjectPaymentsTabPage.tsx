import { Outlet, useOutletContext } from "react-router-dom";
import { TabNav } from "@/shared/components/ui/TabNav";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

const PEMBAYARAN_TABS = [
  { to: "client", label: "Dari Client" },
  { to: "vendor", label: "Ke Vendor" },
  { to: "venue", label: "Ke Venue" },
];

// Layout for the three payment ledgers, mirroring ProjectDetailLayout's own
// TabNav + Outlet shape one level deeper (PLAN.md "Venue Payments +
// Pembayaran tab restructuring") -- reuses TabNav as-is since it's already
// URL-driven, not local-state.
export default function ProjectPaymentsTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  return (
    <div className="flex flex-col gap-5">
      <TabNav items={PEMBAYARAN_TABS} />
      <Outlet context={{ projectId } satisfies ProjectDetailContext} />
    </div>
  );
}
