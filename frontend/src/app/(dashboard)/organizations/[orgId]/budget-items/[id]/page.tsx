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
import {
  formatCurrency,
  formatPeriod,
  formatDateForApi,
  eurosToCents,
  centsToEuros,
} from '@/lib/utils/formatting';
import { budgetItemEntrySchema, type BudgetItemEntryFormData } from '@/lib/schemas';

export default function BudgetItemDetailPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const budgetItemId = Number(params.id);
  const t = useTranslations();

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
    formState: { errors },
  } = useForm<BudgetItemEntryFormData>({
    resolver: zodResolver(budgetItemEntrySchema),
    defaultValues: { from: '', to: '', amount_euros: 0, notes: '' },
  });

  const detailQueryKey = queryKeys.budgetItems.detail(orgId, budgetItemId);
  const financialsKey = queryKeys.statistics.financials(orgId);

  const createEntryMutation = useResourceMutation({
    mutationFn: (data: BudgetItemEntryCreateRequest) =>
      apiClient.createBudgetItemEntry(orgId, budgetItemId, data),
    invalidateQueryKey: [detailQueryKey, financialsKey],
    successMessage: t('budgetItems.entryCreated'),
    errorMessage: t('budgetItems.failedToSaveEntry'),
    onSuccess: () => {
      setIsEntryDialogOpen(false);
      setEditingEntry(null);
      reset();
    },
  });

  const updateEntryMutation = useResourceMutation({
    mutationFn: ({ entryId, data }: { entryId: number; data: BudgetItemEntryUpdateRequest }) =>
      apiClient.updateBudgetItemEntry(orgId, budgetItemId, entryId, data),
    invalidateQueryKey: [detailQueryKey, financialsKey],
    successMessage: t('budgetItems.entryUpdated'),
    errorMessage: t('budgetItems.failedToSaveEntry'),
    onSuccess: () => {
      setIsEntryDialogOpen(false);
      setEditingEntry(null);
      reset();
    },
  });

  const deleteEntryMutation = useResourceMutation({
    mutationFn: (entryId: number) => apiClient.deleteBudgetItemEntry(orgId, budgetItemId, entryId),
    invalidateQueryKey: [detailQueryKey, financialsKey],
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
      to: formatDateForApi(data.to) || null,
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
          <Button size="sm" onClick={handleAddEntry}>
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
                    <TableCell>
                      {formatPeriod(entry.from, entry.to, 'en', t('common.ongoing'))}
                    </TableCell>
                    <TableCell>{formatCurrency(entry.amount_cents)}</TableCell>
                    <TableCell>{entry.notes || '-'}</TableCell>
                    <TableCell className="text-right">
                      <Button size="icon" variant="ghost" onClick={() => handleEditEntry(entry)}>
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        size="icon"
                        variant="ghost"
                        onClick={() => {
                          setDeletingEntry(entry);
                          setIsDeleteEntryDialogOpen(true);
                        }}
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
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="from">{t('budgetItems.fromDate')}</Label>
                <Input id="from" type="date" {...register('from')} />
                {errors.from && (
                  <p className="text-destructive text-sm">{t('validation.fromDateRequired')}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="to">{t('budgetItems.toDateOptional')}</Label>
                <Input id="to" type="date" {...register('to')} />
                {errors.to && (
                  <p className="text-destructive text-sm">
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
                {...register('amount_euros', { valueAsNumber: true })}
              />
              {errors.amount_euros && (
                <p className="text-destructive text-sm">{t('validation.required')}</p>
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
