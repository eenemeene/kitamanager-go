'use client';

import { useEffect, useState, useCallback } from 'react';
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
import { formatDate, formatDateForInput, propertiesToValues } from '@/lib/utils/formatting';
import { getActiveContract, isDateBefore } from '@/lib/utils/contracts';
import { calculateContractEndDate } from '@/lib/utils/school-enrollment';
import type { Child, Section, ContractProperties } from '@/lib/api/types';

export interface ChildContractCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orgId: number;
  orgState: string | undefined;
  child: Child | null;
  sections: Section[];
  isSaving: boolean;
  onSubmit: (data: ChildContractFormData, child: Child, endCurrentContract: boolean) => void;
}

export function ChildContractCreateDialog({
  open,
  onOpenChange,
  orgId,
  orgState,
  child,
  sections,
  isSaving,
  onSubmit,
}: ChildContractCreateDialogProps) {
  const t = useTranslations();
  const { toast } = useToast();
  const [endCurrentContract, setEndCurrentContract] = useState(true);

  const {
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    control,
    formState: { errors },
  } = useForm<ChildContractFormData>({
    resolver: zodResolver(childContractSchema),
    defaultValues: {
      from: '',
      to: '',
      section_id: 0,
      properties: undefined,
    },
  });

  const contractFromDate = watch('from');
  const contractToDate = watch('to');

  const { fundingAttributes, attributesByKey } = useFundingAttributes(
    orgId,
    contractFromDate,
    contractToDate
  );

  const activeContract = child ? getActiveContract(child.contracts) : null;

  // Reset form when dialog opens with a child
  useEffect(() => {
    if (open && child) {
      setEndCurrentContract(true);

      const birthdate = formatDateForInput(child.birthdate);
      const suggestedTo =
        birthdate && orgState ? calculateContractEndDate(birthdate, orgState) || '' : '';

      const active = getActiveContract(child.contracts);
      if (active) {
        const tomorrow = new Date();
        tomorrow.setDate(tomorrow.getDate() + 1);
        const tomorrowStr = tomorrow.toISOString().split('T')[0];

        reset({
          from: tomorrowStr,
          to: suggestedTo,
          section_id: active.section_id,
          properties: active.properties as Record<string, string> | undefined,
        });
      } else {
        reset({ from: '', to: suggestedTo, section_id: 0, properties: undefined });
      }
    }
  }, [open, child, orgState, reset]);

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
          {activeContract && (
            <Alert>
              <AlertDescription className="space-y-3">
                <p className="font-medium">{t('contracts.hasActiveContract')}</p>
                <p className="text-sm text-muted-foreground">
                  {t('contracts.activeSince', {
                    date: formatDate(activeContract.from),
                    attrs:
                      propertiesToValues(activeContract.properties as ContractProperties).join(
                        ', '
                      ) || t('contracts.noAttributes'),
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
                    className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
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
                <p className="text-sm text-destructive">{t('contracts.startDateRequired')}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="to">{t('contracts.endDateOptional')}</Label>
              <Input id="to" type="date" {...register('to')} />
              {child && orgState && (
                <p className="text-xs text-muted-foreground">{t('children.contractEndHint')}</p>
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
                <p className="text-sm text-destructive">{t('validation.sectionRequired')}</p>
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
                  value={field.value}
                  onChange={field.onChange}
                  fundingAttributes={fundingAttributes}
                  attributesByKey={attributesByKey}
                  placeholder={t('contracts.propertiesPlaceholder')}
                  suggestionsLabel={t('contracts.suggestedProperties')}
                />
              )}
            />
            <p className="text-xs text-muted-foreground">{t('contracts.propertiesHelp')}</p>
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
