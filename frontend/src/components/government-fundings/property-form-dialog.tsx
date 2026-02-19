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
import {
  governmentFundingPropertySchema,
  type GovernmentFundingPropertyFormData,
} from '@/lib/schemas';

interface PropertyFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: GovernmentFundingPropertyFormData) => void;
  isSaving: boolean;
}

export function PropertyFormDialog({
  open,
  onOpenChange,
  onSubmit,
  isSaving,
}: PropertyFormDialogProps) {
  const t = useTranslations();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<GovernmentFundingPropertyFormData>({
    resolver: zodResolver(governmentFundingPropertySchema),
    defaultValues: {
      key: '',
      value: '',
      label: '',
      payment_euros: 0,
      requirement: 0,
      min_age: null,
      max_age: null,
      comment: '',
    },
  });

  const handleOpenChange = (isOpen: boolean) => {
    if (isOpen) {
      reset({
        key: '',
        value: '',
        label: '',
        payment_euros: 0,
        requirement: 0,
        min_age: null,
        max_age: null,
        comment: '',
      });
    }
    onOpenChange(isOpen);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('governmentFundings.addProperty')}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="label">{t('governmentFundings.label')}</Label>
            <Input id="label" placeholder="Ganztag" {...register('label')} />
            {errors.label && (
              <p className="text-sm text-destructive">{t('validation.labelRequired')}</p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="key">{t('governmentFundings.key')}</Label>
              <Input id="key" placeholder="care_type" {...register('key')} />
              {errors.key && (
                <p className="text-sm text-destructive">{t('validation.keyRequired')}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="value">{t('governmentFundings.value')}</Label>
              <Input id="value" placeholder="ganztag" {...register('value')} />
              {errors.value && (
                <p className="text-sm text-destructive">{t('validation.valueRequired')}</p>
              )}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="payment_euros">{t('governmentFundings.paymentInEuros')}</Label>
              <Input
                id="payment_euros"
                type="number"
                min={0}
                step={0.01}
                {...register('payment_euros', { valueAsNumber: true })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="requirement">{t('governmentFundings.requirement')}</Label>
              <Input
                id="requirement"
                type="number"
                min={0}
                step={0.01}
                {...register('requirement', { valueAsNumber: true })}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="min_age">{t('governmentFundings.minAge')}</Label>
              <Input
                id="min_age"
                type="number"
                min={0}
                {...register('min_age', { valueAsNumber: true })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="max_age">{t('governmentFundings.maxAge')}</Label>
              <Input
                id="max_age"
                type="number"
                min={0}
                {...register('max_age', { valueAsNumber: true })}
              />
            </div>
          </div>
          <p className="text-xs text-muted-foreground">{t('governmentFundings.ageRangeHelp')}</p>

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
