'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Plus, Pencil, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
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
import { useToast } from '@/lib/hooks/use-toast';
import { apiClient, getErrorMessage } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import type { CostEntry, CostEntryCreateRequest, CostEntryUpdateRequest } from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  formatCurrency,
  formatPeriod,
  formatDateForApi,
  eurosToCents,
  centsToEuros,
} from '@/lib/utils/formatting';
import { costEntrySchema, type CostEntryFormData } from '@/lib/schemas';

export default function CostDetailPage() {
  const params = useParams();
  const router = useRouter();
  const orgId = Number(params.orgId);
  const costId = Number(params.id);
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const [isEntryDialogOpen, setIsEntryDialogOpen] = useState(false);
  const [editingEntry, setEditingEntry] = useState<CostEntry | null>(null);
  const [isDeleteEntryDialogOpen, setIsDeleteEntryDialogOpen] = useState(false);
  const [deletingEntry, setDeletingEntry] = useState<CostEntry | null>(null);

  const { data: cost, isLoading } = useQuery({
    queryKey: queryKeys.costs.detail(orgId, costId),
    queryFn: () => apiClient.getCost(orgId, costId),
    enabled: !!orgId && !!costId,
  });

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<CostEntryFormData>({
    resolver: zodResolver(costEntrySchema),
    defaultValues: { from: '', to: '', amount_euros: 0, notes: '' },
  });

  const createEntryMutation = useMutation({
    mutationFn: (data: CostEntryCreateRequest) => apiClient.createCostEntry(orgId, costId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.costs.detail(orgId, costId) });
      toast({ title: t('costs.entryCreated') });
      setIsEntryDialogOpen(false);
      setEditingEntry(null);
      reset();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('costs.failedToSaveEntry')),
        variant: 'destructive',
      });
    },
  });

  const updateEntryMutation = useMutation({
    mutationFn: ({ entryId, data }: { entryId: number; data: CostEntryUpdateRequest }) =>
      apiClient.updateCostEntry(orgId, costId, entryId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.costs.detail(orgId, costId) });
      toast({ title: t('costs.entryUpdated') });
      setIsEntryDialogOpen(false);
      setEditingEntry(null);
      reset();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('costs.failedToSaveEntry')),
        variant: 'destructive',
      });
    },
  });

  const deleteEntryMutation = useMutation({
    mutationFn: (entryId: number) => apiClient.deleteCostEntry(orgId, costId, entryId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.costs.detail(orgId, costId) });
      toast({ title: t('costs.entryDeleted') });
      setIsDeleteEntryDialogOpen(false);
      setDeletingEntry(null);
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('costs.failedToDeleteEntry')),
        variant: 'destructive',
      });
    },
  });

  const handleAddEntry = () => {
    setEditingEntry(null);
    reset({ from: '', to: '', amount_euros: 0, notes: '' });
    setIsEntryDialogOpen(true);
  };

  const handleEditEntry = (entry: CostEntry) => {
    setEditingEntry(entry);
    reset({
      from: entry.from?.slice(0, 10) || '',
      to: entry.to?.slice(0, 10) || '',
      amount_euros: centsToEuros(entry.amount_cents),
      notes: entry.notes || '',
    });
    setIsEntryDialogOpen(true);
  };

  const onSubmitEntry = (data: CostEntryFormData) => {
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

  if (!cost) {
    return <div className="text-center text-muted-foreground">{t('costs.failedToLoadCost')}</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => router.back()}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="text-3xl font-bold tracking-tight">{cost.name}</h1>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>{t('payPlans.entry')}</CardTitle>
          <Button size="sm" onClick={handleAddEntry}>
            <Plus className="mr-2 h-4 w-4" />
            {t('costs.addEntry')}
          </Button>
        </CardHeader>
        <CardContent>
          {!cost.entries?.length ? (
            <p className="py-8 text-center text-muted-foreground">{t('costs.noEntriesDefined')}</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('governmentFundings.period')}</TableHead>
                  <TableHead>{t('governmentFundings.amount')}</TableHead>
                  <TableHead>{t('costs.notes')}</TableHead>
                  <TableHead className="text-right">{t('common.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {cost.entries.map((entry) => (
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
            <DialogTitle>{editingEntry ? t('costs.editEntry') : t('costs.addEntry')}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit(onSubmitEntry)} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="from">{t('costs.fromDate')}</Label>
                <Input id="from" type="date" {...register('from')} />
                {errors.from && (
                  <p className="text-sm text-destructive">{t('validation.fromDateRequired')}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="to">{t('costs.toDateOptional')}</Label>
                <Input id="to" type="date" {...register('to')} />
                {errors.to && (
                  <p className="text-sm text-destructive">
                    {t('validation.toDateMustBeAfterFromDate')}
                  </p>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="amount_euros">{t('costs.amountInEuros')}</Label>
              <Input
                id="amount_euros"
                type="number"
                min={0}
                step={0.01}
                {...register('amount_euros', { valueAsNumber: true })}
              />
              {errors.amount_euros && (
                <p className="text-sm text-destructive">{t('validation.required')}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="notes">{t('costs.notes')}</Label>
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
            <AlertDialogDescription>{t('costs.deleteEntryConfirm')}</AlertDialogDescription>
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
