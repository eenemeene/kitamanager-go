'use client';

import { useTranslations } from 'next-intl';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { Gender } from '@/lib/api/types';

export interface GenderSelectProps {
  value: Gender;
  onValueChange: (value: Gender) => void;
  /**
   * Goes on the trigger, which is the element that takes focus — so a
   * `<Label htmlFor>` beside this actually resolves. Without it the label
   * pointed at nothing and the select was announced with only its value.
   */
  id?: string;
  'aria-invalid'?: boolean;
  'aria-describedby'?: string;
}

export function GenderSelect({
  value,
  onValueChange,
  id,
  'aria-invalid': ariaInvalid,
  'aria-describedby': ariaDescribedBy,
}: GenderSelectProps) {
  const t = useTranslations();

  return (
    <Select value={value} onValueChange={onValueChange}>
      <SelectTrigger id={id} aria-invalid={ariaInvalid} aria-describedby={ariaDescribedBy}>
        <SelectValue placeholder={t('gender.selectGender')} />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="male">{t('gender.male')}</SelectItem>
        <SelectItem value="female">{t('gender.female')}</SelectItem>
        <SelectItem value="diverse">{t('gender.diverse')}</SelectItem>
      </SelectContent>
    </Select>
  );
}
