'use client';

import { useTranslations } from 'next-intl';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
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
import { governmentFundingPeriodSchema, type GovernmentFundingPeriodFormData } from '@/lib/schemas';
import { validationTiming } from '@/lib/forms/validation-timing';
import { useProblemFormErrors } from '@/lib/forms/use-problem-form-errors';
import { FormErrorSummary } from '@/components/forms/form-error-summary';

interface PeriodFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /**
   * The rejection from the mutation this dialog submits, so the fields the
   * server named get marked rather than described in a toast that scrolls away.
   */
  submitError?: unknown;
  onSubmit: (data: GovernmentFundingPeriodFormData) => void;
  isSaving: boolean;
}

export function PeriodFormDialog({
  open,
  onOpenChange,
  submitError,
  onSubmit,
  isSaving,
}: PeriodFormDialogProps) {
  const t = useTranslations();

  const {
    register,
    handleSubmit,
    reset,
    setError,
    clearErrors,
    getValues,
    formState: { errors },
  } = useForm<GovernmentFundingPeriodFormData>({
    ...validationTiming,
    resolver: zodResolver(governmentFundingPeriodSchema),
    defaultValues: { from: '', to: '', full_time_weekly_hours: 39, comment: '' },
  });

  const unmapped = useProblemFormErrors(submitError, { setError, clearErrors, getValues });

  const handleOpenChange = (isOpen: boolean) => {
    if (isOpen) {
      reset({ from: '', to: '', full_time_weekly_hours: 39, comment: '' });
    }
    onOpenChange(isOpen);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('governmentFundings.addPeriod')}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <FormErrorSummary
            errors={errors}
            unmapped={unmapped}
            labels={{
              from: t('governmentFundings.fromDate'),
              to: t('governmentFundings.toDateOptional'),
              full_time_weekly_hours: t('governmentFundings.fullTimeWeeklyHours'),
              comment: t('common.comment'),
            }}
          />
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="from">{t('governmentFundings.fromDate')}</Label>
              <Input id="from" type="date" aria-invalid={!!errors.from} {...register('from')} />
              {errors.from && (
                <p className="text-destructive text-sm">{t('validation.fromDateRequired')}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="to">{t('governmentFundings.toDateOptional')}</Label>
              <Input id="to" type="date" aria-invalid={!!errors.to} {...register('to')} />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="full_time_weekly_hours">
              {t('governmentFundings.fullTimeWeeklyHours')}
            </Label>
            <Input
              id="full_time_weekly_hours"
              type="number"
              min={0.1}
              max={80}
              step={0.5}
              aria-invalid={!!errors.full_time_weekly_hours}
              {...register('full_time_weekly_hours', { valueAsNumber: true })}
            />
            {errors.full_time_weekly_hours && (
              <p className="text-destructive text-sm">
                {t('validation.fullTimeWeeklyHoursRequired')}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="comment">{t('common.comment')}</Label>
            <Input id="comment" aria-invalid={!!errors.comment} {...register('comment')} />
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
