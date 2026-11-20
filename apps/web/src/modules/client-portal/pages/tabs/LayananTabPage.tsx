import { useOutletContext } from "react-router-dom";
import { AlertTriangle, ListChecks, Users } from "lucide-react";
import { EngagementStatusBadge } from "@/modules/projects/components/StatusBadges";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { useCan } from "@/shared/stores/usePermissionStore";
import { IssueCard } from "@/modules/client-portal/components/IssueCard";
import { MilestoneRail } from "@/modules/client-portal/components/MilestoneRail";
import { useProofViewer, type ProofToOpen } from "@/modules/client-portal/components/useProofViewer";
import { useLoad } from "@/modules/client-portal/lib/useLoad";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import type { ProjectSupplier, VendorIssue, VendorMilestone } from "@/modules/projects/types";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";
import { PageIntro } from "@/modules/client-portal/components/PageIntro";
import { usePortalContact, type PortalContact } from "@/modules/client-portal/lib/usePortalContact";
import { PortalRincian } from "@/modules/rincian";

const isOpenIssue = (i: VendorIssue) => i.status !== "Resolved" && i.status !== "Closed";

/**
 * Layanan (DESAIN §3.3, §6.11): what the WO prepares for the couple. First "Paket & rincian" (the packages with their
 * isi, the items, bonus and potongan; amounts and the contract value only for a role with portal.finance.view, which
 * the API decides by leaving them out), then "Progres vendor": each vendor and venue with its steps, proof and kendala.
 * The API already names a vendor the couple may not know by its service
 * (GET /projects/{id}/suppliers, redacted for a client), so this page shows the name it gets. Each event's venue
 * address is on Jadwal.
 */
export default function LayananTabPage() {
  const { projectId } = useOutletContext<ClientPortalContext>();
  const suppliers = useProjectStore((s) => s.suppliers);
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const allIssues = useProjectStore((s) => s.issues);
  const fetchSuppliers = useProjectStore((s) => s.fetchSuppliers);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);
  const canIssues = useCan("portal.issues.view");
  const viewer = useProofViewer(projectId);
  const contact = usePortalContact();

  const { status, retry } = useLoad(
    () => Promise.all([fetchSuppliers(projectId), canIssues ? fetchIssues(projectId) : undefined]).then(() => undefined),
    [projectId, fetchSuppliers, fetchIssues, canIssues],
  );
  const issues = canIssues ? allIssues : [];

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6">
      <PageIntro title="Layanan">Paket yang Anda pilih, lalu vendor dan venue yang menyiapkan hari bahagia Anda beserta tahapan kerja yang sudah selesai dan buktinya.</PageIntro>

      <RincianSection projectId={projectId} />

      {status === "loading" && (
        <div className="rounded-2xl border border-border bg-card p-5">
          <SkeletonRows rows={4} label="Memuat layanan" />
        </div>
      )}
      {status === "error" && <LoadError message="Layanan belum bisa dimuat. Periksa koneksi, lalu coba lagi." onRetry={retry} />}

      {status === "ready" && suppliers.length === 0 && (
        <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-border bg-card p-10 text-center">
          <Users className="size-8 text-muted-foreground" aria-hidden />
          <p className="text-[14px] text-muted-foreground">Vendor dan venue untuk pernikahan Anda belum ditentukan.</p>
        </div>
      )}

      {status === "ready" && suppliers.length > 0 && (
        <section aria-labelledby="supplier-progress" className="flex flex-col gap-3">
          <div className="flex items-baseline justify-between gap-3">
            <h2 id="supplier-progress" className="text-base font-semibold text-foreground">Progres vendor</h2>
            <span className="text-[13px] text-muted-foreground">{suppliers.length} vendor dan venue</span>
          </div>
          {suppliers.map((sp) => (
            <SupplierCard
              key={sp.id}
              supplier={sp}
              steps={vendorMilestones.filter((m) => m.projectSupplierId === sp.id && m.status !== "Cancelled")}
              issues={issues.filter((i) => i.projectSupplierId === sp.id)}
              onOpenProof={viewer.open}
              contact={contact}
            />
          ))}
        </section>
      )}
      {viewer.element}
    </div>
  );
}

