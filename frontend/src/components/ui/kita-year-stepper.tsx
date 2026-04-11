'use client';

import { useTranslations } from 'next-intl';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface KitaYearStepperProps {
  value: number;
  onChange: (year: number) => void;
}

/**
 * Stepper for Kita years (Aug–Jul). The value is the start year,
 * displayed as "2026/27".
 */
export function KitaYearStepper({ value, onChange }: KitaYearStepperProps) {
  const t = useTranslations('statistics');
  const label = `${value}/${String(value + 1).slice(2)}`;

  return (
    <div className="flex items-center gap-1">
      <Button
        variant="outline"
        size="icon"
        className="h-8 w-8"
        onClick={() => onChange(value - 1)}
        aria-label={t('previousYear')}
      >
        <ChevronLeft className="h-4 w-4" />
      </Button>

      <span className="min-w-[80px] text-center text-sm font-medium">{label}</span>

      <Button
        variant="outline"
        size="icon"
        className="h-8 w-8"
        onClick={() => onChange(value + 1)}
        aria-label={t('nextYear')}
      >
        <ChevronRight className="h-4 w-4" />
      </Button>
    </div>
  );
}
