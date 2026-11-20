import { useNavigate } from "react-router-dom";
import { ChevronDown, Plus } from "lucide-react";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/shared/components/kit/dropdown-menu";
import { buttonVariants } from "@/shared/components/kit/button";
import { cn } from "@/shared/lib/cn";
import type { CreateItem } from "@/shared/layouts/Topbar";

interface Props {
  items: CreateItem[];
  triggerClassName?: string;
}

/** The "Baru" menu itself, loaded on the first click of the top bar's button (Topbar) and opened at once. */
export default function CreateMenuDropdown({ items, triggerClassName }: Props) {
  const navigate = useNavigate();
  return (
    <DropdownMenu defaultOpen>
      <DropdownMenuTrigger className={cn(buttonVariants(), triggerClassName)}>
        <Plus aria-hidden /> Baru <ChevronDown aria-hidden />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-44">
        {items.map((item) => (
          <DropdownMenuItem key={item.key} onClick={() => navigate(item.to)} className="text-[13px]">
            {item.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
