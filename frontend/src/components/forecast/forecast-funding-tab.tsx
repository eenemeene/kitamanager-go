'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { Plus, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { useForecastStore } from '@/stores/forecast-store';

export function ForecastFundingTab() {
  const t = useTranslations();
  const store = useForecastStore();

  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [fullTimeWeeklyHours, setFullTimeWeeklyHours] = useState<number | ''>(40);

  const handleAdd = () => {
    if (!from || !fullTimeWeeklyHours) return;
    store.addFundingPeriod({
      from,
      to: to || undefined,
      full_time_weekly_hours: Number(fullTimeWeeklyHours),
      properties: [],
    });
    setFrom('');
    setTo('');
  };

  return (
    <div className="space-y-6">
      <div className="space-y-4">
        <h4 className="text-sm font-medium">{t('statistics.forecastAddFunding')}</h4>
        <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
          <div className="space-y-1">
            <Label>{t('contracts.from')}</Label>
            <Input type="date" value={from} onChange={(e) => setFrom(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label>{t('contracts.to')}</Label>
            <Input type="date" value={to} onChange={(e) => setTo(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label>{t('employees.weeklyHours')}</Label>
            <Input
              type="number"
              min={0}
              step={0.5}
              value={fullTimeWeeklyHours}
              onChange={(e) => setFullTimeWeeklyHours(e.target.value ? Number(e.target.value) : '')}
            />
          </div>
        </div>
        <Button size="sm" onClick={handleAdd} disabled={!from || !fullTimeWeeklyHours}>
          <Plus className="mr-1 h-4 w-4" />
          {t('statistics.forecastAddFunding')}
        </Button>
      </div>

      {store.addFundingPeriods.length > 0 && (
        <div className="space-y-2">
          <h4 className="text-sm font-medium">{t('statistics.forecastAdded')}</h4>
          <div className="flex flex-wrap gap-2">
            {store.addFundingPeriods.map((period, i) => (
              <Badge key={i} variant="secondary" className="gap-1">
                {period.from} — {period.to ?? '∞'} ({period.full_time_weekly_hours}h)
                <button onClick={() => store.removeFundingPeriod(i)} className="ml-1">
                  <X className="h-3 w-3" />
                </button>
              </Badge>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
