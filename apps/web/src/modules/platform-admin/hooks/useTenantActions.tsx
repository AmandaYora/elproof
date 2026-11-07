import { useEffect, useState, type ReactNode } from "react";
import { TenantFormModal } from "@/modules/platform-admin/components/TenantFormModal";
import { ActivateSubscriptionModal } from "@/modules/platform-admin/components/ActivateSubscriptionModal";
import { AccessLinkDialog } from "@/shared/components/AccessLinkDialog";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import type { AccessLink } from "@/shared/lib/access-link";
import type { Tenant } from "@/modules/platform-admin/data/types";
import { usePlatformAdminStore } from "@/modules/platform-admin/stores/usePlatformAdminStore";
import { useSubscriptionPlanStore } from "@/shared/stores/useSubscriptionPlanStore";
import type { TenantFormValues } from "@/modules/platform-admin/schemas/tenant.schema";
import type { ActivateSubscriptionFormValues } from "@/modules/platform-admin/schemas/activate-subscription.schema";
import { getApiErrorMessage } from "@/shared/lib/api-error";

export interface OwnerLinkShare {
  tenant: Tenant;
  link: AccessLink;
  /** Google account that must accept the invitation. */
  email: string;
  notice: string;
}

/**
 * The actions on one tenant (edit, activate subscription, freeze, re-invite
 * the Owner) with their dialogs, shared by the tenant list and the Tenant
 * 360° page so the two behave identically. `onChanged` runs after each
 * successful change so the caller can reload what it shows.
 */
export function useTenantActions(onChanged: () => Promise<void> | void) {
  const plans = useSubscriptionPlanStore((s) => s.plans);
  const fetchPlans = useSubscriptionPlanStore((s) => s.fetchPlans);
  const updateTenant = usePlatformAdminStore((s) => s.updateTenant);
  const uploadTenantLogo = usePlatformAdminStore((s) => s.uploadTenantLogo);
  const toggleTenantSuspension = usePlatformAdminStore((s) => s.toggleTenantSuspension);
  const reinviteTenantOwner = usePlatformAdminStore((s) => s.reinviteTenantOwner);
  const activateTenantSubscription = usePlatformAdminStore((s) => s.activateTenantSubscription);

  const [editing, setEditing] = useState<Tenant | undefined>(undefined);
  const [activating, setActivating] = useState<Tenant | undefined>(undefined);
  const [isActivating, setIsActivating] = useState(false);
  const [activationError, setActivationError] = useState<string | null>(null);
  const [suspension, setSuspension] = useState<Tenant | undefined>(undefined);
  const [suspensionError, setSuspensionError] = useState<string | null>(null);
  const [isTogglingSuspension, setIsTogglingSuspension] = useState(false);
  const [ownerLink, setOwnerLink] = useState<OwnerLinkShare | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchPlans().catch(() => undefined);
  }, [fetchPlans]);

  async function handleEditSubmit(values: TenantFormValues) {
    if (!editing) return;
    setActionError(null);
    try {
      await updateTenant(editing.id, values);
      setEditing(undefined);
      await onChanged();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal memperbarui tenant"));
    }
  }

  async function reinvite(tenant: Tenant) {
    setActionError(null);
    try {
      const { link, email } = await reinviteTenantOwner(tenant.id);
      setOwnerLink({
        tenant,
        link,
        email,
        notice: "Link lama tidak berlaku lagi. Setelah link ini diterima, sesi Owner yang masih terbuka berakhir.",
      });
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal membuat undangan Owner"));
    }
  }

  async function handleActivateSubmit(values: ActivateSubscriptionFormValues) {
    if (!activating) return;
    setActivationError(null);
    setIsActivating(true);
    try {
      await activateTenantSubscription(activating.id, values);
      await onChanged();
      setActivating(undefined);
    } catch (err) {
      // Keep the dialog open with what was typed, so the admin can fix it.
      setActivationError(getApiErrorMessage(err, "Gagal mengaktifkan langganan tenant"));
    } finally {
      setIsActivating(false);
    }
  }

  async function handleConfirmSuspension() {
    if (!suspension) return;
    setSuspensionError(null);
    setIsTogglingSuspension(true);
    try {
      await toggleTenantSuspension(suspension.id);
      await onChanged();
      setSuspension(undefined);
    } catch (err) {
      setSuspensionError(getApiErrorMessage(err, "Gagal mengubah status tenant"));
    } finally {
      setIsTogglingSuspension(false);
    }
  }

  const dialogs: ReactNode = (
    <>
      <TenantFormModal
        key={editing?.id ?? "edit-none"}
        open={editing !== undefined}
        onClose={() => setEditing(undefined)}
        initialTenant={editing}
        onSubmitCreate={() => undefined}
        onSubmitEdit={handleEditSubmit}
        onUploadLogo={async (id, file) => {
          await uploadTenantLogo(id, file);
          await onChanged();
        }}
      />

      <AccessLinkDialog
        open={ownerLink !== null}
        onClose={() => setOwnerLink(null)}
        link={ownerLink?.link ?? null}
        recipient={
          ownerLink
            ? { name: ownerLink.tenant.ownerName, phone: ownerLink.tenant.phone, email: ownerLink.email, roleLabel: "Owner" }
            : { name: "" }
        }
        woName={ownerLink?.tenant.businessName ?? ""}
        notice={ownerLink?.notice}
      />

      <ActivateSubscriptionModal
        key={activating?.id ?? "activate-none"}
        open={activating !== undefined}
        onClose={() => setActivating(undefined)}
        tenant={activating}
        plans={plans}
        isSubmitting={isActivating}
        error={activationError}
        onSubmit={handleActivateSubmit}
      />

      <ConfirmDialog
        open={suspension !== undefined}
        title={suspension?.isSuspended ? "Aktifkan kembali tenant?" : "Bekukan tenant?"}
        confirmLabel={suspension?.isSuspended ? "Aktifkan kembali" : "Bekukan tenant"}
        tone={suspension?.isSuspended ? "primary" : "danger"}
        confirmText={suspension?.isSuspended ? undefined : suspension?.businessName}
        isSubmitting={isTogglingSuspension}
        error={suspensionError}
        onConfirm={() => void handleConfirmSuspension()}
        onClose={() => setSuspension(undefined)}
      >
        {suspension?.isSuspended ? (
          <p>Staf dan klien {suspension.businessName} bisa masuk dan memakai ElProof lagi, sesuai status langganannya.</p>
        ) : (
          <p>
            Semua staf dan klien <span className="font-semibold text-text-primary">{suspension?.businessName}</span> langsung keluar dan tidak bisa masuk sampai
            tenant diaktifkan kembali. Data tetap tersimpan. Gunakan hanya untuk pelanggaran atau penyalahgunaan; tunggakan langganan sudah ditangani otomatis.
          </p>
        )}
      </ConfirmDialog>
    </>
  );

  return {
    plans,
    actionError,
    clearActionError: () => setActionError(null),
    setActionError,
    showOwnerLink: setOwnerLink,
    openEdit: (tenant: Tenant) => {
      setActionError(null);
      setEditing(tenant);
    },
    openActivation: (tenant: Tenant) => {
      setActivationError(null);
      setActivating(tenant);
    },
    openSuspension: (tenant: Tenant) => {
      setSuspensionError(null);
      setSuspension(tenant);
    },
    reinvite,
    dialogs,
  };
}
