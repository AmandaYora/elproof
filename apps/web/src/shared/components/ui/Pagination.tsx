import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { cn } from "@/shared/lib/cn";

interface PaginationProps {
  page: number;
  totalPages: number;
  totalItems: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  className?: string;
}

export function Pagination({ page, totalPages, totalItems, pageSize, onPageChange, className }: PaginationProps) {
  if (totalItems === 0 || totalPages <= 1) return null;

  const start = (page - 1) * pageSize + 1;
  const end = Math.min(page * pageSize, totalItems);

  return (
    <div className={cn("flex flex-wrap items-center justify-between gap-3 border-t border-border-light px-5 py-3", className)}>
      <span className="text-[12.5px] text-text-secondary">
        Menampilkan <span className="font-medium text-text-primary">{start}–{end}</span> dari{" "}
        <span className="font-medium text-text-primary">{totalItems}</span>
      </span>
      <div className="flex items-center gap-3">
        <Button
          variant="secondary"
          size="sm"
          disabled={page === 1}
          onClick={() => onPageChange(page - 1)}
          icon={<ChevronLeft className="h-3.5 w-3.5" />}
        >
          Sebelumnya
        </Button>
        <span className="whitespace-nowrap text-[12.5px] font-medium text-text-secondary">
          Halaman {page} dari {totalPages}
        </span>
        <Button variant="secondary" size="sm" disabled={page === totalPages} onClick={() => onPageChange(page + 1)}>
          Berikutnya <ChevronRight className="ml-1.5 h-3.5 w-3.5" />
        </Button>
      </div>
    </div>
  );
}
