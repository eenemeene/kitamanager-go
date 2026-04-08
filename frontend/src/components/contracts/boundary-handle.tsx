'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { CalendarIcon } from 'lucide-react';
import { parseISO, addDays } from 'date-fns';
import { formatDate } from '@/lib/utils/formatting';
import { Calendar } from '@/components/ui/calendar';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import type { BaseContract } from './timeline-utils';

interface BoundaryHandleProps {
  upperContract: BaseContract;
  lowerContract: BaseContract;
  minDate: Date;
  maxDate: Date | null;
  onBoundaryChange: (newTo: string, newFrom: string) => void;
  isUpdating?: boolean;
}

function formatISODate(date: Date): string {
  const y = date.getFullYear();
  const m = (date.getMonth() + 1).toString().padStart(2, '0');
  const d = date.getDate().toString().padStart(2, '0');
  return `${y}-${m}-${d}T00:00:00Z`;
}

export function BoundaryHandle({
  upperContract,
  lowerContract,
  minDate,
  maxDate,
  onBoundaryChange,
  isUpdating,
}: BoundaryHandleProps) {
  const t = useTranslations();
  const [open, setOpen] = useState(false);

  const endDate = lowerContract.to || '';
  const startDate = upperContract.from;

  const currentBoundary = endDate ? parseISO(endDate) : new Date();

  const handleSelect = (date: Date | undefined) => {
    if (!date) return;
    const newTo = formatISODate(date);
    const newFrom = formatISODate(addDays(date, 1));
    setOpen(false);
    onBoundaryChange(newTo, newFrom);
  };

  return (
    <div className="relative flex gap-4 pl-3">
      {/* Timeline connector dot */}
      <div className="relative flex w-7 shrink-0 items-center justify-center">
        <div className="bg-primary/60 z-10 h-2 w-2 rounded-full ring-2 ring-white dark:ring-gray-950" />
      </div>
      {/* Handle */}
      <div className="min-w-0 flex-1">
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <button
              data-testid="boundary-handle"
              type="button"
              disabled={isUpdating}
              aria-label={t('timeline.clickToAdjust')}
              className={`flex w-full cursor-pointer items-center justify-center rounded-md border-2 border-dashed py-1.5 transition-colors select-none ${
                open
                  ? 'border-primary bg-primary/10'
                  : 'hover:border-primary hover:bg-primary/5 border-muted-foreground/30'
              } ${isUpdating ? 'pointer-events-none opacity-60' : ''}`}
            >
              <CalendarIcon className="text-muted-foreground h-3.5 w-3.5" />
              <span className="text-muted-foreground ml-2 text-xs font-medium">
                {formatDate(endDate)} | {formatDate(startDate)}
              </span>
            </button>
          </PopoverTrigger>
          <PopoverContent className="w-auto p-0" align="center">
            <Calendar
              mode="single"
              selected={currentBoundary}
              onSelect={handleSelect}
              defaultMonth={currentBoundary}
              disabled={(date) => {
                if (date < minDate) return true;
                if (maxDate && date > maxDate) return true;
                return false;
              }}
            />
          </PopoverContent>
        </Popover>
      </div>
    </div>
  );
}
