import { Link } from "react-router-dom";
import { ChevronRight, FileText, HeartHandshake } from "lucide-react";
import { useClientHomeStore } from "@/modules/client-portal/stores/useClientHomeStore";
import { useCan } from "@/shared/stores/usePermissionStore";
import { HomeHero } from "@/modules/client-portal/components/home/HomeHero";
import { ActionCard } from "@/modules/client-portal/components/home/ActionCard";
import { NextPaymentCard } from "@/modules/client-portal/components/home/NextPaymentCard";
import { NextVendorBillCard } from "@/modules/client-portal/components/home/NextVendorBillCard";
import { ProgressStages } from "@/modules/client-portal/components/home/ProgressStages";
import { RecentUpdates } from "@/modules/client-portal/components/home/RecentUpdates";
import { ContactCard } from "@/modules/client-portal/components/home/ContactCard";
import { InstallHint } from "@/modules/client-portal/components/home/InstallHint";
import { ClientTasksCard, isUrgentTask } from "@/modules/client-portal/components/home/ClientTasks";
import { useProofViewer } from "@/modules/client-portal/components/useProofViewer";
import { usePortalContact } from "@/modules/client-portal/lib/usePortalContact";
import { useLastSeen } from "@/modules/client-portal/lib/useLastSeen";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

const ROW_LINK =
  "flex min-h-14 items-center gap-3 rounded-2xl border border-border bg-card p-4 hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring sm:p-5";

/**
 * The portal's landing page: what to do, what to pay, how far along, who to call. Everything comes from one call.
 * On a phone it is one column in that order; from 1024 px the things to act on sit left and the things to read right.
 */
export default function HomeTabPage() {
  const home = useClientHomeStore((s) => s.home);
  const canDocuments = useCan("portal.documents.view");
  const contact = usePortalContact();
  const viewer = useProofViewer(home?.project.id ?? "");
  const lastSeen = useLastSeen(home?.project.id ?? "");
  if (!home) return null;

  const { project, finance, progress } = home;
  const open = project.status !== "Cancelled";
  const tasks = home.clientTasks ?? [];
  const urgentTasks = open ? tasks.filter(isUrgentTask).sort((a, b) => a.targetDate.localeCompare(b.targetDate)) : [];
  const actions = open ? home.actions : [];
  const issuesLink = (home.openIssueCount ?? 0) > 0 && !home.actions.some((a) => a.kind === "open_issues");

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-4 sm:gap-5 lg:max-w-none">
      <HomeHero project={project} daysToEvent={home.countdown.daysToEvent} />

      <div className="grid grid-cols-1 items-start gap-4 sm:gap-5 lg:grid-cols-[minmax(0,7fr)_minmax(0,5fr)] lg:gap-6">
        <div className="flex flex-col gap-4 sm:gap-5">
          <ActionCard actions={actions} tasks={urgentTasks} contact={contact} />
          {open && finance && <NextPaymentCard finance={finance} payTo={home.payTo} contact={contact} />}
          {open && finance?.nextSupplierBill && <NextVendorBillCard bill={finance.nextSupplierBill} />}
          {open && tasks.length > 0 && <ClientTasksCard tasks={tasks} eventDate={project.eventDate} />}

          {issuesLink && (
            <Link to={ROUTE_PATHS.portal("kendala")} className={ROW_LINK}>
              <span aria-hidden className="flex size-10 shrink-0 items-center justify-center rounded-full bg-warning-soft text-warning-text">
                <HeartHandshake className="size-5" />
              </span>
              <span className="min-w-0 flex-1">
                <span className="block text-[14px] font-semibold text-foreground">{home.openIssueCount} hal sedang kami tangani</span>
                <span className="block text-[13px] text-muted-foreground">Lihat apa yang terjadi dan apa yang sedang kami lakukan.</span>
              </span>
              <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
            </Link>
          )}
        </div>

        <div className="flex flex-col gap-4 sm:gap-5">
          <ProgressStages status={project.status} daysToEvent={home.countdown.daysToEvent} percent={progress?.overallPercent} />
          <RecentUpdates updates={home.recentUpdates} onOpenProof={viewer.open} lastSeen={lastSeen} />
          <ContactCard contact={contact} />
          {canDocuments && (
            <Link to={ROUTE_PATHS.portal("dokumen")} className={ROW_LINK}>
              <FileText className="size-5 shrink-0 text-primary" aria-hidden />
              <span className="min-w-0 flex-1">
                <span className="block text-[14px] font-semibold text-foreground">Dokumen penting</span>
                <span className="block text-[13px] text-muted-foreground">Rundown, buku acara, dan lainnya</span>
              </span>
              <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
            </Link>
          )}
          <InstallHint />
        </div>
      </div>
      {viewer.element}
    </div>
  );
}
