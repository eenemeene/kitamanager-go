'use client';

import { format, addDays, subDays } from 'date-fns';
import { de, enUS } from 'date-fns/locale';
import { useLocale, useTranslations } from 'next-intl';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Calendar } from '@/components/ui/calendar';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { useState } from 'react';

const dateFnsLocales: Record<string, typeof de> = {
  de: de,
  en: enUS,
};

interface DayStepperProps {
  value: Date;
  onChange: (date: Date) => void;
}

export function DayStepper({ value, onChange }: DayStepperProps) {
  const locale = useLocale();
  const t = useTranslations('common');
  const tAttendance = useTranslations('attendance');
  const [open, setOpen] = useState(false);
  const dfLocale = dateFnsLocales[locale] ?? enUS;

  return (
    <div className="flex items-center gap-1">
      <Button
        variant="outline"
        size="icon"
        onClick={() => onChange(subDays(value, 1))}
        aria-label={tAttendance('previousDay')}
      >
        <ChevronLeft className="h-4 w-4" />
      </Button>

      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            // The rendered label is localized and, below `sm`, abbreviated.
            // Tests and any other machine reader use this instead, so they do
            // not break when the presentation changes — which is exactly what
            // happened when the short form landed.
            data-testid="day-stepper-value"
            data-value={format(value, 'yyyy-MM-dd')}
            className="text-sm font-medium md:min-w-[200px]"
          >
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
              {format(value, 'EEEE, d. MMMM yyyy', { locale: dfLocale })}
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
          />
        </PopoverContent>
      </Popover>

      <Button
        variant="outline"
        size="icon"
        onClick={() => onChange(addDays(value, 1))}
        aria-label={tAttendance('nextDay')}
      >
        <ChevronRight className="h-4 w-4" />
      </Button>

      <Button variant="ghost" className="text-sm" onClick={() => onChange(new Date())}>
        {t('today')}
      </Button>
    </div>
  );
}
