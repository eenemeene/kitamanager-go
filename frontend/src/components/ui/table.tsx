'use client';

import * as React from 'react';
import { useTranslations } from 'next-intl';

import { cn } from '@/lib/utils';

interface TableProps extends React.HTMLAttributes<HTMLTableElement> {
  /**
   * Names the scroll region for whoever reaches it by keyboard. Pass the
   * heading the table already sits under.
   *
   * A page with more than one table needs a distinct name per table, or it
   * gets two landmarks a screen-reader user cannot tell apart. Omitting it
   * falls back to a generic label, which is right for the single-table pages
   * that make up most of the app.
   */
  scrollRegionLabel?: string;
}

const Table = React.forwardRef<HTMLTableElement, TableProps>(
  ({ className, scrollRegionLabel, ...props }, ref) => {
    const t = useTranslations('common');
    return (
      // A scroll container with no focusable child cannot be scrolled by
      // keyboard (WCAG 2.1.1) -- the columns hidden off the right edge on a
      // phone were reachable by touch drag and by nothing else. tabIndex makes
      // it a tab stop, which in turn obliges it to show focus, hence the ring.
      //
      // A tab stop also has to say what it is. With no role and no name this
      // one announced nothing: you tabbed into silence and had to guess that
      // the arrow keys now scrolled a table. `region` is the role axe accepts
      // for a focusable scrollable area and the one the APG recommends --
      // `group` reads like the right answer and is not in axe's list.
      <div
        role="region"
        aria-label={scrollRegionLabel ?? t('tableScrollRegion')}
        tabIndex={0}
        className="focus-visible:ring-ring relative w-full overflow-auto focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none"
      >
        <table ref={ref} className={cn('w-full caption-bottom text-sm', className)} {...props} />
      </div>
    );
  }
);
Table.displayName = 'Table';

const TableHeader = React.forwardRef<
  HTMLTableSectionElement,
  React.HTMLAttributes<HTMLTableSectionElement>
>(({ className, ...props }, ref) => (
  <thead ref={ref} className={cn('[&_tr]:border-b', className)} {...props} />
));
TableHeader.displayName = 'TableHeader';

const TableBody = React.forwardRef<
  HTMLTableSectionElement,
  React.HTMLAttributes<HTMLTableSectionElement>
>(({ className, ...props }, ref) => (
  <tbody ref={ref} className={cn('[&_tr:last-child]:border-0', className)} {...props} />
));
TableBody.displayName = 'TableBody';

const TableFooter = React.forwardRef<
  HTMLTableSectionElement,
  React.HTMLAttributes<HTMLTableSectionElement>
>(({ className, ...props }, ref) => (
  <tfoot
    ref={ref}
    className={cn('bg-muted/50 border-t font-medium [&>tr]:last:border-b-0', className)}
    {...props}
  />
));
TableFooter.displayName = 'TableFooter';

const TableRow = React.forwardRef<HTMLTableRowElement, React.HTMLAttributes<HTMLTableRowElement>>(
  ({ className, ...props }, ref) => (
    <tr
      ref={ref}
      className={cn(
        'hover:bg-muted/50 data-[state=selected]:bg-muted border-b transition-colors',
        className
      )}
      {...props}
    />
  )
);
TableRow.displayName = 'TableRow';

const TableHead = React.forwardRef<
  HTMLTableCellElement,
  React.ThHTMLAttributes<HTMLTableCellElement>
>(({ className, ...props }, ref) => (
  <th
    ref={ref}
    className={cn(
      'text-muted-foreground h-9 px-3 text-left align-middle text-xs font-medium [&:has([role=checkbox])]:pr-0',
      className
    )}
    {...props}
  />
));
TableHead.displayName = 'TableHead';

const TableCell = React.forwardRef<
  HTMLTableCellElement,
  React.TdHTMLAttributes<HTMLTableCellElement>
>(({ className, ...props }, ref) => (
  <td
    ref={ref}
    className={cn('px-3 py-2 align-middle text-sm [&:has([role=checkbox])]:pr-0', className)}
    {...props}
  />
));
TableCell.displayName = 'TableCell';

const TableCaption = React.forwardRef<
  HTMLTableCaptionElement,
  React.HTMLAttributes<HTMLTableCaptionElement>
>(({ className, ...props }, ref) => (
  <caption ref={ref} className={cn('text-muted-foreground mt-4 text-sm', className)} {...props} />
));
TableCaption.displayName = 'TableCaption';

export { Table, TableHeader, TableBody, TableFooter, TableHead, TableRow, TableCell, TableCaption };
