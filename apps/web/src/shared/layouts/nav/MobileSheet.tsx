import type { ReactNode } from "react";
import { Sheet, SheetContent, SheetDescription, SheetTitle } from "@/shared/components/kit/sheet";
import { Menu } from "@/shared/layouts/nav/NavMenu";
import type { NavSection } from "@/shared/layouts/nav/types";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  brand: ReactNode;
  sections: NavSection[];
  footer: ReactNode;
  aboveFooter?: ReactNode;
}

/** The menu as a sheet on a phone. Loaded the first time it is opened: see AppSidebar. */
export default function MobileSheet({ open, onOpenChange, brand, sections, footer, aboveFooter }: Props) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="left" showCloseButton className="w-72 gap-0 bg-sidebar p-0 text-sidebar-foreground data-[side=left]:sm:max-w-72">
        <SheetTitle className="sr-only">Menu</SheetTitle>
        <SheetDescription className="sr-only">Pindah ke halaman lain di aplikasi.</SheetDescription>
        <div className="flex min-h-16 items-center px-5 pr-14">{brand}</div>
        <Menu sections={sections} rail={false} />
        {aboveFooter}
        <div className="border-t px-2.5 py-2">{footer}</div>
      </SheetContent>
    </Sheet>
  );
}
