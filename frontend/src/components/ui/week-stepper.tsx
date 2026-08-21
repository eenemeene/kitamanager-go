'use client';

import { format, addDays, subDays, startOfWeek, endOfWeek } from 'date-fns';
import { de, enUS } from 'date-fns/locale';
import { useLocale, useTranslations } from 'next-intl';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Calendar } from '@/components/ui/calendar';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { useState } from 'react';
import { todayBerlinDate } from '@/lib/utils/contracts';

const dateFnsLocales: Record<string, typeof de> = {
  de: de,
  en: enUS,
};

interface WeekStepperProps {
  value: Date;
  onChange: (date: Date) => void;
}

export function WeekStepper({ value, onChange }: WeekStepperProps) {
  const locale = useLocale();
  const t = useTranslations('attendance');
  const [open, setOpen] = useState(false);
  const dfLocale = dateFnsLocales[locale] ?? enUS;

  const monday = startOfWeek(value, { weekStartsOn: 1 });
  const friday = addDays(monday, 4);

  const label = `${format(monday, 'EEE dd.MM', { locale: dfLocale })} – ${format(friday, 'EEE dd.MM yyyy', { locale: dfLocale })}`;
  // Same week, without the weekday names and the year, for a phone-width row.
  const shortLabel = `${format(monday, 'dd.MM', { locale: dfLocale })} – ${format(friday, 'dd.MM', { locale: dfLocale })}`;

  return (
    <div className="flex items-center gap-1">
      <Button
        variant="outline"
        size="icon"
        onClick={() => onChange(subDays(monday, 7))}
        aria-label={t('previousWeek')}
      >
        <ChevronLeft className="h-4 w-4" />
      </Button>

      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            // See month-stepper: the label is localized and abbreviated on
            // small screens, so the machine-readable value lives here.
            data-testid="week-stepper-value"
            data-value={format(monday, 'yyyy-MM-dd')}
            className="text-sm font-medium md:min-w-[260px]"
          >
            {/* Two renderings, toggled by CSS rather than a media-query hook, so
              there is no hydration mismatch and no layout shift after mount.
              The long form is what a desktop user reads; the short one exists
              because this row cannot otherwise fit a 412px phone — six controls
              at 44px touch targets plus this label came to 419px, which pushed
              mobile Chrome to widen the layout viewport and gave every page a
              horizontal scroll. Nothing is hidden: the same date, written
              shorter. */}
            <span className="sm:hidden">{shortLabel}</span>
            <span className="hidden sm:inline">{label}</span>
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0" align="center">
          <Calendar
            mode="single"
            selected={value}
            onSelect={(date) => {
              if (date) {
                onChange(startOfWeek(date, { weekStartsOn: 1 }));
                setOpen(false);
              }
            }}
            defaultMonth={value}
          />
        </PopoverContent>
      </Popover>

      <Button
        variant="outline"
        size="icon"
        onClick={() => onChange(addDays(monday, 7))}
        aria-label={t('nextWeek')}
      >
        <ChevronRight className="h-4 w-4" />
      </Button>

      <Button
        variant="ghost"
        className="text-sm"
        onClick={() => onChange(startOfWeek(todayBerlinDate(), { weekStartsOn: 1 }))}
      >
        {t('thisWeek')}
      </Button>
    </div>
  );
}
