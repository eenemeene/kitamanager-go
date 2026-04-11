'use client';

import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { LOOKUP_FETCH_LIMIT } from '@/lib/api/types';
import { useForecastStore } from '@/stores/forecast-store';

export function ForecastSalaryTab() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const store = useForecastStore();

  const { data: payPlans } = useQuery({
    queryKey: queryKeys.payPlans.all(orgId),
    queryFn: () => apiClient.getPayPlans(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });

  const handlePercentChange = (value: string) => {
    const percent = value ? parseFloat(value) : null;
    store.setSalaryIncrease(percent, store.salaryEffectiveFrom, payPlans?.data ?? []);
  };

  const handleDateChange = (value: string) => {
    store.setSalaryIncrease(store.salaryIncreasePercent, value || null, payPlans?.data ?? []);
  };

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
        <div className="space-y-1">
          <Label>{t('statistics.forecastSalaryPercent')}</Label>
          <Input
            type="number"
            step={0.1}
            min={0}
            placeholder="e.g. 3.5"
            value={store.salaryIncreasePercent ?? ''}
            onChange={(e) => handlePercentChange(e.target.value)}
          />
        </div>
        <div className="space-y-1">
          <Label>{t('statistics.forecastEffectiveFrom')}</Label>
          <Input
            type="date"
            value={store.salaryEffectiveFrom ?? ''}
            onChange={(e) => handleDateChange(e.target.value)}
          />
        </div>
      </div>

      {store.addPayPlanPeriods.length > 0 && (
        <div className="space-y-2">
          <p className="text-muted-foreground text-sm">
            {t('statistics.forecastSalaryIncrease')}: +{store.salaryIncreasePercent}%{' '}
            {store.salaryEffectiveFrom && (
              <>
                {t('statistics.forecastEffectiveFrom').toLowerCase()} {store.salaryEffectiveFrom}
              </>
            )}
          </p>
          <div className="flex flex-wrap gap-2">
            {store.addPayPlanPeriods.map((period, i) => {
              const pp = payPlans?.data.find((p) => p.id === period.pay_plan_id);
              return (
                <Badge key={i} variant="secondary">
                  {pp?.name ?? `Pay Plan #${period.pay_plan_id}`}: {period.entries.length}{' '}
                  {period.entries.length === 1 ? 'entry' : 'entries'}
                </Badge>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
