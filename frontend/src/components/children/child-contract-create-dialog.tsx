'use client';

import { useEffect, useState, useCallback, useRef } from 'react';
import { useTranslations } from 'next-intl';
import { useForm, Controller } from 'react-hook-form';
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Checkbox } from '@/components/ui/checkbox';
import { PropertyTagInput } from '@/components/ui/tag-input';
import { useToast } from '@/lib/hooks/use-toast';
import { useFundingAttributes } from '@/lib/hooks/use-funding-attributes';
import { childContractSchema, type ChildContractFormData } from '@/lib/schemas';
import { formatDate, formatDateForInput, toLocalDateString } from '@/lib/utils/formatting';
import { propertiesToLabelKeys } from '@/lib/utils/contract-properties';
import { getActiveContract, isDateBefore } from '@/lib/utils/contracts';
import { calculateContractEndDate } from '@/lib/utils/school-enrollment';
import type { Child, Section, ContractProperties } from '@/lib/api/types';
import { validationTiming } from '@/lib/forms/validation-timing';
import { useProblemFormErrors } from '@/lib/forms/use-problem-form-errors';
import { FormErrorSummary } from '@/components/forms/form-error-summary';

export interface ChildContractCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orgId: number;
  orgState: string | undefined;
  child: Child | null;
  sections: Section[];
  isSaving: boolean;
  onSubmit: (data: ChildContractFormData, child: Child, endCurrentContract: boolean) => void;
  /**
   * The rejection from the mutation this dialog submits, so the fields the
   * server named get marked rather than described in a toast that scrolls away.
   */
  submitError?: unknown;
}