/** "Paket & rincian": loaded on its own, so a failure here leaves the vendor progress below readable. */
function RincianSection({ projectId }: { projectId: string }) {
  const rincian = useProjectStore((s) => (s.rincianFor === projectId ? s.rincian : null));
  const fetchRincian = useProjectStore((s) => s.fetchRincian);
  const { status, retry } = useLoad(() => fetchRincian(projectId), [projectId, fetchRincian]);

  return (
    <section aria-labelledby="paket-rincian" className="flex flex-col gap-3">
      <h2 id="paket-rincian" className="text-base font-semibold text-foreground">
        Paket & rincian
      </h2>
      {status === "loading" && (
        <div className="rounded-2xl border border-border bg-card p-5">
          <SkeletonRows rows={3} label="Memuat rincian" />
        </div>
      )}
      {status === "error" && <LoadError message="Rincian belum bisa dimuat. Periksa koneksi, lalu coba lagi." onRetry={retry} />}
      {status === "ready" && rincian && rincian.lines.length === 0 && (
        <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-border bg-card p-8 text-center">
          <ListChecks className="size-8 text-muted-foreground" aria-hidden />
          <p className="text-[14px] text-muted-foreground">Rincian belum disusun wedding organizer Anda.</p>
        </div>
      )}
      {status === "ready" && rincian && rincian.lines.length > 0 && (
        <div className="rounded-2xl border border-border bg-card p-5">
          <PortalRincian rincian={rincian} />
        </div>
      )}
    </section>
  );
}

function SupplierCard({
  supplier,
  steps,
  issues,
  onOpenProof,
  contact,
}: {
  supplier: ProjectSupplier;
  steps: VendorMilestone[];
  issues: VendorIssue[];
  onOpenProof: (proof: ProofToOpen, title: string) => void;
  contact: PortalContact;
}) {
  const doneCount = steps.filter((m) => m.status === "Completed").length;
  const openCount = issues.filter(isOpenIssue).length;
  const ordered = [...steps].sort((a, b) => a.order - b.order);
  // A hidden vendor arrives named by its service: the service line would then only repeat it.
  const service = supplier.categoryName && supplier.categoryName !== supplier.name ? supplier.categoryName : "";
  const fallbackName = supplier.kind === "venue" ? "Venue" : "Vendor";

  return (
    <article className="overflow-hidden rounded-2xl border border-border bg-card">
      <div className="flex flex-wrap items-start justify-between gap-3 border-b p-5">
        <div className="min-w-0">
          <h3 className="text-base font-semibold text-foreground">{supplier.name || fallbackName}</h3>
          {(service || supplier.scope) && <p className="mt-0.5 text-[13px] text-muted-foreground">{[service, supplier.scope].filter(Boolean).join(" · ")}</p>}
        </div>
        <EngagementStatusBadge status={supplier.engagementStatus} audience="client" />
      </div>

      <div className="flex flex-col gap-4 p-5">
        {openCount > 0 && (
          <p className="flex items-center gap-2 rounded-xl bg-warning-soft px-3.5 py-2.5 text-[13px] font-medium text-warning-text">
            <AlertTriangle className="size-4 shrink-0" aria-hidden /> {openCount} kendala sedang kami tangani
          </p>
        )}

        {ordered.length === 0 ? (
          <p className="text-[14px] text-muted-foreground">Belum ada tahapan yang tercatat untuk {supplier.kind === "venue" ? "venue" : "vendor"} ini.</p>
        ) : (
          <details className="group">
            <summary className="flex min-h-11 cursor-pointer items-center justify-between rounded-lg text-[14px] font-semibold text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground">
              <span>
                {doneCount} dari {ordered.length} tahapan selesai
              </span>
              <span className="text-[13px] font-medium text-muted-foreground group-open:hidden">Lihat tahapan</span>
              <span className="hidden text-[13px] font-medium text-muted-foreground group-open:inline">Sembunyikan</span>
            </summary>
            <div className="mt-3">
              <MilestoneRail steps={ordered} onOpenProof={onOpenProof} />
            </div>
          </details>
        )}

        {issues.length > 0 && (
          <div className="flex flex-col gap-3">
            {[...issues]
              .sort((a, b) => Number(isOpenIssue(b)) - Number(isOpenIssue(a)) || b.foundDate.localeCompare(a.foundDate))
              .map((i) => (
                <IssueCard key={i.id} issue={i} showVendorName={false} onOpenProof={onOpenProof} contact={contact} />
              ))}
          </div>
        )}
      </div>
    </article>
  );
}
