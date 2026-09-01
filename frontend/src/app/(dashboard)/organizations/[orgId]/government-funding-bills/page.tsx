'use client';

import { useMemo, useState, useRef } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Upload, Trash2, Eye, CheckCircle2, XCircle, AlertTriangle } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { TooltipProvider } from '@/components/ui/tooltip';
import { HeaderWithTooltip } from '@/components/ui/header-with-tooltip';
import { KitaYearStepper } from '@/components/ui/kita-year-stepper';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { apiClient, getErrorMessage } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import type {
  GovernmentFundingBillPeriodListItem,
  FundingComparisonResponse,
} from '@/lib/api/types';
import { useToast } from '@/lib/hooks/use-toast';
import { useDebouncedValue } from '@/lib/hooks/use-debounced-value';

import { QueryError } from '@/components/crud/query-error';
import { SearchInput } from '@/components/ui/search-input';
import { useFormatters } from '@/hooks/use-formatters';

/** Return the kita-year start year for a given date string (YYYY-MM-DD).
 *  Kita year runs Aug 1 – Jul 31. e.g. 2025-08-01 → 2025, 2026-03-01 → 2025 */
function kitaYearForDate(dateStr: string): number {
  const d = new Date(dateStr);
  return d.getMonth() >= 7 ? d.getFullYear() : d.getFullYear() - 1;
}

