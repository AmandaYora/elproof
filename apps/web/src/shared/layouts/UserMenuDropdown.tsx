import type { ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { LogOut, MessageCircle } from "lucide-react";
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger,
} from "@/shared/components/kit/dropdown-menu";
import { logoutAndRedirect } from "@/shared/lib/auth-actions";
import { USER_BUTTON } from "@/shared/layouts/Sidebar";

interface Props {
  name: string;
  role: string;
  collapsed: boolean;
  face: ReactNode;
  helpUrl: string;
}

/** The user menu itself, loaded on the first click of the user button (Sidebar) and opened at once. */
export default function UserMenuDropdown({ name, role, collapsed, face, helpUrl }: Props) {
  const navigate = useNavigate();
  return (
    <DropdownMenu defaultOpen>
      <DropdownMenuTrigger aria-label={`Menu akun ${name}`} className={USER_BUTTON(collapsed)}>
        {face}
      </DropdownMenuTrigger>
      <DropdownMenuContent side="top" align="start" className="min-w-56">
        <DropdownMenuGroup>
          <DropdownMenuLabel className="text-[12.5px]">
            <span className="block truncate font-semibold text-foreground">{name}</span>
            <span className="block truncate font-normal text-muted-foreground">{role}</span>
          </DropdownMenuLabel>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => window.open(helpUrl, "_blank", "noopener,noreferrer")} className="text-[13px]">
          <MessageCircle aria-hidden /> Bantuan lewat WhatsApp
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => void logoutAndRedirect(navigate)} className="text-[13px]">
          <LogOut aria-hidden /> Keluar
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
