'use client';

import { useEffect } from 'react';
import { useParams, notFound } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { getProblem } from '@/lib/api/problem';
import { useUiStore } from '@/stores/ui-store';

/**
 * Every org-scoped route hangs off this segment, so this is where "which
 * organization?" gets answered once instead of being assumed by each page.
 *
 * Before this existed, /organizations/99999/employees rendered a working
 * Employees page: the heading, the toolbar, and an empty state inviting the user
 * to add their first employee -- with a *different* organization's name in the
 * selector, because the sidebar reads the selected org from the store while the
 * page reads the id from the URL. Nothing on screen said the org was wrong.
 *
 * Only a definitive 404 turns into a not-found page. A timeout, an offline
 * laptop or a 500 leaves the children to render and report the failure
 * themselves -- telling somebody their organization does not exist because their
 * wifi dropped would be a worse bug than the one this fixes.
 *
 * Children render while the check is in flight, so a valid organization -- which
 * is all of them, nearly all of the time -- pays nothing for this.
 *
 * It is also where the store learns which organization the URL is talking
 * about. `syncFromRoute` existed for this and was never called from anywhere,
 * so on any deep link -- a bookmark, a shared URL, the browser restoring tabs --
 * the page rendered organization 7 from the path while the store still held
 * organization 3 from localStorage. The consequences were not cosmetic:
 *
 *   - Every sidebar link pointed at organization 3, so the next click silently
 *     changed organization.
 *   - The selector showed organization 3's name above organization 7's data.
 *   - `useCurrentRole()` resolved organization *3's* role and drove all the
 *     `hasMinimumRole` gating with it. An admin in 3 who is only staff in 7 was
 *     offered admin controls that the backend then refused; the reverse locked
 *     a user out of pages they were entitled to.
 *
 * This segment is the one place every org-scoped route passes through, which is
 * what makes it the right place to answer the question once.
 */
export default function OrganizationScopedLayout({ children }: { children: React.ReactNode }) {
  const params = useParams();
  const raw = params.orgId;
  const orgId = Number(raw);
  // /organizations/abc/employees never had an organization to look up.
  const addressable = typeof raw === 'string' && Number.isInteger(orgId) && orgId > 0;

  const { error } = useQuery({
    queryKey: queryKeys.organizations.detail(orgId),
    queryFn: () => apiClient.getOrganization(orgId),
    // Asking the server about "abc" would only earn a 400. The hook still runs
    // on every render -- rules of hooks -- it just has nothing to do.
    enabled: addressable,
    // The backend answers 404 both for an organization that does not exist and
    // for one the caller cannot see, which is the same thing from here.
    retry: false,
    staleTime: 5 * 60 * 1000,
  });

  const syncFromRoute = useUiStore((state) => state.syncFromRoute);
  useEffect(() => {
    // Only for an addressable id, and only forward: this teaches the store what
    // the URL says, never the other way round. A non-numeric segment is about to
    // become a 404 and must not clear a good selection on the way.
    if (addressable) {
      syncFromRoute(orgId);
    }
  }, [addressable, orgId, syncFromRoute]);

  if (!addressable || getProblem(error)?.status === 404) {
    notFound();
  }

  return <>{children}</>;
}