export default function GovernmentFundingBillsPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations('governmentFundingBills');
  const fmt = useFormatters();
  const tCommon = useTranslations('common');
  const tStats = useTranslations('statistics');
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<GovernmentFundingBillPeriodListItem | null>(
    null
  );

  // Search is server-side: matches facility name, child name, or voucher number.
  const [searchInput, setSearchInput] = useState('');
  const search = useDebouncedValue(searchInput, 300);

  // Kita year filter — default to current kita year
  const now = new Date();
  const currentKitaYear = now.getMonth() >= 7 ? now.getFullYear() : now.getFullYear() - 1;
  const [kitaYear, setKitaYear] = useState(currentKitaYear);
  const kitaYearFrom = `${kitaYear}-08-01`;
  const kitaYearTo = `${kitaYear + 1}-07-31`;

  const {
    data: billPeriods,
    isLoading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: queryKeys.governmentFundingBillPeriods.list(orgId, 1, search || undefined),
    queryFn: () =>
      apiClient.getGovernmentFundingBillPeriods(orgId, { limit: 100, search: search || undefined }),
  });

  // Server already filtered by search; client narrows to the selected kita year.
  const filteredItems = useMemo(() => {
    const all = billPeriods?.data ?? [];
    return all.filter((item) => kitaYearForDate(item.from) === kitaYear);
  }, [billPeriods, kitaYear]);

  // Fetch comparison data for the entire kita year range in one call
  const { data: comparison, isLoading: comparisonLoading } = useQuery({
    queryKey: queryKeys.governmentFundingBillPeriods.compareRange(orgId, kitaYearFrom, kitaYearTo),
    queryFn: () => apiClient.compareBills(orgId, { from: kitaYearFrom, to: kitaYearTo }),
    enabled: !!orgId && filteredItems.length > 0,
    staleTime: 5 * 60 * 1000,
  });

  // Build a lookup map: bill_id → comparison data
  const comparisonByBillId = useMemo(() => {
    const map = new Map<number, FundingComparisonResponse>();
    if (comparison?.comparisons) {
      for (const comp of comparison.comparisons) {
        map.set(comp.bill_id, comp);
      }
    }
    return map;
  }, [comparison]);

  // Compute summary stats from comparison data
  const summary = useMemo(() => {
    if (!comparison?.comparisons?.length) return null;
    let matchCount = 0;
    let differenceCount = 0;
    let totalDifference = 0;
    for (const comp of comparison.comparisons) {
      if (comp.difference_count === 0 && comp.bill_only_count === 0 && comp.calc_only_count === 0) {
        matchCount++;
      } else {
        differenceCount++;
      }
      totalDifference += comp.difference;
    }
    return { matchCount, differenceCount, totalDifference };
  }, [comparison]);

  const uploadMutation = useMutation({
    mutationFn: async (file: File) => {
      const formData = new FormData();
      formData.append('file', file);
      return apiClient.uploadGovernmentFundingBill(orgId, formData);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.governmentFundingBillPeriods.all(orgId),
      });
      setSelectedFile(null);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
      toast({ title: tCommon('success') });
    },
    onError: (error) => {
      toast({
        title: t('uploadError'),
        description: getErrorMessage(error, t('uploadError')),
        variant: 'destructive',
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => apiClient.deleteGovernmentFundingBillPeriod(orgId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.governmentFundingBillPeriods.all(orgId),
      });
      setDeleteTarget(null);
      toast({ title: t('deleteSuccess') });
    },
    onError: (error) => {
      toast({
        title: tCommon('error'),
        description: getErrorMessage(error, tCommon('error')),
        variant: 'destructive',
      });
    },
  });

  const handleUpload = () => {
    if (selectedFile) {
      uploadMutation.mutate(selectedFile);
    }
  };

  /** Row background class based on comparison status */
  const rowStatusClass = (billId: number): string => {
    const comp = comparisonByBillId.get(billId);
    if (!comp) return '';
    if (comp.difference_count === 0 && comp.bill_only_count === 0 && comp.calc_only_count === 0) {
      return '';
    }
    return 'bg-destructive/5';
  };

  return (
    <TooltipProvider>
      <div className="space-y-6">
        <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
          <div className="min-w-0">
            <h1 className="text-3xl font-bold tracking-tight">{t('title')}</h1>
            <p className="text-muted-foreground mt-1 max-w-3xl text-sm">{t('description')}</p>
          </div>
          <div className="flex flex-wrap items-center gap-4">
            <SearchInput
              id="search-bills"
              value={searchInput}
              onChange={setSearchInput}
              placeholder={t('searchPlaceholder')}
            />
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground text-sm">{tCommon('kitaYearLabel')}</span>
              <KitaYearStepper value={kitaYear} onChange={setKitaYear} />
            </div>
          </div>
        </div>

        {/* Upload Card */}
        <Card>
          <CardHeader>
            <CardTitle>{t('selectFile')}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
              <div className="flex-1">
                <Input
                  ref={fileInputRef}
                  type="file"
                  accept=".xlsx"
                  onChange={(e) => setSelectedFile(e.target.files?.[0] ?? null)}
                />
              </div>
              <Button onClick={handleUpload} disabled={!selectedFile || uploadMutation.isPending}>
                <Upload className="mr-2 h-4 w-4" />
                {uploadMutation.isPending ? t('uploading') : t('upload')}
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Summary Bar */}
        {summary && (
          <div className="flex flex-wrap gap-4">
            <div className="bg-success/10 border-success/30 flex items-center gap-2 rounded-md border px-4 py-2">
              <CheckCircle2 className="text-success h-4 w-4" />
              <span data-visual-mask="stat" className="text-sm font-medium">
                {t('summaryMatch', { count: summary.matchCount })}
              </span>
            </div>
            {summary.differenceCount > 0 && (
              <div className="bg-destructive/10 border-destructive/30 flex items-center gap-2 rounded-md border px-4 py-2">
                <XCircle className="text-destructive h-4 w-4" />
                <span data-visual-mask="stat" className="text-sm font-medium">
                  {t('summaryDifference', { count: summary.differenceCount })}
                </span>
              </div>
            )}
            <div className="flex items-center gap-2 rounded-md border px-4 py-2">
              <span className="text-muted-foreground text-sm">{t('summaryTotal')}:</span>
              <span
                data-visual-mask="currency"
                className={`text-sm font-semibold ${summary.totalDifference >= 0 ? 'text-success' : 'text-destructive'}`}
              >
                {fmt.currency(summary.totalDifference)}
              </span>
            </div>
          </div>
        )}

        {/* Bill Periods List */}
        <Card>
          <CardHeader>
            <CardTitle>
              {t('title')} {/* the Kita year defaults to the current one */}
              <span data-visual-mask="date">
                {tStats('kitaYear', { year: `${kitaYear}/${String(kitaYear + 1).slice(2)}` })}
              </span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <QueryError error={queryError} onRetry={refetch} />
            {isLoading ? (
              <p className="text-muted-foreground py-4 text-center">{tCommon('loading')}</p>
            ) : filteredItems.length === 0 ? (
              <p className="text-muted-foreground py-4 text-center">{t('noBills')}</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('billingMonth')}</TableHead>
                    <TableHead>{t('facilityName')}</TableHead>
                    <TableHead className="hidden md:table-cell">
                      <HeaderWithTooltip
                        label={t('facilityTotal')}
                        tooltip={t('facilityTotalTooltip')}
                      />
                    </TableHead>
                    <TableHead className="hidden md:table-cell">
                      <HeaderWithTooltip
                        label={t('correctionTotal')}
                        tooltip={t('correctionTotalTooltip')}
                      />
                    </TableHead>
                    <TableHead className="hidden md:table-cell">
                      <HeaderWithTooltip
                        label={t('calculatedTotal')}
                        tooltip={t('calculatedTotalTooltip')}
                      />
                    </TableHead>
                    <TableHead className="hidden md:table-cell">
                      <HeaderWithTooltip label={t('difference')} tooltip={t('differenceTooltip')} />
                    </TableHead>
                    <TableHead className="hidden md:table-cell">{t('fileName')}</TableHead>
                    <TableHead className="text-right">{tCommon('actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredItems.map((item) => {
                    const comp = comparisonByBillId.get(item.id);
                    return (
                      <TableRow key={item.id} className={rowStatusClass(item.id)}>
                        {/*
                          Masked: the seeded ISBJ periods are generated relative
                          to seeding time, so every row's month moves with the
                          calendar. Crossing a month boundary rewrote all six
                          rows and expired this page's baseline three days after
                          it was taken.
                        */}
                        <TableCell data-visual-mask="date">
                          {fmt.monthYear(item.from, { month: 'long', year: 'numeric' })}
                        </TableCell>
                        <TableCell>{item.facility_name}</TableCell>
                        <TableCell data-visual-mask="currency" className="hidden md:table-cell">
                          {fmt.currency(item.facility_total)}
                        </TableCell>
                        {comparisonLoading ? (
                          <>
                            <TableCell className="hidden md:table-cell">
                              <Skeleton className="h-4 w-20" />
                            </TableCell>
                            <TableCell className="hidden md:table-cell">
                              <Skeleton className="h-4 w-20" />
                            </TableCell>
                            <TableCell className="hidden md:table-cell">
                              <Skeleton className="h-4 w-20" />
                            </TableCell>
                          </>
                        ) : comp ? (
                          <>
                            <TableCell data-visual-mask="currency" className="hidden md:table-cell">
                              {comp.correction_total ? (
                                <span className="text-info">
                                  {fmt.currency(comp.correction_total)}
                                </span>
                              ) : (
                                '\u2014'
                              )}
                            </TableCell>
                            <TableCell data-visual-mask="currency" className="hidden md:table-cell">
                              {fmt.currency(comp.calculated_total)}
                            </TableCell>
                            <TableCell data-visual-mask="currency" className="hidden md:table-cell">
                              <span
                                className={
                                  comp.difference < 0 ? 'text-destructive' : 'text-success'
                                }
                              >
                                {fmt.currency(comp.difference)}
                              </span>
                            </TableCell>
                          </>
                        ) : (
                          <>
                            <TableCell className="text-muted-foreground hidden md:table-cell">
                              &mdash;
                            </TableCell>
                            <TableCell className="text-muted-foreground hidden md:table-cell">
                              &mdash;
                            </TableCell>
                            <TableCell className="text-muted-foreground hidden md:table-cell">
                              &mdash;
                            </TableCell>
                          </>
                        )}
                        {/* The seeded file names carry the period they cover. */}
                        <TableCell data-visual-mask="date" className="hidden text-sm md:table-cell">
                          {item.file_name}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-1">
                            <Button variant="ghost" size="icon" asChild>
                              {/* An icon with no text needs a name, or a screen
                                  reader announces only "link" and the way into
                                  the per-child breakdown is unreachable. */}
                              <Link
                                href={`/organizations/${orgId}/government-funding-bills/${item.id}`}
                                aria-label={tCommon('viewDetails')}
                              >
                                <Eye className="h-4 w-4" />
                              </Link>
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon"
                              onClick={() => setDeleteTarget(item)}
                            >
                              <Trash2 className="text-destructive h-4 w-4" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>

        {/* Delete Confirmation Dialog */}
        <AlertDialog open={!!deleteTarget} onOpenChange={() => setDeleteTarget(null)}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>{t('deleteBill')}</AlertDialogTitle>
              <AlertDialogDescription>{t('deleteConfirm')}</AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>{tCommon('cancel')}</AlertDialogCancel>
              <AlertDialogAction
                onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
                className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              >
                {tCommon('delete')}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </TooltipProvider>
  );
}
