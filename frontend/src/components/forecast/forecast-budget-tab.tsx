'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Plus, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { LOOKUP_FETCH_LIMIT } from '@/lib/api/types';
import { useForecastStore } from '@/stores/forecast-store';

export function ForecastBudgetTab() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const store = useForecastStore();

  const [name, setName] = useState('');
  const [category, setCategory] = useState('expense');
  const [perChild, setPerChild] = useState(false);
  const [entryFrom, setEntryFrom] = useState('');
  const [entryTo, setEntryTo] = useState('');
  const [amountEur, setAmountEur] = useState<number | ''>('');

  const { data: existingBudgetItems } = useQuery({
    queryKey: queryKeys.budgetItems.all(orgId),
    queryFn: () => apiClient.getBudgetItems(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });

  const canAdd = name && entryFrom && amountEur;

  const handleAdd = () => {
    if (!canAdd) return;
    store.addBudgetItem({
      name,
      category,
      per_child: perChild,
      entries: [
        {
          from: entryFrom,
          to: entryTo || undefined,
          amount_cents: Math.round(Number(amountEur) * 100),
        },
      ],
    });
    setName('');
    setAmountEur('');
    setEntryFrom('');
    setEntryTo('');
  };

  return (
    <div className="space-y-6">
      {/* Add Budget Item Form */}
      <div className="space-y-4">
        <h4 className="text-sm font-medium">{t('statistics.forecastAddBudgetItem')}</h4>
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
          <div className="space-y-1">
            <Label>{t('common.name')}</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label>{t('budgetItems.category')}</Label>
            <Select value={category} onValueChange={setCategory}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="income">{t('budgetItems.categoryIncome')}</SelectItem>
                <SelectItem value="expense">{t('budgetItems.categoryExpense')}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-end space-x-2 pb-0.5">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={perChild}
                onChange={(e) => setPerChild(e.target.checked)}
                className="h-4 w-4"
              />
              {t('budgetItems.perChild')}
            </label>
          </div>
          <div className="space-y-1">
            <Label>{t('contracts.from')}</Label>
            <Input type="date" value={entryFrom} onChange={(e) => setEntryFrom(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label>{t('contracts.to')}</Label>
            <Input type="date" value={entryTo} onChange={(e) => setEntryTo(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label>{t('budgetItems.amountInEuros')}</Label>
            <Input
              type="number"
              min={0}
              step={0.01}
              value={amountEur}
              onChange={(e) => setAmountEur(e.target.value ? Number(e.target.value) : '')}
            />
          </div>
        </div>
        <Button size="sm" onClick={handleAdd} disabled={!canAdd}>
          <Plus className="mr-1 h-4 w-4" />
          {t('statistics.forecastAddBudgetItem')}
        </Button>
      </div>

      {/* Added budget items */}
      {store.addBudgetItems.length > 0 && (
        <div className="space-y-2">
          <h4 className="text-sm font-medium">{t('statistics.forecastAdded')}</h4>
          <div className="flex flex-wrap gap-2">
            {store.addBudgetItems.map((item, i) => (
              <Badge key={i} variant="secondary" className="gap-1">
                {item.name} ({item.category})
                <button onClick={() => store.removeAddedBudgetItem(i)} className="ml-1">
                  <X className="h-3 w-3" />
                </button>
              </Badge>
            ))}
          </div>
        </div>
      )}

      {/* Remove existing budget items */}
      <div className="space-y-2">
        <h4 className="text-sm font-medium">{t('statistics.forecastRemoveBudgetItem')}</h4>
        {existingBudgetItems && existingBudgetItems.data.length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {existingBudgetItems.data.map((item) => {
              const isRemoved = store.removeBudgetItemIds.includes(item.id);
              return (
                <Badge
                  key={item.id}
                  variant={isRemoved ? 'destructive' : 'outline'}
                  className="cursor-pointer"
                  onClick={() => store.toggleRemoveBudgetItem(item.id)}
                >
                  {item.name}
                  {isRemoved && <X className="ml-1 h-3 w-3" />}
                </Badge>
              );
            })}
          </div>
        ) : (
          <p className="text-muted-foreground text-sm">{t('common.noResults')}</p>
        )}
      </div>
    </div>
  );
}
