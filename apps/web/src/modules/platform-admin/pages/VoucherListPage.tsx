import { useEffect, useState } from "react";
import { Ban, CheckCircle2, ListChecks, Pencil, Plus } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
import { Button } from "@/shared/components/ui/Button";
import { Select } from "@/shared/components/ui/Input";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { ListToolbar } from "@/shared/data-table/ListToolbar";
import { useListState } from "@/shared/data-table/useListState";
import { useListDensity } from "@/shared/data-table/useListDensity";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { VOUCHER_STATE_LABELS } from "@/shared/lib/labels";
import { toast } from "@/shared/stores/useToastStore";
import { appLabel, ELPROOF_APP_ID } from "@/shared/data/subscriptionPlans";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";
import { VOUCHER_STATE_TONE } from "@/modules/platform-admin/lib/status";
import { discountLabel, usageLabel, validityLabel } from "@/modules/platform-admin/lib/vouchers";
import { useAppsStore } from "@/modules/platform-admin/stores/useAppsStore";
import { useVoucherStore, type Voucher, type VoucherBody } from "@/modules/platform-admin/stores/useVoucherStore";
import { VoucherFormModal } from "@/modules/platform-admin/components/VoucherFormModal";
import { VoucherRedemptionsModal } from "@/modules/platform-admin/components/VoucherRedemptionsModal";

const LIST_FILTERS = { appId: "" };

