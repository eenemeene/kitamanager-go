'use client';

import { format, addMonths, subMonths, addYears, subYears, startOfMonth } from 'date-fns';
import { de, enUS } from 'date-fns/locale';
import { useLocale, useTranslations } from 'next-intl';
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Calendar } from '@/components/ui/calendar';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { useState } from 'react';

const dateFnsLocales: Record<string, typeof de> = {
  de: de,
  en: enUS,
};

// yearRangeOffsets bounds the calendar popover's year dropdown. Ten
// years back covers historical contract lookups (a long-tenured
// employee's first contract); five years forward covers planning
// scenarios (next year's pay plan, upcoming contracts). Wider than
// that and the dropdown becomes a scroll-jungle for no real use.
const yearRangeBackOffset = 10;
const yearRangeForwardOffset = 5;

interface MonthStepperProps {
  value: Date;
  onChange: (date: Date) => void;
}

export function MonthStepper({ value, onChange }: MonthStepperProps) {
  const locale = useLocale();
  const t = useTranslations('common');
  const [open, setOpen] = useState(false);
  const dfLocale = dateFnsLocales[locale] ?? enUS;

  // Year-range bounds for the calendar popover's dropdown caption.
  // Anchored on today (not `value`) so the bounds don't drift when
  // the user navigates through history — every time the popover
  // opens the dropdown spans the same window.
  const today = new Date();
  const startMonth = startOfMonth(subYears(today, yearRangeBackOffset));
  const endMonth = startOfMonth(addYears(today, yearRangeForwardOffset));

  return (
    <div className="flex items-center gap-1">
      <Button
        variant="outline"
        size="icon"
        onClick={() => onChange(startOfMonth(subYears(value, 1)))}
        aria-label={t('previousYear')}
      >
        <ChevronsLeft className="h-4 w-4" />
      </Button>
      <Button
        variant="outline"
        size="icon"
        onClick={() => onChange(startOfMonth(subMonths(value, 1)))}
        aria-label={t('previousMonth')}
      >
        <ChevronLeft className="h-4 w-4" />
      </Button>

      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button variant="outline" className="text-sm font-medium md:min-w-[180px]">
            {/* Two renderings, toggled by CSS rather than a media-query hook, so
              there is no hydration mismatch and no layout shift after mount.
              The long form is what a desktop user reads; the short one exists
              because this row cannot otherwise fit a 412px phone — six controls
              at 44px touch targets plus this label came to 419px, which pushed
              mobile Chrome to widen the layout viewport and gave every page a
              horizontal scroll. Nothing is hidden: the same date, written
              shorter. */}
            <span className="sm:hidden">{format(value, 'P', { locale: dfLocale })}</span>
            <span className="hidden sm:inline">
              {format(value, 'd. MMMM yyyy', { locale: dfLocale })}
            </span>
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0" align="center">
          <Calendar
            mode="single"
            selected={value}
            onSelect={(date) => {
              if (date) {
                onChange(date);
                setOpen(false);
              }
            }}
            defaultMonth={value}
            // Dropdown caption lets the user jump directly to any
            // month/year in the bounded range — the keyboard-free
            // path to "show me 2028". The chevron buttons outside the
            // popover stay for touch-friendly ±1 month / year stepping.
            captionLayout="dropdown"
            startMonth={startMonth}
            endMonth={endMonth}
          />
        </PopoverContent>
      </Popover>

      <Button
        variant="outline"
        size="icon"
        onClick={() => onChange(startOfMonth(addMonths(value, 1)))}
        aria-label={t('nextMonth')}
      >
        <ChevronRight className="h-4 w-4" />
      </Button>
      <Button
        variant="outline"
        size="icon"
        onClick={() => onChange(startOfMonth(addYears(value, 1)))}
        aria-label={t('nextYear')}
      >
        <ChevronsRight className="h-4 w-4" />
      </Button>

      <Button variant="ghost" className="text-sm" onClick={() => onChange(new Date())}>
        {t('today')}
      </Button>
    </div>
  );
}
