'use client';

import { useState, useCallback } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Sparkles, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { PropertyTagInput } from '@/components/ui/tag-input';
import type { ScalarContractProperties } from '@/components/ui/tag-input';
import { useFundingAttributes } from '@/lib/hooks/use-funding-attributes';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { LOOKUP_FETCH_LIMIT } from '@/lib/api/types';
import type { Section, ForecastAddChild } from '@/lib/api/types';
import { useForecastStore } from '@/stores/forecast-store';
import { useUiStore } from '@/stores/ui-store';
import { calculateContractEndDate } from '@/lib/utils/school-enrollment';
import { formatDateForApi } from '@/lib/utils/formatting';

export function ForecastOptimizeTab() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const store = useForecastStore();
  const organizations = useUiStore((s) => s.organizations);
  const orgState = organizations.find((o) => o.id === orgId)?.state ?? 'berlin';

  // Contract start month derived from Kita year (store.from = YYYY-08-01)
  const contractStartMonth = store.from ? store.from.slice(0, 7) : '';

  // Optimizer inputs
  const [targetBalanceEur, setTargetBalanceEur] = useState(0);
  const [maxPerSectionPerMonth, setMaxPerSectionPerMonth] = useState(2);
  const [sectionIds, setSectionIds] = useState<number[]>([]);
  const [properties, setProperties] = useState<ScalarContractProperties | undefined>({
    care_type: 'ganztag',
  });

  // Optimizer state
  const [isOptimizing, setIsOptimizing] = useState(false);
  const [optimizeResult, setOptimizeResult] = useState<{
    childrenAdded: number;
    finalBalance: number;
  } | null>(null);
  const [optimizeError, setOptimizeError] = useState<string | null>(null);

  const { data: sections } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });

  const { fundingAttributes, attributesByKey } = useFundingAttributes(orgId);

  const canOptimize = sectionIds.length > 0 && !!contractStartMonth && !isOptimizing;

  const toggleSection = (id: number) => {
    setSectionIds((prev) => (prev.includes(id) ? prev.filter((s) => s !== id) : [...prev, id]));
  };

  /** Derive a typical child age (in years) from a section's age range. */
  const getAgeForSection = useCallback((section: Section): number => {
    const minMonths = section.min_age_months ?? 0;
    const maxMonths = section.max_age_months ?? 72; // default 6 years
    // Use midpoint of the section's age range
    const midMonths = Math.round((minMonths + maxMonths) / 2);
    return Math.floor(midMonths / 12);
  }, []);

  const buildChildren = useCallback(
    (count: number): ForecastAddChild[] => {
      const now = new Date();
      const sectionList = sections?.data.filter((s) => sectionIds.includes(s.id)) ?? [];

      const children: ForecastAddChild[] = [];
      const startDate = new Date(contractStartMonth + '-01');
      const endDate = store.to ? new Date(store.to) : new Date(startDate.getFullYear(), 11, 31);

      let added = 0;
      const currentDate = new Date(startDate);

      while (added < count && currentDate <= endDate) {
        const monthStr = `${currentDate.getFullYear()}-${String(currentDate.getMonth() + 1).padStart(2, '0')}`;
        const contractFrom = `${monthStr}-01`;

        for (const sec of sectionList) {
          const childAge = getAgeForSection(sec);
          const birthYear = now.getFullYear() - childAge;
          const birthdate = `${birthYear}-${String(now.getMonth() + 1).padStart(2, '0')}-01`;
          const contractTo = calculateContractEndDate(birthdate, orgState) ?? undefined;

          for (let j = 0; j < maxPerSectionPerMonth && added < count; j++) {
            children.push({
              first_name: 'Child',
              last_name: `#${added + 1}`,
              gender: 'diverse',
              birthdate,
              contracts: [
                {
                  from: contractFrom,
                  to: contractTo,
                  section_id: sec.id,
                  properties: properties ?? undefined,
                },
              ],
            });
            added++;
          }
        }

        currentDate.setMonth(currentDate.getMonth() + 1);
      }

      return children;
    },
    [
      orgState,
      contractStartMonth,
      sectionIds,
      sections,
      maxPerSectionPerMonth,
      properties,
      getAgeForSection,
      store.from,
      store.to,
    ]
  );

  const getCumulativeBalance = (
    response: Awaited<ReturnType<typeof apiClient.postForecast>>
  ): number => {
    if (!response.financials?.data_points) return 0;
    return response.financials.data_points.reduce((sum, dp) => sum + dp.balance, 0);
  };

  const handleOptimize = useCallback(async () => {
    setIsOptimizing(true);
    setOptimizeResult(null);
    setOptimizeError(null);

    try {
      const targetCents = Math.round(targetBalanceEur * 100);
      const maxChildrenPerMonth = sectionIds.length * maxPerSectionPerMonth;

      // Step 1: Get baseline balance (with current store modifications but no optimizer children)
      const baselineReq = store.buildRequest();
      const baselineResp = await apiClient.postForecast(orgId, baselineReq);
      const baselineBalance = getCumulativeBalance(baselineResp);

      if (baselineBalance >= targetCents) {
        setOptimizeResult({ childrenAdded: 0, finalBalance: baselineBalance });
        setIsOptimizing(false);
        return;
      }

      // Step 2: Get per-child impact by adding 1 child
      const oneChildChildren = buildChildren(1);
      const toApiChildren = (children: ForecastAddChild[]) =>
        children.map((c) => ({
          ...c,
          birthdate: formatDateForApi(c.birthdate)!,
          contracts: c.contracts.map((ct) => ({
            ...ct,
            from: formatDateForApi(ct.from)!,
            to: ct.to ? formatDateForApi(ct.to)! : undefined,
          })),
        }));

      const oneChildReq = {
        ...baselineReq,
        add_children: [...(baselineReq.add_children ?? []), ...toApiChildren(oneChildChildren)],
      };
      const oneChildResp = await apiClient.postForecast(orgId, oneChildReq);
      const oneChildBalance = getCumulativeBalance(oneChildResp);
      const perChildImpact = oneChildBalance - baselineBalance;

      if (perChildImpact <= 0) {
        setOptimizeError(t('statistics.forecastOptimizeNoImpact'));
        setIsOptimizing(false);
        return;
      }

      // Step 3: Estimate and binary search
      const deficit = targetCents - baselineBalance;
      const estimate = Math.ceil(deficit / perChildImpact);

      // Calculate max possible children (limited by sections × months × max-per-section)
      const startDate = new Date(contractStartMonth + '-01');
      const endDate = store.to ? new Date(store.to) : new Date(startDate.getFullYear(), 11, 31);
      let months = 0;
      const d = new Date(startDate);
      while (d <= endDate) {
        months++;
        d.setMonth(d.getMonth() + 1);
      }
      const maxPossibleChildren = months * maxChildrenPerMonth;

      let low = 1;
      let high = Math.min(estimate * 2, maxPossibleChildren);
      let bestCount = high;
      let bestBalance = 0;

      // Binary search for minimum children needed
      for (let iter = 0; iter < 15 && low <= high; iter++) {
        const mid = Math.ceil((low + high) / 2);
        const children = buildChildren(mid);
        const req = {
          ...baselineReq,
          add_children: [...(baselineReq.add_children ?? []), ...toApiChildren(children)],
        };
        const resp = await apiClient.postForecast(orgId, req);
        const balance = getCumulativeBalance(resp);

        if (balance >= targetCents) {
          bestCount = mid;
          bestBalance = balance;
          high = mid - 1;
        } else {
          low = mid + 1;
        }
      }

      // Populate the store with the optimal children
      const optimalChildren = buildChildren(bestCount);
      // Clear any previously optimizer-added children (keep manually added ones)
      // We add them to the store so the user can see and edit them
      for (const child of optimalChildren) {
        store.addChild(child);
      }

      setOptimizeResult({ childrenAdded: bestCount, finalBalance: bestBalance });
    } catch (err) {
      setOptimizeError(err instanceof Error ? err.message : String(err));
    } finally {
      setIsOptimizing(false);
    }
  }, [
    targetBalanceEur,
    sectionIds,
    maxPerSectionPerMonth,
    contractStartMonth,
    store,
    orgId,
    buildChildren,
    t,
  ]);

  return (
    <div className="space-y-6">
      <div className="space-y-4">
        <h4 className="text-sm font-medium">{t('statistics.forecastOptimizeDescription')}</h4>

        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
          <div className="space-y-1">
            <Label>{t('statistics.forecastOptimizeTarget')}</Label>
            <Input
              type="number"
              step={100}
              value={targetBalanceEur}
              onChange={(e) => setTargetBalanceEur(Number(e.target.value) || 0)}
            />
          </div>
          <div className="space-y-1">
            <Label>{t('statistics.forecastOptimizeMaxPerSection')}</Label>
            <Input
              type="number"
              min={1}
              value={maxPerSectionPerMonth}
              onChange={(e) => setMaxPerSectionPerMonth(Math.max(1, Number(e.target.value) || 1))}
            />
          </div>
          <div className="space-y-1">
            <Label>{t('contracts.propertiesLabel')}</Label>
            <PropertyTagInput
              value={properties}
              onChange={setProperties}
              fundingAttributes={fundingAttributes}
              attributesByKey={attributesByKey}
              placeholder={t('contracts.propertiesPlaceholder')}
              suggestionsLabel={t('contracts.suggestedProperties')}
            />
          </div>
        </div>

        {/* Section selection */}
        <div className="space-y-2">
          <Label>{t('statistics.forecastOptimizeSections')}</Label>
          <div className="flex flex-wrap gap-2">
            {sections?.data.map((s: Section) => {
              const age = getAgeForSection(s);
              return (
                <Button
                  key={s.id}
                  variant={sectionIds.includes(s.id) ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => toggleSection(s.id)}
                >
                  {s.name} ({t('children.age')}: {age})
                </Button>
              );
            })}
          </div>
        </div>

        <Button onClick={handleOptimize} disabled={!canOptimize}>
          {isOptimizing ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              {t('statistics.forecastOptimizeRunning')}
            </>
          ) : (
            <>
              <Sparkles className="mr-2 h-4 w-4" />
              {t('statistics.forecastOptimize')}
            </>
          )}
        </Button>
      </div>

      {/* Results */}
      {optimizeResult && (
        <div className="rounded-md border bg-green-50 p-4 dark:bg-green-950">
          <p className="text-sm font-medium">
            {t('statistics.forecastOptimizeResult', {
              count: optimizeResult.childrenAdded,
              balance: (optimizeResult.finalBalance / 100).toLocaleString('de-DE', {
                style: 'currency',
                currency: 'EUR',
              }),
            })}
          </p>
          {optimizeResult.childrenAdded > 0 && (
            <p className="text-muted-foreground mt-1 text-sm">
              {t('statistics.forecastOptimizeAddedToStore')}
            </p>
          )}
        </div>
      )}

      {optimizeError && (
        <div className="rounded-md border bg-red-50 p-4 dark:bg-red-950">
          <p className="text-destructive text-sm">{optimizeError}</p>
        </div>
      )}
    </div>
  );
}
