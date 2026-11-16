import { Outlet, useOutletContext } from "react-router-dom";
import { TabNav } from "@/shared/components/ui/TabNav";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

const FINANCE_VIEWS = [
  { to: "klien", label: "Dari klien" },
  { to: "vendor", label: "Ke vendor" },
  { to: "venue", label: "Ke venue" },
];

/** Keuangan tab: the three ledgers of a project (money in from the client, out to vendors and to the venue), one per address. */
export default function ProjectFinanceTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  return (
    <div className="flex flex-col gap-5">
      <TabNav items={FINANCE_VIEWS} variant="pills" aria-label="Keuangan proyek" />
      <Outlet context={{ projectId } satisfies ProjectDetailContext} />
    </div>
  );
}
