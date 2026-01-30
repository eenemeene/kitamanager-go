'use client';

import { ReactNode } from 'react';
import { useTranslations } from 'next-intl';
import { Pencil, Trash2, Eye } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';

export interface Column<TItem> {
  /** Unique key for the column */
  key: string;
  /** i18n key for the header, or a string to display directly */
  header: string;
  /** Function to render the cell content */
  render: (item: TItem) => ReactNode;
  /** Optional class name for the cell */
  className?: string;
  /** Optional class name for the header */
  headerClassName?: string;
}

export interface ResourceTableProps<TItem> {
  /** Array of items to display */
  items: TItem[] | undefined;
  /** Column definitions */
  columns: Column<TItem>[];
  /** Function to get a unique key for each item */
  getItemKey: (item: TItem) => string | number;
  /** Whether the data is loading */
  isLoading?: boolean;
  /** Number of skeleton rows to show while loading */
  skeletonRows?: number;
  /** Handler for edit action (omit to hide edit button) */
  onEdit?: (item: TItem) => void;
  /** Handler for delete action (omit to hide delete button) */
  onDelete?: (item: TItem) => void;
  /** Handler for view action (omit to hide view button) */
  onView?: (item: TItem) => void;
  /** Custom action buttons renderer */
  renderActions?: (item: TItem) => ReactNode;
  /** Whether to show the actions column */
  showActions?: boolean;
  /** Whether action buttons are disabled */
  actionsDisabled?: boolean;
}

/**
 * Reusable table component for displaying resource lists.
 * Includes built-in loading skeleton, empty state, and action buttons.
 */
export function ResourceTable<TItem>({
  items,
  columns,
  getItemKey,
  isLoading = false,
  skeletonRows = 3,
  onEdit,
  onDelete,
  onView,
  renderActions,
  showActions = true,
  actionsDisabled = false,
}: ResourceTableProps<TItem>) {
  const t = useTranslations();

  const hasActions = showActions && (onEdit || onDelete || onView || renderActions);
  const totalColumns = columns.length + (hasActions ? 1 : 0);

  if (isLoading) {
    return (
      <div className="space-y-2">
        {[...Array(skeletonRows)].map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          {columns.map((column) => (
            <TableHead key={column.key} className={column.headerClassName}>
              {column.header.includes('.') ? t(column.header) : column.header}
            </TableHead>
          ))}
          {hasActions && <TableHead className="text-right">{t('common.actions')}</TableHead>}
        </TableRow>
      </TableHeader>
      <TableBody>
        {items?.map((item) => (
          <TableRow key={getItemKey(item)}>
            {columns.map((column) => (
              <TableCell key={column.key} className={column.className}>
                {column.render(item)}
              </TableCell>
            ))}
            {hasActions && (
              <TableCell className="text-right">
                {renderActions ? (
                  renderActions(item)
                ) : (
                  <>
                    {onView && (
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => onView(item)}
                        disabled={actionsDisabled}
                      >
                        <Eye className="h-4 w-4" />
                      </Button>
                    )}
                    {onEdit && (
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => onEdit(item)}
                        disabled={actionsDisabled}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                    )}
                    {onDelete && (
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => onDelete(item)}
                        disabled={actionsDisabled}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    )}
                  </>
                )}
              </TableCell>
            )}
          </TableRow>
        ))}
        {(!items || items.length === 0) && (
          <TableRow>
            <TableCell colSpan={totalColumns} className="text-center text-muted-foreground">
              {t('common.noResults')}
            </TableCell>
          </TableRow>
        )}
      </TableBody>
    </Table>
  );
}
