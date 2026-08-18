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
import { validationTiming } from '@/lib/forms/validation-timing';
import { useProblemFormErrors } from '@/lib/forms/use-problem-form-errors';
import { FormErrorSummary } from '@/components/forms/form-error-summary';
import {
  governmentFundingPropertySchema,
  type GovernmentFundingPropertyFormData,
} from '@/lib/schemas';

interface PropertyFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /**
   * The rejection from the mutation this dialog submits, so the fields the
   * server named get marked rather than described in a toast that scrolls away.
   */
  submitError?: unknown;
  onSubmit: (data: GovernmentFundingPropertyFormData) => void;
  isSaving: boolean;
}

export function PropertyFormDialog({
  open,
  onOpenChange,
  submitError,
  onSubmit,
  isSaving,
}: PropertyFormDialogProps) {
  const t = useTranslations();

  const {
    register,
    handleSubmit,
    reset,
    setError,
    clearErrors,
    getValues,
    formState: { errors },
  } = useForm<GovernmentFundingPropertyFormData>({
    ...validationTiming,
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

  const unmapped = useProblemFormErrors(submitError, { setError, clearErrors, getValues });

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
          <FormErrorSummary
            errors={errors}
            unmapped={unmapped}
            labels={{
              label: t('governmentFundings.label'),
              key: t('governmentFundings.key'),
              value: t('governmentFundings.value'),
              payment_euros: t('governmentFundings.paymentInEuros'),
              requirement: t('governmentFundings.requirement'),
              min_age: t('governmentFundings.minAge'),
              max_age: t('governmentFundings.maxAge'),
              comment: t('common.comment'),
            }}
          />
          <div className="space-y-2">
            <Label htmlFor="label">{t('governmentFundings.label')}</Label>
            <Input
              id="label"
              placeholder="Full-Time"
              aria-invalid={!!errors.label}
              {...register('label')}
            />
            {errors.label && (
              <p className="text-destructive text-sm">{t('validation.labelRequired')}</p>
            )}
          </div>

          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="key">{t('governmentFundings.key')}</Label>
              <Input
                id="key"
                placeholder="care_type"
                aria-invalid={!!errors.key}
                {...register('key')}
              />
              {errors.key && (
                <p className="text-destructive text-sm">{t('validation.keyRequired')}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="value">{t('governmentFundings.value')}</Label>
              <Input
                id="value"
                placeholder="ganztag"
                aria-invalid={!!errors.value}
                {...register('value')}
              />
              {errors.value && (
                <p className="text-destructive text-sm">{t('validation.valueRequired')}</p>
              )}
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="payment_euros">{t('governmentFundings.paymentInEuros')}</Label>
              <Input
                id="payment_euros"
                type="number"
                min={0}
                step={0.01}
                aria-invalid={!!errors.payment_euros}
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
                aria-invalid={!!errors.requirement}
                {...register('requirement', { valueAsNumber: true })}
              />
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="min_age">{t('governmentFundings.minAge')}</Label>
              <Input
                id="min_age"
                type="number"
                min={0}
                aria-invalid={!!errors.min_age}
                {...register('min_age', { valueAsNumber: true })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="max_age">{t('governmentFundings.maxAge')}</Label>
              <Input
                id="max_age"
                type="number"
                min={0}
                aria-invalid={!!errors.max_age}
                {...register('max_age', { valueAsNumber: true })}
              />
            </div>
          </div>
          <p className="text-muted-foreground text-xs">{t('governmentFundings.ageRangeHelp')}</p>

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