export function ChildContractCreateDialog({
  open,
  onOpenChange,
  submitError,
  orgId,
  orgState,
  child,
  sections,
  isSaving,
  onSubmit,
}: ChildContractCreateDialogProps) {
  const t = useTranslations();
  const tLabels = useTranslations('fundingLabels');
  const { toast } = useToast();
  const [endCurrentContract, setEndCurrentContract] = useState(true);

  const {
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    control,
    setError,
    clearErrors,
    getValues,
    formState: { errors },
  } = useForm<ChildContractFormData>({
    ...validationTiming,
    resolver: zodResolver(childContractSchema),
    defaultValues: {
      from: '',
      to: '',
      section_id: 0,
      properties: undefined,
    },
  });

  const unmapped = useProblemFormErrors(submitError, { setError, clearErrors, getValues });

  const contractFromDate = watch('from');
  const contractToDate = watch('to');

  const { fundingAttributes, attributesByKey, defaultProperties } = useFundingAttributes(
    orgId,
    contractFromDate,
    contractToDate
  );

  const activeContract = child ? getActiveContract(child.contracts) : null;

  // Track whether default funding properties have been applied for this dialog
  // session, so a fresh defaultProperties reference can't re-trigger a reset.
  const appliedDefaultsRef = useRef(false);

  // Reset form when dialog opens with a child.
  //
  // defaultProperties must NOT be a dependency here: useFundingAttributes
  // returns a brand-new object reference every time the watched from/to fields
  // change (its useMemo is keyed on those dates and returns a fresh {} even
  // with no funding config). Depending on it would re-run reset() on every date
  // edit, silently reverting the user's start/end date, section and properties.
  // Instead, apply defaults once in a separate setValue effect below — the same
  // pattern child-create-dialog.tsx uses.
  useEffect(() => {
    if (open && child) {
      setEndCurrentContract(true);
      appliedDefaultsRef.current = false;

      const birthdate = formatDateForInput(child.birthdate);
      const suggestedTo =
        birthdate && orgState ? calculateContractEndDate(birthdate, orgState) || '' : '';

      const active = getActiveContract(child.contracts);
      if (active) {
        const tomorrow = new Date();
        tomorrow.setDate(tomorrow.getDate() + 1);
        const tomorrowStr = toLocalDateString(tomorrow);

        reset({
          from: tomorrowStr,
          to: suggestedTo,
          section_id: active.section_id,
          properties: active.properties as Record<string, string> | undefined,
        });
        // An active contract supplies its own properties; don't let the
        // defaults effect overwrite them.
        appliedDefaultsRef.current = true;
      } else {
        reset({ from: '', to: suggestedTo, section_id: 0, properties: undefined });
      }
    }
  }, [open, child, orgState, reset]);

  // Apply default funding properties once when they become available, without
  // resetting the form (which would clobber user-entered dates/section).
  useEffect(() => {
    if (open && !appliedDefaultsRef.current && Object.keys(defaultProperties).length > 0) {
      appliedDefaultsRef.current = true;
      setValue('properties', defaultProperties);
    }
  }, [open, defaultProperties, setValue]);

  const handleFormSubmit = useCallback(
    (data: ChildContractFormData) => {
      if (!child) return;

      // Validate contract start date is not before birthdate
      const childBirthdate = formatDateForInput(child.birthdate);
      if (childBirthdate && data.from && isDateBefore(data.from, childBirthdate)) {
        toast({
          title: t('common.error'),
          description: t('validation.contractBeforeBirthdate'),
          variant: 'destructive',
        });
        return;
      }

      onSubmit(data, child, endCurrentContract);
    },
    [child, endCurrentContract, onSubmit, toast, t]
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {t('contracts.newContractFor', {
              name: child ? `${child.first_name} ${child.last_name}` : '',
            })}
          </DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit(handleFormSubmit)} className="space-y-4">
          <FormErrorSummary
            errors={errors}
            unmapped={unmapped}
            labels={{
              from: t('contracts.startDate'),
              to: t('contracts.endDateOptional'),
              section_id: t('sections.title'),
              properties: t('contracts.propertiesLabel'),
            }}
          />
          {activeContract && (
            <Alert>
              <AlertDescription className="space-y-3">
                <p className="font-medium">{t('contracts.hasActiveContract')}</p>
                <p className="text-muted-foreground text-sm">
                  {t('contracts.activeSince', {
                    date: formatDate(activeContract.from),
                    attrs:
                      propertiesToLabelKeys(activeContract.properties as ContractProperties)
                        .map((k) => (tLabels.has(k) ? tLabels(k) : k.split('--').pop()))
                        .join(', ') || t('contracts.noAttributes'),
                  })}
                </p>
                <div className="flex items-center space-x-2">
                  <Checkbox
                    id="endCurrentContract"
                    checked={endCurrentContract}
                    onCheckedChange={(checked) => setEndCurrentContract(checked === true)}
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

          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="from">{t('contracts.startDate')}</Label>
              <Input id="from" type="date" aria-invalid={!!errors.from} {...register('from')} />
              {errors.from && (
                <p className="text-destructive text-sm">{t('contracts.startDateRequired')}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="to">{t('contracts.endDateOptional')}</Label>
              <Input id="to" type="date" aria-invalid={!!errors.to} {...register('to')} />
              {child && orgState && (
                <p className="text-muted-foreground text-xs">{t('children.contractEndHint')}</p>
              )}
            </div>
          </div>

          {sections.length > 0 && (
            <div className="space-y-2">
              <Label htmlFor="contract_section">{t('sections.title')} *</Label>
              <Select
                value={watch('section_id')?.toString() || ''}
                onValueChange={(value) => setValue('section_id', value ? Number(value) : 0)}
              >
                <SelectTrigger id="contract_section" aria-label={t('sections.title')}>
                  <SelectValue placeholder={t('sections.selectSection')} />
                </SelectTrigger>
                <SelectContent>
                  {sections.map((section) => (
                    <SelectItem key={section.id} value={section.id.toString()}>
                      {section.name}
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
            <Label htmlFor="properties">{t('contracts.propertiesLabel')}</Label>
            <Controller
              name="properties"
              control={control}
              render={({ field }) => (
                <PropertyTagInput
                  id="properties"
                  value={field.value as Record<string, string> | undefined}
                  onChange={field.onChange}
                  fundingAttributes={fundingAttributes}
                  attributesByKey={attributesByKey}
                  placeholder={t('contracts.propertiesPlaceholder')}
                  suggestionsLabel={t('contracts.suggestedProperties')}
                />
              )}
            />
            <p className="text-muted-foreground text-xs">{t('contracts.propertiesHelp')}</p>
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
