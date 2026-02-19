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

interface PeriodFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: GovernmentFundingPeriodFormData) => void;
  isSaving: boolean;
}

export function PeriodFormDialog({
  open,
  onOpenChange,
  onSubmit,
  isSaving,
}: PeriodFormDialogProps) {
  const t = useTranslations();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<GovernmentFundingPeriodFormData>({
    resolver: zodResolver(governmentFundingPeriodSchema),
    defaultValues: { from: '', to: '', full_time_weekly_hours: 39, comment: '' },
  });

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
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="from">{t('governmentFundings.fromDate')}</Label>
              <Input id="from" type="date" {...register('from')} />
              {errors.from && (
                <p className="text-sm text-destructive">{t('validation.fromDateRequired')}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="to">{t('governmentFundings.toDateOptional')}</Label>
              <Input id="to" type="date" {...register('to')} />
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
              {...register('full_time_weekly_hours', { valueAsNumber: true })}
            />
            {errors.full_time_weekly_hours && (
              <p className="text-sm text-destructive">
                {t('validation.fullTimeWeeklyHoursRequired')}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="comment">{t('common.comment')}</Label>
            <Input id="comment" {...register('comment')} />
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
