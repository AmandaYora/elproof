import { Drawer } from "@/shared/components/ui/Drawer";
import { CalendarRow } from "@/modules/calendar/components/CalendarEntry";
import { longDate } from "@/modules/calendar/lib/dates";
import type { CalendarItem } from "@/modules/calendar/types";

interface DayDrawerProps {
  day: string | null;
  items: CalendarItem[];
  today: string;
  onClose: () => void;
}

/** Every entry of one day, as links (opened by "+N lainnya", a day number, or Enter on a day). Loaded on demand. */
export default function DayDrawer({ day, items, today, onClose }: DayDrawerProps) {
  return (
    <Drawer
      open={day !== null}
      onClose={onClose}
      title={day ? longDate(day) : ""}
      description={items.length === 0 ? "Tidak ada agenda di hari ini." : `${items.length} agenda`}
    >
      {items.length > 0 && (
        <ul className="-mx-2 flex flex-col">
          {items.map((item, i) => (
            <li key={`${item.kind}-${item.projectId}-${i}`}>
              <CalendarRow item={item} today={today} />
            </li>
          ))}
        </ul>
      )}
    </Drawer>
  );
}
