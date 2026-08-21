'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Plus, Pencil, Trash2, Check } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Breadcrumb } from '@/components/ui/breadcrumb';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
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
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useResourceMutation } from '@/lib/hooks/use-resource-mutation';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import type {
  BudgetItemEntry,
  BudgetItemEntryCreateRequest,
  BudgetItemEntryUpdateRequest,
} from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { formatDateForApi, eurosToCents, centsToEuros } from '@/lib/utils/formatting';
import { budgetItemEntrySchema, type BudgetItemEntryFormData } from '@/lib/schemas';
import { validationTiming } from '@/lib/forms/validation-timing';
import { useProblemFormErrors, suppressesToast } from '@/lib/forms/use-problem-form-errors';
import { useResetOnReopen } from '@/lib/forms/use-reset-on-reopen';
import { FormErrorSummary } from '@/components/forms/form-error-summary';
import { useFormatters } from '@/hooks/use-formatters';

export default function BudgetItemDetailPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const budgetItemId = Number(params.id);
  const t = useTranslations();

  const fmt = useFormatters();
  const [isEntryDialogOpen, setIsEntryDialogOpen] = useState(false);
  const [editingEntry, setEditingEntry] = useState<BudgetItemEntry | null>(null);
  const [isDeleteEntryDialogOpen, setIsDeleteEntryDialogOpen] = useState(false);
  const [deletingEntry, setDeletingEntry] = useState<BudgetItemEntry | null>(null);

  const { data: budgetItem, isLoading } = useQuery({
    queryKey: queryKeys.budgetItems.detail(orgId, budgetItemId),
    queryFn: () => apiClient.getBudgetItem(orgId, budgetItemId),
    enabled: !!orgId && !!budgetItemId,
  });

  const {
    register,
    handleSubmit,
    reset,
    setError,
    clearErrors,
    getValues,
    formState: { errors },
  } = useForm<BudgetItemEntryFormData>({
    ...validationTiming,
    resolver: zodResolver(budgetItemEntrySchema),
    defaultValues: { from: '', to: '', amount_euros: 0, notes: '' },
  });

  const detailQueryKey = queryKeys.budgetItems.detail(orgId, budgetItemId);
  // Use the statistics-namespace prefix so every concrete financials
  // query (parameterised by from/to) is invalidated after an entry
  // mutation. The previous `queryKeys.statistics.financials(orgId)`
  // produced `['statistics', orgId, 'financials', undefined, undefined]`,
  // which TanStack Query matches LITERALLY against `undefined` slots
  // and therefore missed every parameterised financials query.
  const statisticsKey = queryKeys.statistics.all(orgId);

  const createEntryMutation = useResourceMutation({
    onMutationError: suppressesToast,
    mutationFn: (data: BudgetItemEntryCreateRequest) =>
      apiClient.createBudgetItemEntry(orgId, budgetItemId, data),
    invalidateQueryKey: [detailQueryKey, statisticsKey],
    successMessage: t('budgetItems.entryCreated'),
    errorMessage: t('budgetItems.failedToSaveEntry'),
    onSuccess: () => {
      setIsEntryDialogOpen(false);
      setEditingEntry(null);
      reset();
    },
  });

  const updateEntryMutation = useResourceMutation({
    onMutationError: suppressesToast,
    mutationFn: ({ entryId, data }: { entryId: number; data: BudgetItemEntryUpdateRequest }) =>
      apiClient.updateBudgetItemEntry(orgId, budgetItemId, entryId, data),
    invalidateQueryKey: [detailQueryKey, statisticsKey],
    successMessage: t('budgetItems.entryUpdated'),
    errorMessage: t('budgetItems.failedToSaveEntry'),
    onSuccess: () => {
      setIsEntryDialogOpen(false);
      setEditingEntry(null);
      reset();
    },
  });

  // Both mutations feed the one entry form -- it is the same dialog for adding
  // and editing an entry.
  // A dialog that just opened has nothing pending -- react-query keeps a
  // mutation's rejection until the next attempt, so without this the form
  // reopened showing the previous submit's summary.
  useResetOnReopen(isEntryDialogOpen, createEntryMutation, updateEntryMutation);

  const unmappedViolations = useProblemFormErrors(
    [createEntryMutation.error, updateEntryMutation.error],
    { setError, clearErrors, getValues }
  );

  const deleteEntryMutation = useResourceMutation({
    mutationFn: (entryId: number) => apiClient.deleteBudgetItemEntry(orgId, budgetItemId, entryId),
    invalidateQueryKey: [detailQueryKey, statisticsKey],
    successMessage: t('budgetItems.entryDeleted'),
    errorMessage: t('budgetItems.failedToDeleteEntry'),
    onSuccess: () => {
      setIsDeleteEntryDialogOpen(false);
      setDeletingEntry(null);
    },
  });

  const handleAddEntry = () => {
    setEditingEntry(null);
    reset({ from: '', to: '', amount_euros: 0, notes: '' });
    setIsEntryDialogOpen(true);
  };

  const handleEditEntry = (entry: BudgetItemEntry) => {
    setEditingEntry(entry);
    reset({
      from: entry.from?.slice(0, 10) || '',
      to: entry.to?.slice(0, 10) || '',
      amount_euros: centsToEuros(entry.amount_cents),
      notes: entry.notes || '',
    });
    setIsEntryDialogOpen(true);
  };

  const onSubmitEntry = (data: BudgetItemEntryFormData) => {
    const payload = {
      from: formatDateForApi(data.from) || data.from,
      to: formatDateForApi(data.to) || undefined,
      amount_cents: eurosToCents(data.amount_euros),
      notes: data.notes || '',
    };

    if (editingEntry) {
      updateEntryMutation.mutate({ entryId: editingEntry.id, data: payload });
    } else {
      createEntryMutation.mutate(payload);
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-[400px] w-full" />
      </div>
    );
  }

  if (!budgetItem) {
    return (
      <div className="text-muted-foreground text-center">
        {t('budgetItems.failedToLoadBudgetItem')}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="space-y-1">
        <Breadcrumb
          items={[
            { label: t('nav.budgetItems'), href: `/organizations/${orgId}/budget-items` },
            { label: budgetItem.name },
          ]}
        />
        <div className="flex flex-wrap items-center gap-4">
          <h1 className="text-3xl font-bold tracking-tight">{budgetItem.name}</h1>
          <Badge variant={budgetItem.category === 'income' ? 'default' : 'secondary'}>
            {t(`budgetItems.category${budgetItem.category === 'income' ? 'Income' : 'Expense'}`)}
          </Badge>
          {budgetItem.per_child && (
            <span className="text-muted-foreground flex items-center gap-1 text-sm">
              <Check className="h-4 w-4" />
              {t('budgetItems.perChild')}
            </span>
          )}
        </div>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>{t('payPlans.entry')}</CardTitle>
          <Button onClick={handleAddEntry}>
            <Plus className="mr-2 h-4 w-4" />
            {t('budgetItems.addEntry')}
          </Button>
        </CardHeader>
        <CardContent>
          {!budgetItem.entries?.length ? (
            <p className="text-muted-foreground py-8 text-center">
              {t('budgetItems.noEntriesDefined')}
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('governmentFundings.period')}</TableHead>
                  <TableHead>{t('governmentFundings.amount')}</TableHead>
                  <TableHead>{t('budgetItems.notes')}</TableHead>
                  <TableHead className="text-right">{t('common.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {budgetItem.entries.map((entry) => (
                  <TableRow key={entry.id}>
                    <TableCell>{fmt.period(entry.from, entry.to, t('common.ongoing'))}</TableCell>
                    <TableCell>{fmt.currency(entry.amount_cents)}</TableCell>
                    <TableCell>{entry.notes || '-'}</TableCell>
                    <TableCell className="text-right">
                      <Button
                        size="icon"
                        variant="ghost"
                        onClick={() => handleEditEntry(entry)}
                        aria-label={t('common.edit')}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        size="icon"
                        variant="ghost"
                        onClick={() => {
                          setDeletingEntry(entry);
                          setIsDeleteEntryDialogOpen(true);
                        }}
                        aria-label={t('common.delete')}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Entry Dialog (Create/Edit) */}
      <Dialog open={isEntryDialogOpen} onOpenChange={setIsEntryDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editingEntry ? t('budgetItems.editEntry') : t('budgetItems.addEntry')}
            </DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit(onSubmitEntry)} className="space-y-4">
            <FormErrorSummary
              errors={errors}
              unmapped={unmappedViolations}
              labels={{
                from: t('budgetItems.fromDate'),
                to: t('budgetItems.toDateOptional'),
                amount_euros: t('budgetItems.amountInEuros'),
                notes: t('budgetItems.notes'),
              }}
            />
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="from">{t('budgetItems.fromDate')}</Label>
                <Input
                  id="from"
                  type="date"
                  aria-invalid={!!errors.from}
                  aria-describedby={errors.from ? 'from-error' : undefined}
                  {...register('from')}
                />
                {errors.from && (
                  <p id="from-error" className="text-destructive text-sm">
                    {t('validation.fromDateRequired')}
                  </p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="to">{t('budgetItems.toDateOptional')}</Label>
                <Input
                  id="to"
                  type="date"
                  aria-invalid={!!errors.to}
                  aria-describedby={errors.to ? 'to-error' : undefined}
                  {...register('to')}
                />
                {errors.to && (
                  <p id="to-error" className="text-destructive text-sm">
                    {t('validation.toDateMustBeAfterFromDate')}
                  </p>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="amount_euros">{t('budgetItems.amountInEuros')}</Label>
              <Input
                id="amount_euros"
                type="number"
                min={0}
                step={0.01}
                aria-invalid={!!errors.amount_euros}
                aria-describedby={errors.amount_euros ? 'amount-error' : undefined}
                {...register('amount_euros', { valueAsNumber: true })}
              />
              {errors.amount_euros && (
                <p id="amount-error" className="text-destructive text-sm">
                  {t('validation.required')}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="notes">{t('budgetItems.notes')}</Label>
              <Input id="notes" {...register('notes')} />
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setIsEntryDialogOpen(false)}>
                {t('common.cancel')}
              </Button>
              <Button
                type="submit"
                disabled={createEntryMutation.isPending || updateEntryMutation.isPending}
              >
                {t('common.save')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Delete Entry Confirmation */}
      <AlertDialog open={isDeleteEntryDialogOpen} onOpenChange={setIsDeleteEntryDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('common.confirmDelete')}</AlertDialogTitle>
            <AlertDialogDescription>{t('budgetItems.deleteEntryConfirm')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deletingEntry && deleteEntryMutation.mutate(deletingEntry.id)}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {t('common.delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