export default function VoucherListPage() {
  const vouchers = useVoucherStore((s) => s.vouchers);
  const meta = useVoucherStore((s) => s.meta);
  const fetchVouchers = useVoucherStore((s) => s.fetchVouchers);
  const createVoucher = useVoucherStore((s) => s.createVoucher);
  const updateVoucher = useVoucherStore((s) => s.updateVoucher);
  const setVoucherActive = useVoucherStore((s) => s.setVoucherActive);
  const apps = useAppsStore((s) => s.apps);
  const fetchApps = useAppsStore((s) => s.fetchApps);
  // Creating, editing and (de)activating vouchers is Billing's job, like plans; every other role only looks.
  const canWrite = usePlatformCan("plans");

  const list = useListState({ filters: LIST_FILTERS });
  const { density, setDensity } = useListDensity();
  const [loadState, setLoadState] = useState<"loading" | "error" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Voucher | undefined>(undefined);
  const [usageOf, setUsageOf] = useState<Voucher | null>(null);
  // The voucher stays set while the dialog animates closed, so its title does not blank out.
  const [deactivating, setDeactivating] = useState<Voucher | null>(null);
  const [deactivateOpen, setDeactivateOpen] = useState(false);
  const [deactivateError, setDeactivateError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<number | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchApps().catch(() => undefined);
  }, [fetchApps]);

  useEffect(() => {
    let cancelled = false;
    setLoadState("loading");
    fetchVouchers(list.params)
      .then(() => !cancelled && setLoadState("ready"))
      .catch(() => !cancelled && setLoadState("error"));
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchVouchers, list.paramsKey, attempt]);

  function nameOf(appId: string) {
    return appLabel(apps.find((a) => a.appId === appId), appId);
  }

  function openCreate() {
    setEditing(undefined);
    setFormOpen(true);
  }

  function openEdit(v: Voucher) {
    setEditing(v);
    setFormOpen(true);
  }

  function closeForm() {
    setFormOpen(false);
  }

  // Errors propagate to the form: a 422 lands on its fields, a 409 is answered there with the latest data.
  async function handleSubmit(body: VoucherBody) {
    setActionError(null);
    if (editing) {
      await updateVoucher(editing.id, body, editing.updatedAt);
      toast.success(`Voucher ${body.code} diperbarui`);
    } else {
      await createVoucher(body);
      toast.success(`Voucher ${body.code} dibuat`);
    }
    setFormOpen(false);
  }

  async function activate(v: Voucher) {
    setActionError(null);
    setBusyId(v.id);
    try {
      await setVoucherActive(v.id, true);
      toast.success(`Voucher ${v.code} diaktifkan`);
    } catch (err) {
      setActionError(getApiErrorMessage(err, `Voucher ${v.code} belum bisa diaktifkan. Coba lagi.`));
    } finally {
      setBusyId(null);
    }
  }

  async function confirmDeactivate() {
    if (!deactivating) return;
    setDeactivateError(null);
    setBusyId(deactivating.id);
    try {
      await setVoucherActive(deactivating.id, false);
      toast.success(`Voucher ${deactivating.code} dinonaktifkan`);
      setDeactivateOpen(false);
    } catch (err) {
      setDeactivateError(getApiErrorMessage(err, "Voucher belum bisa dinonaktifkan. Coba lagi."));
    } finally {
      setBusyId(null);
    }
  }

  const filterOptions = [ELPROOF_APP_ID, ...apps.map((a) => a.appId).filter((id) => id !== ELPROOF_APP_ID)];
  if (list.filters.appId && !filterOptions.includes(list.filters.appId)) filterOptions.push(list.filters.appId);

  const columns: Column<Voucher>[] = [
    { id: "code", header: "Kode", primary: true, cell: (v) => <span className="font-mono text-[13px] font-semibold">{v.code}</span> },
    { id: "app", header: "Aplikasi", className: "min-w-24", cell: (v) => nameOf(v.appId) },
    { id: "plan", header: "Paket", cell: (v) => (v.planId === null ? <span className="text-muted-foreground">Semua paket</span> : v.planName ?? `Paket #${v.planId}`) },
    { id: "discount", header: "Potongan", className: "min-w-24", cell: (v) => <span className="tabular-nums">{discountLabel(v)}</span> },
    { id: "validity", header: "Masa berlaku", className: "min-w-28", cell: (v) => validityLabel(v) },
    { id: "usage", header: "Pemakaian", className: "min-w-28", cell: (v) => <span className="tabular-nums">{usageLabel(v)}</span> },
    { id: "state", header: "Status", className: "whitespace-nowrap", cell: (v) => <Badge tone={VOUCHER_STATE_TONE[v.state]}>{VOUCHER_STATE_LABELS[v.state]}</Badge> },
  ];

  function rowActions(v: Voucher) {
    return (
      <div className="flex items-center gap-1.5">
        <IconActionButton icon={ListChecks} label={`Pemakaian ${v.code}`} tone="info" readOnly onClick={() => setUsageOf(v)} />
        {canWrite && (
          <>
            <IconActionButton icon={Pencil} label={`Ubah ${v.code}`} onClick={() => openEdit(v)} />
            {v.isActive ? (
              <IconActionButton
                icon={Ban}
                label={`Nonaktifkan ${v.code}`}
                tone="danger"
                disabled={busyId === v.id}
                onClick={() => {
                  setDeactivateError(null);
                  setDeactivating(v);
                  setDeactivateOpen(true);
                }}
              />
            ) : (
              <IconActionButton icon={CheckCircle2} label={`Aktifkan ${v.code}`} tone="success" disabled={busyId === v.id} onClick={() => void activate(v)} />
            )}
          </>
        )}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Voucher</h1>
          <p className="mt-1 max-w-2xl text-[13px] text-text-secondary">
            Kode potongan harga untuk paket satu aplikasi. Voucher ElProof dipakai tenant WO di halaman Langganan; voucher aplikasi lain dipakai lewat API
            aplikasi itu. Harga akhir tidak pernah Rp0.
          </p>
        </div>
        {canWrite && (
          <Button icon={<Plus className="h-4 w-4" />} onClick={openCreate}>
            Tambah voucher
          </Button>
        )}
      </div>

      {actionError && (
        <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
          {actionError}
        </p>
      )}

      <ListToolbar
        search={{ value: list.q, onChange: list.setQ, placeholder: "Cari kode voucher..." }}
        density={{ value: density, onChange: setDensity }}
      >
        <Select className="w-60" aria-label="Saring voucher per aplikasi" value={list.filters.appId} onChange={(e) => list.setFilter("appId", e.target.value)}>
          <option value="">Semua aplikasi</option>
          {filterOptions.map((appId) => (
            <option key={appId} value={appId}>
              {nameOf(appId)}
            </option>
          ))}
        </Select>
      </ListToolbar>

      <DataTable
        label="Daftar voucher"
        columns={columns}
        rows={vouchers}
        rowId={(v) => String(v.id)}
        status={loadState}
        onRetry={() => setAttempt((n) => n + 1)}
        meta={meta}
        onPageChange={list.setPage}
        sort={null}
        onSortChange={() => undefined}
        filtered={list.isFiltered}
        onResetFilters={list.reset}
        empty={{
          title: "Belum ada voucher",
          description: canWrite
            ? "Buat voucher pertama untuk memberi potongan harga paket kepada pelanggan."
            : "Voucher yang dibuat tim Billing akan tampil di sini.",
          action: canWrite ? <Button onClick={openCreate}>Tambah voucher</Button> : undefined,
        }}
        density={density}
        actions={rowActions}
      />

      {canWrite && (
        <VoucherFormModal
          open={formOpen}
          onClose={closeForm}
          onSubmit={handleSubmit}
          initialVoucher={editing}
          onStale={(latest) => {
            setEditing(latest);
            void fetchVouchers(list.params).catch(() => undefined);
          }}
          apps={apps}
          defaultAppId={list.filters.appId}
        />
      )}

      <VoucherRedemptionsModal voucher={usageOf} onClose={() => setUsageOf(null)} apps={apps} />

      <ConfirmDialog
        open={deactivateOpen}
        title={`Nonaktifkan voucher ${deactivating?.code ?? ""}?`}
        confirmLabel="Nonaktifkan voucher"
        isSubmitting={busyId !== null && busyId === deactivating?.id}
        error={deactivateError}
        onConfirm={() => void confirmDeactivate()}
        onClose={() => setDeactivateOpen(false)}
      >
        Kode ini tidak bisa dipakai lagi untuk pembayaran baru. Pembayaran yang sudah memakainya tidak berubah, dan voucher bisa diaktifkan kembali kapan
        saja.
      </ConfirmDialog>
    </div>
  );
}
