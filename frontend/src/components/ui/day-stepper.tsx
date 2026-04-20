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
        className="h-11 w-11 md:h-10 md:w-10"
        onClick={() => onChange(subDays(value, 1))}
        aria-label={tAttendance('previousDay')}
      >
        <ChevronLeft className="h-4 w-4" />
      </Button>

      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button variant="outline" className="text-sm font-medium md:min-w-[200px]">
            {format(value, 'EEEE, d. MMMM yyyy', { locale: dfLocale })}
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
        className="h-11 w-11 md:h-10 md:w-10"
        onClick={() => onChange(addDays(value, 1))}
        aria-label={tAttendance('nextDay')}
      >
        <ChevronRight className="h-4 w-4" />
      </Button>

      <Button variant="ghost" size="sm" className="text-sm" onClick={() => onChange(new Date())}>
        {t('today')}
      </Button>
    </div>
  );
}
