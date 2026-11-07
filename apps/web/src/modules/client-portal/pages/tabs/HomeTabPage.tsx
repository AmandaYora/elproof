import { useOutletContext } from "react-router-dom";
import { Link } from "react-router-dom";
import { ChevronRight, FileText, HeartHandshake } from "lucide-react";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { useCan } from "@/shared/stores/usePermissionStore";
import { daysUntil } from "@/modules/projects/lib/dates";
import { remainingToPay } from "@/modules/projects/lib/payments";
import { HomeHero } from "@/modules/client-portal/components/home/HomeHero";
import { ActionCard } from "@/modules/client-portal/components/home/ActionCard";
import { NextPaymentCard } from "@/modules/client-portal/components/home/NextPaymentCard";
import { ProgressStages } from "@/modules/client-portal/components/home/ProgressStages";
import { RecentUpdates } from "@/modules/client-portal/components/home/RecentUpdates";
import { ContactCard } from "@/modules/client-portal/components/home/ContactCard";
import { pickContactPhone } from "@/modules/client-portal/lib/contact-phone";
import { useContactStore } from "@/shared/stores/useContactStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";

const OPEN_ISSUE = new Set(["Open", "In Review", "In Resolution"]);

/** The portal's landing page: what to do, what to pay, how far along, who to call. */
export default function HomeTabPage() {
  const { finance, retryFinance } = useOutletContext<ClientPortalContext>();
  const project = useProjectStore((s) => s.currentProject);
  const milestones = useProjectStore((s) => s.milestones);
  const clientPayments = useProjectStore((s) => s.clientPayments);
  const terms = useProjectStore((s) => s.paymentTerms);
  const issues = useProjectStore((s) => s.issues);
  const contact = useContactStore((s) => s.contact);
  const contactFailed = useContactStore((s) => s.failed);
  const clientName = useAuthStore((s) => s.session?.displayName ?? "");
  const canFinance = useCan("portal.finance.view");
  const canIssues = useCan("portal.issues.view");
  const canDocuments = useCan("portal.documents.view");
  const canTimeline = useCan("portal.timeline.view");

  if (!project) return null;
  const days = daysUntil(project.eventDate);
  const remaining = remainingToPay(project.contractValue, clientPayments);
  const nextTerm = terms.filter((t) => t.status !== "paid").sort((a, b) => a.dueDate.localeCompare(b.dueDate))[0];
  const openIssues = canIssues ? issues.filter((i) => OPEN_ISSUE.has(i.status)).length : 0;
  const open = project.status !== "Cancelled";

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-4 sm:gap-5">
      <HomeHero project={project} />

      {open && canFinance && finance === "ready" && <ActionCard terms={terms} />}

      {openIssues > 0 && (
        <Link
          to={ROUTE_PATHS.portal("kendala")}
          className="flex items-center gap-3 rounded-2xl border border-border bg-white p-4 hover:bg-surface-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy-900 sm:p-5"
        >
          <span aria-hidden className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-warning-soft text-warning-strong">
            <HeartHandshake className="h-5 w-5" />
          </span>
          <span className="min-w-0">
            <span className="block text-[14px] font-semibold text-text-primary">{openIssues} hal sedang kami tangani</span>
            <span className="block text-[13px] text-text-secondary">Lihat apa yang terjadi dan apa yang sedang kami lakukan.</span>
          </span>
        </Link>
      )}

      {open && canFinance && finance === "loading" && (
        <div className="rounded-2xl border border-border bg-white p-5"><SkeletonRows rows={2} label="Memuat tagihan" /></div>
      )}
      {open && canFinance && finance === "error" && (
        <LoadError message="Tagihan Anda belum bisa dimuat. Periksa koneksi, lalu coba lagi." onRetry={retryFinance} />
      )}
      {open && canFinance && finance === "ready" && project.contractValue <= 0 && terms.length === 0 && (
        <section aria-labelledby="pay-none" className="rounded-2xl border border-border bg-white p-5 sm:p-6">
          <h2 id="pay-none" className="text-[15px] font-semibold text-text-primary">Tagihan berikutnya</h2>
          <p className="mt-2 text-[14px] text-text-secondary">Belum ada tagihan yang diatur. Wedding organizer Anda akan memberi tahu begitu ada.</p>
        </section>
      )}
      {open && canFinance && finance === "ready" && (project.contractValue > 0 || terms.length > 0) && (
        <NextPaymentCard
          projectName={project.name}
          clientName={clientName}
          term={nextTerm}
          remaining={remaining}
          contact={contact}
          contactPhone={pickContactPhone(project.picPhone, contact?.whatsapp)}
          contactFailed={contactFailed}
        />
      )}

      <div className="grid grid-cols-1 gap-4 sm:gap-5 md:grid-cols-2">
        <ProgressStages status={project.status} daysToEvent={days} percent={canTimeline ? project.progress?.overallPercent : undefined} />
        <div className="flex flex-col gap-4 sm:gap-5">
          {canTimeline && <RecentUpdates milestones={milestones} />}
          <ContactCard picName={project.picName} picPhone={pickContactPhone(project.picPhone, contact?.whatsapp)} contact={contact} clientName={clientName} projectName={project.name} />
        </div>
      </div>

      {canDocuments && (
        <Link
          to={ROUTE_PATHS.portal("dokumen")}
          className="flex items-center gap-3 rounded-2xl border border-border bg-white p-4 hover:bg-surface-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy-900 sm:p-5"
        >
          <FileText className="h-5 w-5 shrink-0 text-navy-900" aria-hidden />
          <span className="min-w-0 flex-1">
            <span className="block text-[14px] font-semibold text-text-primary">Dokumen penting</span>
            <span className="block text-[12.5px] text-text-secondary">Rundown, buku acara, dan lainnya</span>
          </span>
          <ChevronRight className="h-4 w-4 shrink-0 text-text-secondary" aria-hidden />
        </Link>
      )}
    </div>
  );
}
