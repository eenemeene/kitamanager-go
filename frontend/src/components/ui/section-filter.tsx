'use client';

import { useTranslations } from 'next-intl';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { Section } from '@/lib/api/types';

interface SectionFilterProps {
  sections: Section[];
  value: number | undefined;
  onChange: (sectionId: number | undefined) => void;
}

export function SectionFilter({ sections, value, onChange }: SectionFilterProps) {
  const t = useTranslations();

  return (
    <Select
      value={value?.toString() ?? 'all'}
      onValueChange={(v) => onChange(v === 'all' ? undefined : Number(v))}
    >
      {/* A placeholder is not an accessible name, and it disappears the moment
          a value is chosen. Filter controls sit in a bar with no visible label,
          so the name has to be on the trigger itself. */}
      <SelectTrigger aria-label={t('statistics.filterBySection')} className="w-full md:w-[200px]">
        <SelectValue placeholder={t('statistics.filterBySection')} />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="all">{t('statistics.allSections')}</SelectItem>
        {sections.map((section) => (
          <SelectItem key={section.id} value={String(section.id)}>
            {section.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
