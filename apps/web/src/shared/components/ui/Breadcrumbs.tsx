import { Fragment } from "react";
import { Link } from "react-router-dom";
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from "@/shared/components/kit/breadcrumb";

export interface Crumb {
  label: string;
  /** Omitted on the last crumb: the page the person is on. */
  to?: string;
}

/** "Proyek > Citra & Bima": where this page sits, each ancestor a link. The current page is the last, not a link. */
export function Breadcrumbs({ items }: { items: Crumb[] }) {
  return (
    <Breadcrumb aria-label="Jejak halaman">
      <BreadcrumbList>
        {items.map((crumb, i) => {
          const last = i === items.length - 1;
          return (
            <Fragment key={`${crumb.label}-${i}`}>
              <BreadcrumbItem>
                {last || !crumb.to ? <BreadcrumbPage>{crumb.label}</BreadcrumbPage> : <BreadcrumbLink render={<Link to={crumb.to} />}>{crumb.label}</BreadcrumbLink>}
              </BreadcrumbItem>
              {!last && <BreadcrumbSeparator />}
            </Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
