'use client';

import { Search } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { Input } from '@/components/ui/input';

export interface SearchInputProps {
  id: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}

export function SearchInput({ id, value, onChange, placeholder }: SearchInputProps) {
  const t = useTranslations();

  return (
    <div className="relative max-w-sm">
      <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
      <label htmlFor={id} className="sr-only">
        {placeholder ?? t('common.search')}
      </label>
      <Input
        id={id}
        placeholder={placeholder ?? t('common.search')}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="pl-9"
      />
    </div>
  );
}
