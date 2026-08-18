'use client';

import { useParams, notFound } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { getProblem } from '@/lib/api/problem';

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

  if (!addressable || getProblem(error)?.status === 404) {
    notFound();
  }

  return <>{children}</>;
}
