'use client';

import type { LucideIcon } from 'lucide-react';
import { MoreHorizontal } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

export interface RowAction {
  /** Stable key, also used as the React key. */
  key: string;
  label: string;
  icon: LucideIcon;
  onSelect: () => void;
  disabled?: boolean;
}

export interface RowActionsMenuProps {
  actions: RowAction[];
  /** Accessible name for the trigger, e.g. "Actions for Emma Schmidt". */
  label: string;
  className?: string;
}

/**
 * The row actions a narrow table cannot show inline, as a menu.
 *
 * A wide table shows every action as its own icon button and hides the
 * secondary ones below `sm`, which is the right call for the layout and was the
 * wrong one for the product: the pages those buttons lead to — a child's
 * contract history, their vouchers, an employee's contracts — had no other
 * route into them anywhere in the app. On a phone they did not exist. Adding a
 * contract to an existing child or employee was simply not possible.
 *
 * So this is not a convenience. It is the only door, and it is rendered `sm:hidden`
 * beside the inline buttons that take over above that width.
 */
export function RowActionsMenu({ actions, label, className }: RowActionsMenuProps) {
  if (actions.length === 0) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" aria-label={label} className={className}>
          <MoreHorizontal className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      {/* `align="end"` keeps the panel inside the viewport when the trigger sits
          in a right-aligned action column on a 375px screen. */}
      <DropdownMenuContent align="end" className="w-56">
        {actions.map(({ key, label: itemLabel, icon: Icon, onSelect, disabled }) => (
          <DropdownMenuItem key={key} onSelect={onSelect} disabled={disabled}>
            <Icon className="mr-2 h-4 w-4" aria-hidden="true" />
            {itemLabel}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
