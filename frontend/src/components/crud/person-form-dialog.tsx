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
import { GenderSelect } from '@/components/ui/gender-select';
import type { Gender } from '@/lib/api/types';

export interface PersonFormData {
  first_name: string;
  last_name: string;
  gender: Gender;
  birthdate: string;
}

export interface PersonFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  isEditing: boolean;
  register: UseFormRegister<PersonFormData>;
  onSubmit: (e: React.FormEvent) => void;
  errors: FieldErrors<PersonFormData>;
  watch: UseFormWatch<PersonFormData>;
  setValue: UseFormSetValue<PersonFormData>;
  isSaving: boolean;
  translationPrefix: 'children' | 'employees';
}

export function PersonFormDialog({
  open,
  onOpenChange,
  isEditing,
  register,
  onSubmit,
  errors,
  watch,
  setValue,
  isSaving,
  translationPrefix,
}: PersonFormDialogProps) {
  const t = useTranslations();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEditing ? t(`${translationPrefix}.edit`) : t(`${translationPrefix}.create`)}
          </DialogTitle>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="first_name">{t(`${translationPrefix}.firstName`)}</Label>
              <Input
                id="first_name"
                aria-invalid={!!errors.first_name}
                aria-describedby={errors.first_name ? 'first-name-error' : undefined}
                {...register('first_name')}
              />
              {errors.first_name && (
                <p id="first-name-error" className="text-destructive text-sm">
                  {t('validation.firstNameRequired')}
                </p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="last_name">{t(`${translationPrefix}.lastName`)}</Label>
              <Input
                id="last_name"
                aria-invalid={!!errors.last_name}
                aria-describedby={errors.last_name ? 'last-name-error' : undefined}
                {...register('last_name')}
              />
              {errors.last_name && (
                <p id="last-name-error" className="text-destructive text-sm">
                  {t('validation.lastNameRequired')}
                </p>
              )}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="gender">{t('gender.label')}</Label>
            <GenderSelect
              value={watch('gender')}
              onValueChange={(value: Gender) => setValue('gender', value)}
              aria-invalid={!!errors.gender}
              aria-describedby={errors.gender ? 'gender-error' : undefined}
            />
            {errors.gender && (
              <p id="gender-error" className="text-destructive text-sm">
                {t('validation.genderRequired')}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="birthdate">{t(`${translationPrefix}.birthdate`)}</Label>
            <Input
              id="birthdate"
              type="date"
              aria-invalid={!!errors.birthdate}
              aria-describedby={errors.birthdate ? 'birthdate-error' : undefined}
              {...register('birthdate')}
            />
            {errors.birthdate && (
              <p id="birthdate-error" className="text-destructive text-sm">
                {t('validation.birthdateRequired')}
              </p>
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
