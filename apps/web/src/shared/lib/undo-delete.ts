import { toast } from "@/shared/stores/useToastStore";
import { getApiErrorMessage } from "@/shared/lib/api-error";

/**
 * Deletes something routine with no "Yakin?" step: it disappears at once and the toast offers "Urungkan" for a few
 * seconds (the server keeps it for 24 hours, so undo is a real restore). `del` returns the function that restores it.
 * Returns whether the delete went through.
 */
export async function deleteWithUndo(what: string, del: () => Promise<() => Promise<void>>, failed = `${what} belum bisa dihapus`): Promise<boolean> {
  try {
    const restore = await del();
    toast.success(`${what} dihapus`, {
      undo: () => {
        void restore()
          .then(() => toast.success(`${what} dipulihkan`))
          .catch((err) => toast.error(getApiErrorMessage(err, `${what} belum bisa dipulihkan`)));
      },
    });
    return true;
  } catch (err) {
    toast.error(getApiErrorMessage(err, failed));
    return false;
  }
}
