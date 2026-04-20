'use client';

import { format, addDays, subDays, startOfWeek, endOfWeek } from 'date-fns';
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

  return (
    <div className="flex items-center gap-1">
      <Button
        variant="outline"
        size="icon"
        className="h-11 w-11 md:h-10 md:w-10"
        onClick={() => onChange(subDays(monday, 7))}
        aria-label={t('previousWeek')}
      >
        <ChevronLeft className="h-4 w-4" />
      </Button>

      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button variant="outline" className="text-sm font-medium md:min-w-[260px]">
            {label}
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
        className="h-11 w-11 md:h-10 md:w-10"
        onClick={() => onChange(addDays(monday, 7))}
        aria-label={t('nextWeek')}
      >
        <ChevronRight className="h-4 w-4" />
      </Button>

      <Button
        variant="ghost"
        size="sm"
        className="text-sm"
        onClick={() => onChange(startOfWeek(new Date(), { weekStartsOn: 1 }))}
      >
        {t('thisWeek')}
      </Button>
    </div>
  );
}
