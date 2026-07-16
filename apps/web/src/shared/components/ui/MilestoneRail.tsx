import { cn } from "@/shared/lib/cn";

export type RailMilestoneStatus = "Not Started" | "In Progress" | "Completed" | "Blocked" | "Cancelled";

interface MilestoneRailProps {
  milestones: { status: RailMilestoneStatus }[];
  size?: "sm" | "md";
  showCount?: boolean;
  className?: string;
}

const STRIPE_STYLE = {
  backgroundImage: "repeating-linear-gradient(45deg, currentColor 0, currentColor 1.5px, transparent 1.5px, transparent 5px)",
};

function Tick({ status, size }: { status: RailMilestoneStatus; size: "sm" | "md" }) {
  const dims = size === "sm" ? "h-4 w-[5px]" : "h-5 w-[7px]";

  if (status === "Completed") {
    return <span className={cn(dims, "rounded-[2px] bg-navy-900")} title="Selesai" />;
  }
  if (status === "In Progress") {
    return (
      <span
        className={cn(dims, "rounded-[2px] border border-info text-info/70 bg-info/10")}
        style={STRIPE_STYLE}
        title="Sedang berjalan"
      />
    );
  }
  if (status === "Blocked") {
    return (
      <span
        className={cn(dims, "rounded-[2px] border border-danger text-danger/70 bg-danger-soft")}
        style={STRIPE_STYLE}
        title="Terhambat"
      />
    );
  }
  return <span className={cn(dims, "rounded-[2px] border border-border bg-white")} title="Belum dimulai" />;
}

export function MilestoneRail({ milestones, size = "md", showCount = true, className }: MilestoneRailProps) {
  const relevant = milestones.filter((m) => m.status !== "Cancelled");
  const completed = relevant.filter((m) => m.status === "Completed").length;
  const total = relevant.length;

  if (total === 0) {
    return <span className="text-[13px] text-text-secondary">Belum ada milestone</span>;
  }

  return (
    <div className={cn("flex items-center gap-2.5", className)}>
      <div className="flex items-center gap-[3px]">
        {relevant.map((m, idx) => (
          <Tick key={idx} status={m.status} size={size} />
        ))}
      </div>
      {showCount && (
        <span className="whitespace-nowrap text-[12px] font-medium tabular-nums text-text-secondary">
          {completed}/{total} selesai
        </span>
      )}
    </div>
  );
}

export function MilestoneRailLegend() {
  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-[12px] text-text-secondary">
      <span className="flex items-center gap-1.5">
        <span className="h-4 w-[5px] rounded-[2px] bg-navy-900" /> Selesai
      </span>
      <span className="flex items-center gap-1.5">
        <span className="h-4 w-[5px] rounded-[2px] border border-info bg-info/10" /> Sedang berjalan
      </span>
      <span className="flex items-center gap-1.5">
        <span className="h-4 w-[5px] rounded-[2px] border border-danger bg-danger-soft" /> Terhambat
      </span>
      <span className="flex items-center gap-1.5">
        <span className="h-4 w-[5px] rounded-[2px] border border-border bg-white" /> Belum dimulai
      </span>
    </div>
  );
}
