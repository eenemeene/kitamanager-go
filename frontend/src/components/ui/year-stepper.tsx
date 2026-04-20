'use client';

import { useTranslations } from 'next-intl';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface YearStepperProps {
  value: number;
  onChange: (year: number) => void;
}

export function YearStepper({ value, onChange }: YearStepperProps) {
  const t = useTranslations('statistics');

  return (
    <div className="flex items-center gap-1">
      <Button
        variant="outline"
        size="icon"
        className="h-11 w-11 md:h-10 md:w-10"
        onClick={() => onChange(value - 1)}
        aria-label={t('previousYear')}
      >
        <ChevronLeft className="h-4 w-4" />
      </Button>

      <span className="min-w-[60px] text-center text-sm font-medium">{value}</span>

      <Button
        variant="outline"
        size="icon"
        className="h-11 w-11 md:h-10 md:w-10"
        onClick={() => onChange(value + 1)}
        aria-label={t('nextYear')}
      >
        <ChevronRight className="h-4 w-4" />
      </Button>
    </div>
  );
}
