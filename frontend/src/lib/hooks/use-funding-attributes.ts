import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { LOOKUP_FETCH_LIMIT } from '@/lib/api/types';
import { useUiStore } from '@/stores/ui-store';

export interface FundingAttribute {
  key: string;
  value: string;
  label: string;
}

/**
 * Hook to get suggested attributes for a child contract based on the
 * organization's government funding configuration.
 *
 * Returns property key/value pairs from funding periods that overlap with
 * the given date range. Properties with the same key are mutually exclusive
 * (e.g., care_type can only be one of: ganztag, halbtag, teilzeit).
 */
export function useFundingAttributes(orgId: number, fromDate?: string, toDate?: string) {
  const organizations = useUiStore((state) => state.organizations);
  const org = organizations.find((o) => o.id === orgId);
  const state = org?.state;

  // Fetch all government fundings (cached by React Query)
  const { data: fundingsResponse } = useQuery({
    queryKey: queryKeys.governmentFundings.allCached(),
    queryFn: () => apiClient.getGovernmentFundings({ page: 1, limit: LOOKUP_FETCH_LIMIT }),
    staleTime: 5 * 60 * 1000, // 5 minutes - funding configs rarely change
    enabled: !!state,
  });

  // Find funding for this org's state
  const funding = fundingsResponse?.data?.find((f) => f.state === state);

  // Fetch funding details with all periods if we found a matching funding
  const { data: fundingDetails } = useQuery({
    queryKey: queryKeys.governmentFundings.detailCached(funding?.id),
    queryFn: () => apiClient.getGovernmentFunding(funding!.id, 0), // 0 = all periods
    staleTime: 5 * 60 * 1000,
    enabled: !!funding?.id,
  });

  // Extract unique attributes with their keys
  // Use value as the unique identifier (same value won't appear twice)
  const attributeMap = new Map<string, FundingAttribute>();
  // Collect properties that should be auto-applied to every contract
  const defaultProperties: Record<string, string> = {};

  if (fundingDetails?.periods) {
    const contractFrom = fromDate || '';
    const contractTo = toDate || '9999-12-31'; // Far future if no end date

    for (const period of fundingDetails.periods) {
      const periodFrom = period.from;
      const periodTo = period.to || '9999-12-31';

      // Check if periods overlap
      const overlaps = periodFrom <= contractTo && periodTo >= contractFrom;

      if (overlaps && period.properties) {
        for (const prop of period.properties) {
          const key = prop.key?.toLowerCase();
          const value = prop.value?.toLowerCase();
          if (key && value && !attributeMap.has(value)) {
            attributeMap.set(value, { key, value, label: prop.label || value });
          }
          if (key && value && prop.apply_to_all_contracts && !(key in defaultProperties)) {
            defaultProperties[key] = value;
          }
        }
      }
    }
  }

  // Build list of attributes sorted by value
  const fundingAttributes = Array.from(attributeMap.values()).sort((a, b) =>
    a.value.localeCompare(b.value)
  );

  // Group attributes by key for UI organization
  const attributesByKey: Record<string, FundingAttribute[]> = {};
  for (const attr of fundingAttributes) {
    if (!attributesByKey[attr.key]) {
      attributesByKey[attr.key] = [];
    }
    attributesByKey[attr.key].push(attr);
  }

  return {
    fundingAttributes,
    attributesByKey,
    defaultProperties,
    isLoading: !fundingDetails && !!state,
    hasNoFunding: !!state && !funding,
  };
}
