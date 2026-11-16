import { Navigate, useLocation } from "react-router-dom";

/**
 * Sends an old address to its new one, keeping the rest of the path, the query and the hash (`/projects/7/vendor?x=1`
 * → `/proyek/7/vendor?x=1`). Old links in chats, bookmarks and e-mails keep working after the console's addresses
 * moved to Indonesian (RISET WO Console §3 #12).
 */
export function LegacyPathRedirect({ from, to }: { from: string; to: string }) {
  const { pathname, search, hash } = useLocation();
  const rest = pathname.toLowerCase().startsWith(from.toLowerCase()) ? pathname.slice(from.length) : "";
  return <Navigate to={`${to}${rest}${search}${hash}`} replace />;
}
