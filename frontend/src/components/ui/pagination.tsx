'use client';

import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useTranslations } from 'next-intl';

interface PaginationProps {
  page: number;
  totalPages: number;
  total: number;
  limit: number;
  onPageChange: (page: number) => void;
  isLoading?: boolean;
}

export function Pagination({
  page,
  totalPages,
  total,
  limit,
  onPageChange,
  isLoading,
}: PaginationProps) {
  const t = useTranslations();

  if (totalPages <= 1) {
    return null;
  }

  const startItem = (page - 1) * limit + 1;
  const endItem = Math.min(page * limit, total);

  return (
    <div className="flex items-center justify-between px-2 py-4">
      <div className="text-sm text-muted-foreground">
        {t('pagination.showing', { start: startItem, end: endItem, total })}
      </div>
      <div className="flex items-center gap-1">
        <Button
          variant="outline"
          size="icon"
          onClick={() => onPageChange(1)}
          disabled={page <= 1 || isLoading}
          title={t('pagination.firstPage')}
        >
          <ChevronsLeft className="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="icon"
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1 || isLoading}
          title={t('pagination.previousPage')}
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>
        <span className="px-3 text-sm">{t('pagination.pageOf', { page, totalPages })}</span>
        <Button
          variant="outline"
          size="icon"
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages || isLoading}
          title={t('pagination.nextPage')}
        >
          <ChevronRight className="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="icon"
          onClick={() => onPageChange(totalPages)}
          disabled={page >= totalPages || isLoading}
          title={t('pagination.lastPage')}
        >
          <ChevronsRight className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
