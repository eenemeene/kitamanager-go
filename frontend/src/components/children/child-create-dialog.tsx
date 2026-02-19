'use client';

import { useEffect } from 'react';
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
import { GenderSelect } from '@/components/ui/gender-select';
import { PropertyTagInput } from '@/components/ui/tag-input';
import { useFundingAttributes } from '@/lib/hooks/use-funding-attributes';
import { calculateContractEndDate } from '@/lib/utils/school-enrollment';
import { childWithContractSchema, type ChildWithContractFormData } from '@/lib/schemas';
import type { Gender, Section } from '@/lib/api/types';

export interface ChildCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orgId: number;
  orgState: string | undefined;
  sections: Section[];
  isSaving: boolean;
  onSubmit: (data: ChildWithContractFormData) => void;
}

export function ChildCreateDialog({
  open,
  onOpenChange,
  orgId,
  orgState,
  sections,
  isSaving,
  onSubmit,
}: ChildCreateDialogProps) {
  const t = useTranslations();

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    control,
    formState: { errors },
  } = useForm<ChildWithContractFormData>({
    resolver: zodResolver(childWithContractSchema),
    defaultValues: {
      first_name: '',
      last_name: '',
      gender: 'male',
      birthdate: '',
      contract_from: '',
      contract_to: '',
      section_id: 0,
      properties: undefined,
    },
  });

  const birthdate = watch('birthdate');
  const contractFrom = watch('contract_from');
  const contractTo = watch('contract_to');

  // Auto-fill contract end date based on birthdate + org state
  useEffect(() => {
    if (birthdate && orgState) {
      const suggestedEnd = calculateContractEndDate(birthdate, orgState);
      if (suggestedEnd) {
        setValue('contract_to', suggestedEnd);
      }
    }
  }, [birthdate, orgState, setValue]);

  // Reset form when dialog opens
  useEffect(() => {
    if (open) {
      reset({
        first_name: '',
        last_name: '',
        gender: 'male',
        birthdate: '',
        contract_from: '',
        contract_to: '',
        section_id: 0,
        properties: undefined,
      });
    }
  }, [open, reset]);

  const { fundingAttributes, attributesByKey } = useFundingAttributes(
    orgId,
    contractFrom,
    contractTo
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('children.create')}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="create_first_name">{t('children.firstName')}</Label>
              <Input id="create_first_name" {...register('first_name')} />
              {errors.first_name && (
                <p className="text-destructive text-sm">{t('validation.firstNameRequired')}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="create_last_name">{t('children.lastName')}</Label>
              <Input id="create_last_name" {...register('last_name')} />
              {errors.last_name && (
                <p className="text-destructive text-sm">{t('validation.lastNameRequired')}</p>
              )}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="create_gender">{t('gender.label')}</Label>
            <GenderSelect
              value={watch('gender')}
              onValueChange={(value: Gender) => setValue('gender', value)}
            />
            {errors.gender && (
              <p className="text-destructive text-sm">{t('validation.genderRequired')}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="create_birthdate">{t('children.birthdate')}</Label>
            <Input id="create_birthdate" type="date" {...register('birthdate')} />
            {errors.birthdate && (
              <p className="text-destructive text-sm">{t('validation.birthdateRequired')}</p>
            )}
          </div>

          <div className="border-t pt-4">
            <h4 className="mb-3 text-sm font-medium">{t('children.initialContract')}</h4>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="create_contract_from">{t('contracts.startDate')}</Label>
                <Input id="create_contract_from" type="date" {...register('contract_from')} />
                {errors.contract_from && (
                  <p className="text-destructive text-sm">
                    {errors.contract_from.type === 'custom'
                      ? t('validation.contractBeforeBirthdate')
                      : t('contracts.startDateRequired')}
                  </p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="create_contract_to">{t('contracts.endDateOptional')}</Label>
                <Input id="create_contract_to" type="date" {...register('contract_to')} />
                {birthdate && orgState && (
                  <p className="text-muted-foreground text-xs">{t('children.contractEndHint')}</p>
                )}
              </div>
            </div>

            {sections.length > 0 && (
              <div className="mt-4 space-y-2">
                <Label htmlFor="create_section">{t('sections.title')} *</Label>
                <Select
                  value={watch('section_id')?.toString() || ''}
                  onValueChange={(value) => setValue('section_id', value ? Number(value) : 0)}
                >
                  <SelectTrigger id="create_section">
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

            <div className="mt-4 space-y-2">
              <Label htmlFor="create_properties">{t('contracts.propertiesLabel')}</Label>
              <Controller
                name="properties"
                control={control}
                render={({ field }) => (
                  <PropertyTagInput
                    id="create_properties"
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
