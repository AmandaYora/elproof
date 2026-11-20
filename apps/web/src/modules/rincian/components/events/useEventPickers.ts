import { useCallback, useEffect, useMemo, useState } from "react";
import { useEventKindStore } from "@/modules/event-kinds/stores/useEventKindStore";
import { useVenueStore } from "@/modules/venues/stores/useVenueStore";
import type { VenueCaps } from "@/modules/rincian/components/events/event-display";

/**
 * What an event form picks from: the workspace's event kinds and the active venues of the directory with their
 * capacities. Loaded when a form opens, not with the page.
 *
 * The venues come as the directory's whole active list (`all=true`): the paged search of GET /venues needs
 * `vendor.view`, which a Wedding Planner editing their own project does not have. The picker narrows that list as the
 * person types.
 */
export function useEventPickers() {
  const kinds = useEventKindStore((s) => s.kinds);
  const fetchKinds = useEventKindStore((s) => s.fetchKinds);
  const directory = useVenueStore((s) => s.venues);
  const fetchVenues = useVenueStore((s) => s.fetchVenues);
  const [kindStatus, setKindStatus] = useState<"loading" | "ready" | "error">("loading");
  const [venueStatus, setVenueStatus] = useState<"loading" | "ready" | "error">("loading");

  const loadKinds = useCallback(() => {
    setKindStatus("loading");
    fetchKinds().then(
      () => setKindStatus("ready"),
      () => setKindStatus("error"),
    );
  }, [fetchKinds]);

  const loadVenues = useCallback(() => {
    setVenueStatus("loading");
    fetchVenues().then(
      () => setVenueStatus("ready"),
      () => setVenueStatus("error"),
    );
  }, [fetchVenues]);

  useEffect(() => {
    loadKinds();
    loadVenues();
  }, [loadKinds, loadVenues]);

  const venues = useMemo<VenueCaps[]>(
    () =>
      directory
        .filter((v) => v.isActive)
        .map((v) => ({ venueId: v.id, name: v.name, city: v.city, capacityStanding: v.capacityStanding, capacitySeated: v.capacitySeated })),
    [directory],
  );

  return { kinds, kindStatus, retryKinds: loadKinds, venues, venueStatus, retryVenues: loadVenues };
}
