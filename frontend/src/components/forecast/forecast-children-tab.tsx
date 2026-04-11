'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Plus, X, UserMinus } from 'lucide-react';
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
import { PropertyTagInput } from '@/components/ui/tag-input';
import type { ScalarContractProperties } from '@/components/ui/tag-input';
import { useFundingAttributes } from '@/lib/hooks/use-funding-attributes';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { LOOKUP_FETCH_LIMIT } from '@/lib/api/types';
import type { Section } from '@/lib/api/types';
import { useForecastStore } from '@/stores/forecast-store';

export function ForecastChildrenTab() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const store = useForecastStore();

  // Form state — focused on what matters for forecasting
  const [count, setCount] = useState(1);
  const [age, setAge] = useState<number | ''>(2);
  const [contractFrom, setContractFrom] = useState('');
  const [contractTo, setContractTo] = useState('');
  const [sectionId, setSectionId] = useState<number | undefined>(undefined);
  const [properties, setProperties] = useState<ScalarContractProperties | undefined>(undefined);

  const { data: sections } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });

  const { data: existingChildren } = useQuery({
    queryKey: queryKeys.children.allUnpaginated(orgId),
    queryFn: () => apiClient.getChildrenAll(orgId),
    enabled: !!orgId,
  });

  const { fundingAttributes, attributesByKey } = useFundingAttributes(orgId);

  const canAdd = age !== '' && contractFrom && sectionId;

  const handleAdd = () => {
    if (!canAdd || !sectionId) return;
    // Convert age to a birthdate (approximate: today minus age years)
    const now = new Date();
    const birthYear = now.getFullYear() - Number(age);
    const birthdate = `${birthYear}-${String(now.getMonth() + 1).padStart(2, '0')}-01`;

    for (let i = 0; i < count; i++) {
      store.addChild({
        first_name: `Child`,
        last_name: `#${store.addChildren.length + i + 1}`,
        gender: 'diverse',
        birthdate,
        contracts: [
          {
            from: contractFrom,
            to: contractTo || undefined,
            section_id: sectionId,
            properties: properties ?? undefined,
          },
        ],
      });
    }
    setCount(1);
    setAge(2);
    setContractFrom('');
    setContractTo('');
    setProperties(undefined);
  };

  return (
    <div className="space-y-6">
      {/* Add Children Form */}
      <div className="space-y-4">
        <h4 className="text-sm font-medium">{t('statistics.forecastAddChild')}</h4>
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
          <div className="space-y-1">
            <Label>{t('common.count')}</Label>
            <Input
              type="number"
              min={1}
              value={count}
              onChange={(e) => setCount(Math.max(1, Number(e.target.value) || 1))}
            />
          </div>
          <div className="space-y-1">
            <Label>{t('children.age')}</Label>
            <Input
              type="number"
              min={0}
              max={14}
              value={age}
              onChange={(e) => setAge(e.target.value ? Number(e.target.value) : '')}
            />
          </div>
          <div className="space-y-1">
            <Label>{t('contracts.from')}</Label>
            <Input
              type="date"
              value={contractFrom}
              onChange={(e) => setContractFrom(e.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label>{t('contracts.to')}</Label>
            <Input type="date" value={contractTo} onChange={(e) => setContractTo(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label>{t('sections.title')}</Label>
            <Select
              value={sectionId?.toString() ?? ''}
              onValueChange={(v) => setSectionId(Number(v))}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {sections?.data.map((s: Section) => (
                  <SelectItem key={s.id} value={s.id.toString()}>
                    {s.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
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
        <Button size="sm" onClick={handleAdd} disabled={!canAdd}>
          <Plus className="mr-1 h-4 w-4" />
          {t('statistics.forecastAddChild')}
          {count > 1 && ` (×${count})`}
        </Button>
      </div>

      {/* Added children table */}
      {store.addChildren.length > 0 && (
        <div className="space-y-2">
          <h4 className="text-sm font-medium">
            {t('statistics.forecastAdded')} ({store.addChildren.length})
          </h4>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-left">
                  <th className="px-2 py-1 font-medium">#</th>
                  <th className="px-2 py-1 font-medium">{t('children.age')}</th>
                  <th className="px-2 py-1 font-medium">{t('contracts.from')}</th>
                  <th className="px-2 py-1 font-medium">{t('sections.title')}</th>
                  <th className="px-2 py-1 font-medium">{t('contracts.propertiesLabel')}</th>
                  <th className="px-2 py-1"></th>
                </tr>
              </thead>
              <tbody>
                {store.addChildren.map((child, i) => {
                  const contract = child.contracts[0];
                  const birthYear = new Date(child.birthdate).getFullYear();
                  const childAge = new Date().getFullYear() - birthYear;
                  const sectionName =
                    sections?.data.find((s) => s.id === contract?.section_id)?.name ?? '';
                  const props = contract?.properties
                    ? Object.entries(contract.properties)
                        .map(([k, v]) => `${k}: ${v}`)
                        .join(', ')
                    : '';
                  return (
                    <tr key={i} className="border-b">
                      <td className="px-2 py-1">{i + 1}</td>
                      <td className="px-2 py-1">{childAge}</td>
                      <td className="px-2 py-1">{contract?.from}</td>
                      <td className="px-2 py-1">{sectionName}</td>
                      <td className="px-2 py-1 text-xs">{props}</td>
                      <td className="px-2 py-1">
                        <button onClick={() => store.removeAddedChild(i)}>
                          <X className="h-3 w-3" />
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Remove existing children */}
      <div className="space-y-2">
        <h4 className="text-sm font-medium">
          <UserMinus className="mr-1 inline h-4 w-4" />
          {t('statistics.forecastRemoveChild')}
        </h4>
        {existingChildren && existingChildren.length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {existingChildren.map((child) => {
              const isRemoved = store.removeChildIds.includes(child.id);
              return (
                <Badge
                  key={child.id}
                  variant={isRemoved ? 'destructive' : 'outline'}
                  className="cursor-pointer"
                  onClick={() => store.toggleRemoveChild(child.id)}
                >
                  {child.first_name} {child.last_name}
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
