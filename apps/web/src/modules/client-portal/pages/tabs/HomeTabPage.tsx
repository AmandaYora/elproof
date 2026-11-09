import { Link } from "react-router-dom";
import { ChevronRight, FileText, HeartHandshake } from "lucide-react";
import { useClientHomeStore } from "@/modules/client-portal/stores/useClientHomeStore";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { useCan } from "@/shared/stores/usePermissionStore";
import { HomeHero } from "@/modules/client-portal/components/home/HomeHero";
import { ActionCard } from "@/modules/client-portal/components/home/ActionCard";
import { NextPaymentCard } from "@/modules/client-portal/components/home/NextPaymentCard";
import { ProgressStages } from "@/modules/client-portal/components/home/ProgressStages";
import { RecentUpdates } from "@/modules/client-portal/components/home/RecentUpdates";
import { ContactCard } from "@/modules/client-portal/components/home/ContactCard";
import { InstallHint } from "@/modules/client-portal/components/home/InstallHint";
import { useProofViewer } from "@/modules/client-portal/components/useProofViewer";
import { pickContactPhone } from "@/modules/client-portal/lib/contact-phone";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

/** The portal's landing page: what to do, what to pay, how far along, who to call. Everything comes from one call. */
export default function HomeTabPage() {
  const home = useClientHomeStore((s) => s.home);
  const clientName = useAuthStore((s) => s.session?.displayName ?? "");
  const businessName = useTenantBrandingStore((s) => s.businessName) ?? "";
  const canDocuments = useCan("portal.documents.view");
  const viewer = useProofViewer(home?.project.id ?? "");
  if (!home) return null;

  const { project, finance, progress } = home;
  const open = project.status !== "Cancelled";
  const phone = pickContactPhone(home.pic.phone || project.picPhone, home.contact?.whatsapp);
  // Open issues already appear as an action when there are any, so the page has one place that asks for attention.
  const showActions = open && home.actions.length > 0;

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-4 sm:gap-5">
      <HomeHero project={project} daysToEvent={home.countdown.daysToEvent} />

      {showActions && <ActionCard actions={home.actions} />}

      {open && finance && (
        <NextPaymentCard
          finance={finance}
          payTo={home.payTo}
          projectName={project.name}
          clientName={clientName}
          businessName={businessName}
          contactPhone={phone}
        />
      )}

      <div className="grid grid-cols-1 gap-4 sm:gap-5 md:grid-cols-2">
        <ProgressStages status={project.status} daysToEvent={home.countdown.daysToEvent} percent={progress?.overallPercent} />
        <div className="flex flex-col gap-4 sm:gap-5">
          <RecentUpdates updates={home.recentUpdates} onOpenProof={viewer.open} />
          <ContactCard picName={home.pic.name || project.picName} phone={phone} businessName={businessName} clientName={clientName} projectName={project.name} />
        </div>
      </div>

      {(home.openIssueCount ?? 0) > 0 && !home.actions.some((a) => a.kind === "open_issues") && (
        <Link
          to={ROUTE_PATHS.portal("kendala")}
          className="flex min-h-14 items-center gap-3 rounded-2xl border border-border bg-card p-4 hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground sm:p-5"
        >
          <span aria-hidden className="flex size-10 shrink-0 items-center justify-center rounded-full bg-warning-soft text-warning-text">
            <HeartHandshake className="size-5" />
          </span>
          <span className="min-w-0">
            <span className="block text-[14px] font-semibold text-foreground">{home.openIssueCount} hal sedang kami tangani</span>
            <span className="block text-[13px] text-muted-foreground">Lihat apa yang terjadi dan apa yang sedang kami lakukan.</span>
          </span>
        </Link>
      )}

      {canDocuments && (
        <Link
          to={ROUTE_PATHS.portal("dokumen")}
          className="flex min-h-14 items-center gap-3 rounded-2xl border border-border bg-card p-4 hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground sm:p-5"
        >
          <FileText className="size-5 shrink-0 text-navy-900" aria-hidden />
          <span className="min-w-0 flex-1">
            <span className="block text-[14px] font-semibold text-foreground">Dokumen penting</span>
            <span className="block text-[12.5px] text-muted-foreground">Rundown, buku acara, dan lainnya</span>
          </span>
          <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
        </Link>
      )}

      <InstallHint />
      {viewer.element}
    </div>
  );
}
