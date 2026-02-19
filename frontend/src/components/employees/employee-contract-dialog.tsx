'use client';

import { useTranslations } from 'next-intl';
import { UseFormRegister, UseFormWatch, UseFormSetValue, FieldErrors } from 'react-hook-form';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Checkbox } from '@/components/ui/checkbox';
import type { EmployeeContract, PayPlan, Section } from '@/lib/api/types';
import { formatDate } from '@/lib/utils/formatting';
import type { EmployeeContractFormData } from '@/lib/schemas';

interface ActiveContractInfo {
  contract: EmployeeContract;
  endCurrentContract: boolean;
  onEndCurrentContractChange: (checked: boolean) => void;
}

export interface EmployeeContractDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  register: UseFormRegister<EmployeeContractFormData>;
  onSubmit: (e: React.FormEvent) => void;
  errors: FieldErrors<EmployeeContractFormData>;
  watch: UseFormWatch<EmployeeContractFormData>;
  setValue: UseFormSetValue<EmployeeContractFormData>;
  isSaving: boolean;
  payPlans: PayPlan[];
  sections: Section[];
  activeContractInfo?: ActiveContractInfo;
}

export function EmployeeContractDialog({
  open,
  onOpenChange,
  title,
  register,
  onSubmit,
  errors,
  watch,
  setValue,
  isSaving,
  payPlans,
  sections,
  activeContractInfo,
}: EmployeeContractDialogProps) {
  const t = useTranslations();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          {activeContractInfo && (
            <Alert>
              <AlertDescription className="space-y-3">
                <p className="font-medium">{t('contracts.hasActiveContractEmployee')}</p>
                <p className="text-muted-foreground text-sm">
                  {t('contracts.activeSinceEmployee', {
                    date: formatDate(activeContractInfo.contract.from),
                    staffCategory: t(
                      `employees.staffCategory.${activeContractInfo.contract.staff_category}`
                    ),
                    grade: activeContractInfo.contract.grade,
                    step: activeContractInfo.contract.step,
                  })}
                </p>
                <div className="flex items-center space-x-2">
                  <Checkbox
                    id="endCurrentContract"
                    checked={activeContractInfo.endCurrentContract}
                    onCheckedChange={(checked) =>
                      activeContractInfo.onEndCurrentContractChange(checked === true)
                    }
                  />
                  <label
                    htmlFor="endCurrentContract"
                    className="text-sm leading-none font-medium peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                  >
                    {t('contracts.endCurrentContract')}
                  </label>
                </div>
              </AlertDescription>
            </Alert>
          )}

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="from">{t('contracts.startDate')}</Label>
              <Input id="from" type="date" {...register('from')} />
              {errors.from && (
                <p className="text-destructive text-sm">{t('contracts.startDateRequired')}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="to">{t('contracts.endDateOptional')}</Label>
              <Input id="to" type="date" {...register('to')} />
            </div>
          </div>

          {sections.length > 0 && (
            <div className="space-y-2">
              <Label htmlFor="section_id">{t('sections.title')} *</Label>
              <Select
                value={watch('section_id')?.toString() || ''}
                onValueChange={(val) => setValue('section_id', val ? Number(val) : 0)}
              >
                <SelectTrigger id="section_id" aria-label={t('sections.title')}>
                  <SelectValue placeholder={t('sections.selectSection')} />
                </SelectTrigger>
                <SelectContent>
                  {sections.map((s) => (
                    <SelectItem key={s.id} value={String(s.id)}>
                      {s.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {errors.section_id && (
                <p className="text-destructive text-sm">{t('validation.sectionRequired')}</p>
              )}
            </div>
          )}

          <div className="space-y-2">
            <Label htmlFor="payplan_id">{t('employees.payPlan')}</Label>
            <Select
              value={String(watch('payplan_id') || '')}
              onValueChange={(val) => setValue('payplan_id', Number(val))}
            >
              <SelectTrigger id="payplan_id" aria-label={t('employees.payPlan')}>
                <SelectValue placeholder={t('employees.selectPayPlan')} />
              </SelectTrigger>
              <SelectContent>
                {payPlans.map((pp) => (
                  <SelectItem key={pp.id} value={String(pp.id)}>
                    {pp.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {errors.payplan_id && (
              <p className="text-destructive text-sm">{t('employees.selectPayPlan')}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="staff_category">{t('employees.staffCategory.label')}</Label>
            <Select
              value={watch('staff_category') || ''}
              onValueChange={(val) =>
                setValue('staff_category', val as 'qualified' | 'supplementary' | 'non_pedagogical')
              }
            >
              <SelectTrigger id="staff_category" aria-label={t('employees.staffCategory.label')}>
                <SelectValue placeholder={t('employees.staffCategory.label')} />
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
            {errors.staff_category && (
              <p className="text-destructive text-sm">{t('validation.staffCategoryRequired')}</p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="grade">{t('employees.grade')}</Label>
              <Input id="grade" {...register('grade')} placeholder="S8a" />
              {errors.grade && (
                <p className="text-destructive text-sm">{t('payPlans.gradeRequired')}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="step">{t('employees.step')}</Label>
              <Input
                id="step"
                type="number"
                min={1}
                max={6}
                {...register('step', { valueAsNumber: true })}
              />
              {errors.step && (
                <p className="text-destructive text-sm">{t('payPlans.stepRequired')}</p>
              )}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="weekly_hours">{t('employees.weeklyHours')}</Label>
            <Input
              id="weekly_hours"
              type="number"
              min={0}
              max={168}
              step={0.5}
              {...register('weekly_hours', { valueAsNumber: true })}
            />
            {errors.weekly_hours && (
              <p className="text-destructive text-sm">{t('validation.weeklyHoursRequired')}</p>
            )}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={isSaving}>
              {t('common.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
