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
      <SelectTrigger className="w-full md:w-[200px]">
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
