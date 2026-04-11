'use client';

import { useTranslations } from 'next-intl';
import { Badge } from '@/components/ui/badge';
import { useForecastStore } from '@/stores/forecast-store';

export function ForecastModificationSummary() {
  const t = useTranslations('statistics');
  const store = useForecastStore();

  const items: { label: string; count: number }[] = [
    { label: t('forecastAddChild'), count: store.addChildren.length },
    { label: t('forecastRemoveChild'), count: store.removeChildIds.length },
    { label: t('forecastAddEmployee'), count: store.addEmployees.length },
    { label: t('forecastRemoveEmployee'), count: store.removeEmployeeIds.length },
    { label: t('forecastSalaryIncrease'), count: store.addPayPlanPeriods.length > 0 ? 1 : 0 },
    { label: t('forecastAddFunding'), count: store.addFundingPeriods.length },
    { label: t('forecastAddBudgetItem'), count: store.addBudgetItems.length },
    { label: t('forecastRemoveBudgetItem'), count: store.removeBudgetItemIds.length },
  ];

  const active = items.filter((i) => i.count > 0);
  if (active.length === 0) return null;

  return (
    <div className="flex flex-wrap gap-2">
      {active.map((item) => (
        <Badge key={item.label} variant="secondary">
          {item.label}: {item.count}
        </Badge>
      ))}
    </div>
  );
}
