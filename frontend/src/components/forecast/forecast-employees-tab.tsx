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
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { LOOKUP_FETCH_LIMIT } from '@/lib/api/types';
import type { Section } from '@/lib/api/types';
import { useForecastStore } from '@/stores/forecast-store';

export function ForecastEmployeesTab() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const store = useForecastStore();

  // Form state — focused on staffing-relevant fields
  const [count, setCount] = useState(1);
  const [contractFrom, setContractFrom] = useState('');
  const [contractTo, setContractTo] = useState('');
  const [sectionId, setSectionId] = useState<number | undefined>(undefined);
  const [staffCategory, setStaffCategory] = useState('qualified');
  const [grade, setGrade] = useState('');
  const [step, setStep] = useState<number | ''>('');
  const [weeklyHours, setWeeklyHours] = useState<number | ''>(39);
  const [payPlanId, setPayPlanId] = useState<number | undefined>(undefined);

  const { data: sections } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });

  const { data: payPlans } = useQuery({
    queryKey: queryKeys.payPlans.all(orgId),
    queryFn: () => apiClient.getPayPlans(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });

  const { data: existingEmployees } = useQuery({
    queryKey: queryKeys.employees.allUnpaginated(orgId),
    queryFn: () => apiClient.getEmployeesAll(orgId),
    enabled: !!orgId,
  });

  const canAdd = contractFrom && sectionId && weeklyHours && payPlanId;

  const handleAdd = () => {
    if (!canAdd || !sectionId || !payPlanId) return;
    // Auto-generate a birthdate (30 years ago) — not relevant for forecast calculations
    const birthYear = new Date().getFullYear() - 30;
    const birthdate = `${birthYear}-01-01`;

    for (let i = 0; i < count; i++) {
      store.addEmployee({
        first_name: `Employee`,
        last_name: `#${store.addEmployees.length + i + 1}`,
        gender: 'diverse',
        birthdate,
        contracts: [
          {
            from: contractFrom,
            to: contractTo || undefined,
            section_id: sectionId,
            staff_category: staffCategory,
            grade: grade || undefined,
            step: step ? Number(step) : undefined,
            weekly_hours: Number(weeklyHours),
            pay_plan_id: payPlanId,
          },
        ],
      });
    }
    setCount(1);
    setContractFrom('');
    setContractTo('');
    setGrade('');
    setStep('');
  };

  return (
    <div className="space-y-6">
      {/* Add Employee Form */}
      <div className="space-y-4">
        <h4 className="text-sm font-medium">{t('statistics.forecastAddEmployee')}</h4>
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
            <Label>{t('employees.staffCategory.label')}</Label>
            <Select value={staffCategory} onValueChange={setStaffCategory}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="qualified">{t('employees.staffCategory.qualified')}</SelectItem>
                <SelectItem value="supplementary">
                  {t('employees.staffCategory.supplementary')}
                </SelectItem>
                <SelectItem value="non_pedagogical">
                  {t('employees.staffCategory.non_pedagogical')}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1">
            <Label>{t('employees.grade')}</Label>
            <Input value={grade} onChange={(e) => setGrade(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label>{t('employees.step')}</Label>
            <Input
              type="number"
              min={1}
              value={step}
              onChange={(e) => setStep(e.target.value ? Number(e.target.value) : '')}
            />
          </div>
          <div className="space-y-1">
            <Label>{t('employees.weeklyHours')}</Label>
            <Input
              type="number"
              min={0}
              step={0.5}
              value={weeklyHours}
              onChange={(e) => setWeeklyHours(e.target.value ? Number(e.target.value) : '')}
            />
          </div>
          <div className="space-y-1">
            <Label>{t('employees.payPlan')}</Label>
            <Select
              value={payPlanId?.toString() ?? ''}
              onValueChange={(v) => setPayPlanId(Number(v))}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {payPlans?.data.map((pp) => (
                  <SelectItem key={pp.id} value={pp.id.toString()}>
                    {pp.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
        <Button size="sm" onClick={handleAdd} disabled={!canAdd}>
          <Plus className="mr-1 h-4 w-4" />
          {t('statistics.forecastAddEmployee')}
          {count > 1 && ` (×${count})`}
        </Button>
      </div>

      {/* Added employees list */}
      {store.addEmployees.length > 0 && (
        <div className="space-y-2">
          <h4 className="text-sm font-medium">
            {t('statistics.forecastAdded')} ({store.addEmployees.length})
          </h4>
          <div className="flex flex-wrap gap-2">
            {store.addEmployees.map((emp, i) => {
              const contract = emp.contracts[0];
              return (
                <Badge key={i} variant="secondary" className="gap-1">
                  {emp.first_name} {emp.last_name}
                  {contract && ` (${contract.staff_category}, ${contract.weekly_hours}h)`}
                  <button onClick={() => store.removeAddedEmployee(i)} className="ml-1">
                    <X className="h-3 w-3" />
                  </button>
                </Badge>
              );
            })}
          </div>
        </div>
      )}

      {/* Remove existing employees */}
      <div className="space-y-2">
        <h4 className="text-sm font-medium">
          <UserMinus className="mr-1 inline h-4 w-4" />
          {t('statistics.forecastRemoveEmployee')}
        </h4>
        {existingEmployees && existingEmployees.length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {existingEmployees.map((emp) => {
              const isRemoved = store.removeEmployeeIds.includes(emp.id);
              return (
                <Badge
                  key={emp.id}
                  variant={isRemoved ? 'destructive' : 'outline'}
                  className="cursor-pointer"
                  onClick={() => store.toggleRemoveEmployee(emp.id)}
                >
                  {emp.first_name} {emp.last_name}
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
