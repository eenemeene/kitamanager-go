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
import type {
  PayPlanPeriod,
  PayPlanEntry,
  PayPlanPeriodCreateRequest,
  PayPlanPeriodUpdateRequest,
  PayPlanEntryCreateRequest,
} from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { formatDate, formatCurrency, formatPeriod } from '@/lib/utils/formatting';
import { PayPlanGrid } from '@/components/payplans/payplan-grid';
import {
  payPlanPeriodSchema,
  payPlanEntrySchema,
  type PayPlanPeriodFormData,
  type PayPlanEntryFormData,
} from '@/lib/schemas';

export default function PayPlanDetailPage() {
  const params = useParams();
  const router = useRouter();
  const orgId = Number(params.orgId);
  const payPlanId = Number(params.id);
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const [view, setView] = useState<'panels' | 'table'>('table');
  const [isPeriodDialogOpen, setIsPeriodDialogOpen] = useState(false);
  const [isEntryDialogOpen, setIsEntryDialogOpen] = useState(false);
  const [isDeletePeriodDialogOpen, setIsDeletePeriodDialogOpen] = useState(false);
  const [isDeleteEntryDialogOpen, setIsDeleteEntryDialogOpen] = useState(false);
  const [selectedPeriod, setSelectedPeriod] = useState<PayPlanPeriod | null>(null);
  const [editingPeriod, setEditingPeriod] = useState<PayPlanPeriod | null>(null);
  const [editingEntry, setEditingEntry] = useState<PayPlanEntry | null>(null);
  const [deletingPeriod, setDeletingPeriod] = useState<PayPlanPeriod | null>(null);
  const [deletingEntry, setDeletingEntry] = useState<{
    period: PayPlanPeriod;
    entry: PayPlanEntry;
  } | null>(null);

  const { data: payPlan, isLoading } = useQuery({
    queryKey: queryKeys.payPlans.detail(orgId, payPlanId),
    queryFn: () => apiClient.getPayPlan(orgId, payPlanId),
    enabled: !!orgId && !!payPlanId,
  });

  // Period mutations
  const createPeriodMutation = useMutation({
    mutationFn: (data: PayPlanPeriodCreateRequest) =>
      apiClient.createPayPlanPeriod(orgId, payPlanId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.payPlans.detail(orgId, payPlanId) });
      toast({ title: t('payPlans.periodCreated') });
      setIsPeriodDialogOpen(false);
      resetPeriod();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('payPlans.failedToSavePeriod')),
        variant: 'destructive',
      });
    },
  });

  const updatePeriodMutation = useMutation({
    mutationFn: ({ periodId, data }: { periodId: number; data: PayPlanPeriodUpdateRequest }) =>
      apiClient.updatePayPlanPeriod(orgId, payPlanId, periodId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.payPlans.detail(orgId, payPlanId) });
      toast({ title: t('payPlans.periodUpdated') });
      setIsPeriodDialogOpen(false);
      setEditingPeriod(null);
      resetPeriod();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('payPlans.failedToSavePeriod')),
        variant: 'destructive',
      });
    },
  });

  const deletePeriodMutation = useMutation({
    mutationFn: (periodId: number) => apiClient.deletePayPlanPeriod(orgId, payPlanId, periodId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.payPlans.detail(orgId, payPlanId) });
      toast({ title: t('payPlans.periodDeleted') });
      setIsDeletePeriodDialogOpen(false);
      setDeletingPeriod(null);
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('payPlans.failedToDeletePeriod')),
        variant: 'destructive',
      });
    },
  });

  // Entry mutations
  const createEntryMutation = useMutation({
    mutationFn: ({ periodId, data }: { periodId: number; data: PayPlanEntryCreateRequest }) =>
      apiClient.createPayPlanEntry(orgId, payPlanId, periodId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.payPlans.detail(orgId, payPlanId) });
      toast({ title: t('payPlans.entryCreated') });
      setIsEntryDialogOpen(false);
      resetEntry();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('payPlans.failedToSaveEntry')),
        variant: 'destructive',
      });
    },
  });

  const deleteEntryMutation = useMutation({
    mutationFn: ({ periodId, entryId }: { periodId: number; entryId: number }) =>
      apiClient.deletePayPlanEntry(orgId, payPlanId, periodId, entryId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.payPlans.detail(orgId, payPlanId) });
      toast({ title: t('payPlans.entryDeleted') });
      setIsDeleteEntryDialogOpen(false);
      setDeletingEntry(null);
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('payPlans.failedToDeleteEntry')),
        variant: 'destructive',
      });
    },
  });

  const {
    register: registerPeriod,
    handleSubmit: handleSubmitPeriod,
    reset: resetPeriod,
    formState: { errors: errorsPeriod },
  } = useForm<PayPlanPeriodFormData>({
    resolver: zodResolver(payPlanPeriodSchema),
    defaultValues: { from: '', to: '', weekly_hours: 39, employer_contribution_rate: 0 },
  });

  const {
    register: registerEntry,
    handleSubmit: handleSubmitEntry,
    reset: resetEntry,
    formState: { errors: errorsEntry },
  } = useForm<PayPlanEntryFormData>({
    resolver: zodResolver(payPlanEntrySchema),
    defaultValues: { grade: '', step: 1, monthly_amount: 0, step_min_years: undefined },
  });

  const handleAddPeriod = () => {
    setEditingPeriod(null);
    resetPeriod({ from: '', to: '', weekly_hours: 39, employer_contribution_rate: 0 });
    setIsPeriodDialogOpen(true);
  };

  const handleEditPeriod = (period: PayPlanPeriod) => {
    setEditingPeriod(period);
    resetPeriod({
      from: period.from.slice(0, 10),
      to: period.to?.slice(0, 10) ?? '',
      weekly_hours: period.weekly_hours,
      employer_contribution_rate: period.employer_contribution_rate / 100,
    });
    setIsPeriodDialogOpen(true);
  };

  const handleAddEntry = (period: PayPlanPeriod) => {
    setSelectedPeriod(period);
    setEditingEntry(null);
    resetEntry({ grade: '', step: 1, monthly_amount: 0, step_min_years: undefined });
    setIsEntryDialogOpen(true);
  };

  const onSubmitPeriod = (data: PayPlanPeriodFormData) => {
    const payload = {
      ...data,
      to: data.to || null,
      employer_contribution_rate: Math.round(data.employer_contribution_rate * 100),
    };
    if (editingPeriod) {
      updatePeriodMutation.mutate({ periodId: editingPeriod.id, data: payload });
    } else {
      createPeriodMutation.mutate(payload);
    }
  };

  const onSubmitEntry = (data: PayPlanEntryFormData) => {
    if (selectedPeriod) {
      createEntryMutation.mutate({
        periodId: selectedPeriod.id,
        data: {
          ...data,
          step_min_years:
            data.step_min_years != null && !isNaN(data.step_min_years)
              ? data.step_min_years
              : undefined,
        },
      });
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

  if (!payPlan) {
    return (
      <div className="text-center text-muted-foreground">{t('payPlans.failedToLoadPayPlan')}</div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => router.back()}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{payPlan.name}</h1>
        </div>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>{t('governmentFundings.periods')}</CardTitle>
          <div className="flex items-center gap-2">
            <div className="flex rounded-md border">
              <Button
                size="sm"
                variant={view === 'panels' ? 'secondary' : 'ghost'}
                className="rounded-r-none"
                onClick={() => setView('panels')}
              >
                {t('payPlans.viewPanels')}
              </Button>
              <Button
                size="sm"
                variant={view === 'table' ? 'secondary' : 'ghost'}
                className="rounded-l-none"
                onClick={() => setView('table')}
              >
                {t('payPlans.viewTable')}
              </Button>
            </div>
            {view === 'panels' && (
              <Button size="sm" onClick={handleAddPeriod}>
                <Plus className="mr-2 h-4 w-4" />
                {t('payPlans.addPeriod')}
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {payPlan.periods?.length === 0 ? (
            <p className="py-8 text-center text-muted-foreground">
              {view === 'panels' ? t('payPlans.noPeriodsDefined') : t('payPlans.noDataDefined')}
            </p>
          ) : view === 'table' ? (
            <div className="space-y-6">
              {payPlan.periods?.map((period) => (
                <div key={period.id}>
                  <h3 className="mb-2 text-sm font-medium">
                    {formatPeriod(period.from, period.to, 'en', t('common.ongoing'))}
                    {' \u2014 '}
                    {period.weekly_hours}h / {t('payPlans.weeklyHours')}
                    {period.employer_contribution_rate > 0 && (
                      <>
                        {' \u2014 '}
                        {t('payPlans.employerContributionRate')}:{' '}
                        {(period.employer_contribution_rate / 100).toFixed(2)}%
                      </>
                    )}
                  </h3>
                  <PayPlanGrid period={period} />
                </div>
              ))}
            </div>
          ) : (
            <div className="space-y-6">
              {payPlan.periods?.map((period) => (
                <Card key={period.id}>
                  <CardHeader className="flex flex-row items-center justify-between py-3">
                    <div>
                      <CardTitle className="text-base">
                        {formatPeriod(period.from, period.to, 'en', t('common.ongoing'))}
                      </CardTitle>
                      <p className="text-sm text-muted-foreground">
                        {period.weekly_hours}h / {t('payPlans.weeklyHours')}
                        {period.employer_contribution_rate > 0 && (
                          <>
                            {' \u2014 '}
                            {t('payPlans.employerContributionRate')}:{' '}
                            {(period.employer_contribution_rate / 100).toFixed(2)}%
                          </>
                        )}
                      </p>
                    </div>
                    <div className="flex gap-2">
                      <Button size="sm" variant="outline" onClick={() => handleAddEntry(period)}>
                        <Plus className="mr-2 h-4 w-4" />
                        {t('payPlans.addEntry')}
                      </Button>
                      <Button size="icon" variant="ghost" onClick={() => handleEditPeriod(period)}>
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        size="icon"
                        variant="ghost"
                        onClick={() => {
                          setDeletingPeriod(period);
                          setIsDeletePeriodDialogOpen(true);
                        }}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </CardHeader>
                  <CardContent>
                    {period.entries?.length === 0 ? (
                      <p className="py-4 text-center text-muted-foreground">
                        {t('payPlans.noEntriesDefined')}
                      </p>
                    ) : (
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>{t('payPlans.grade')}</TableHead>
                            <TableHead>{t('payPlans.step')}</TableHead>
                            <TableHead>{t('payPlans.stepMinYears')}</TableHead>
                            <TableHead>{t('payPlans.monthlyAmount')}</TableHead>
                            <TableHead className="text-right">{t('common.actions')}</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {period.entries?.map((entry) => (
                            <TableRow key={entry.id}>
                              <TableCell className="font-medium">{entry.grade}</TableCell>
                              <TableCell>{entry.step}</TableCell>
                              <TableCell>
                                {entry.step_min_years != null
                                  ? `${entry.step_min_years}y`
                                  : '\u2014'}
                              </TableCell>
                              <TableCell>{formatCurrency(entry.monthly_amount)}</TableCell>
                              <TableCell className="text-right">
                                <Button
                                  size="icon"
                                  variant="ghost"
                                  onClick={() => {
                                    setDeletingEntry({ period, entry });
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
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Period Dialog */}
      <Dialog open={isPeriodDialogOpen} onOpenChange={setIsPeriodDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editingPeriod ? t('payPlans.editPeriod') : t('payPlans.addPeriod')}
            </DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmitPeriod(onSubmitPeriod)} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="from">{t('payPlans.fromDate')}</Label>
                <Input id="from" type="date" {...registerPeriod('from')} />
                {errorsPeriod.from && (
                  <p className="text-sm text-destructive">{t('validation.fromDateRequired')}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="to">{t('payPlans.toDateOptional')}</Label>
                <Input id="to" type="date" {...registerPeriod('to')} />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="weekly_hours">{t('payPlans.weeklyHoursLabel')}</Label>
                <Input
                  id="weekly_hours"
                  type="number"
                  min={0}
                  max={168}
                  step={0.5}
                  {...registerPeriod('weekly_hours', { valueAsNumber: true })}
                />
                {errorsPeriod.weekly_hours && (
                  <p className="text-sm text-destructive">{t('payPlans.weeklyHoursRequired')}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="employer_contribution_rate">
                  {t('payPlans.employerContributionRateLabel')}
                </Label>
                <Input
                  id="employer_contribution_rate"
                  type="number"
                  min={0}
                  max={100}
                  step={0.01}
                  {...registerPeriod('employer_contribution_rate', { valueAsNumber: true })}
                />
                {errorsPeriod.employer_contribution_rate && (
                  <p className="text-sm text-destructive">
                    {t('payPlans.employerContributionRateRequired')}
                  </p>
                )}
              </div>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setIsPeriodDialogOpen(false)}>
                {t('common.cancel')}
              </Button>
              <Button
                type="submit"
                disabled={createPeriodMutation.isPending || updatePeriodMutation.isPending}
              >
                {t('common.save')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Entry Dialog */}
      <Dialog open={isEntryDialogOpen} onOpenChange={setIsEntryDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('payPlans.addEntry')}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmitEntry(onSubmitEntry)} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="grade">{t('payPlans.gradeLabel')}</Label>
                <Input id="grade" {...registerEntry('grade')} placeholder="S8a" />
                {errorsEntry.grade && (
                  <p className="text-sm text-destructive">{t('payPlans.gradeRequired')}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="step">{t('payPlans.stepLabel')}</Label>
                <Input
                  id="step"
                  type="number"
                  min={1}
                  max={6}
                  {...registerEntry('step', { valueAsNumber: true })}
                />
                {errorsEntry.step && (
                  <p className="text-sm text-destructive">{t('payPlans.stepRequired')}</p>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="monthly_amount">{t('payPlans.monthlyAmountInCents')}</Label>
              <Input
                id="monthly_amount"
                type="number"
                min={0}
                {...registerEntry('monthly_amount', { valueAsNumber: true })}
              />
              {errorsEntry.monthly_amount && (
                <p className="text-sm text-destructive">{t('payPlans.monthlyAmountRequired')}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="step_min_years">{t('payPlans.stepMinYearsLabel')}</Label>
              <Input
                id="step_min_years"
                type="number"
                min={0}
                {...registerEntry('step_min_years', { valueAsNumber: true })}
              />
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setIsEntryDialogOpen(false)}>
                {t('common.cancel')}
              </Button>
              <Button type="submit" disabled={createEntryMutation.isPending}>
                {t('common.save')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Delete Period Confirmation */}
      <AlertDialog open={isDeletePeriodDialogOpen} onOpenChange={setIsDeletePeriodDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('common.confirmDelete')}</AlertDialogTitle>
            <AlertDialogDescription>{t('payPlans.deletePeriodConfirm')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deletingPeriod && deletePeriodMutation.mutate(deletingPeriod.id)}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {t('common.delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Delete Entry Confirmation */}
      <AlertDialog open={isDeleteEntryDialogOpen} onOpenChange={setIsDeleteEntryDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('common.confirmDelete')}</AlertDialogTitle>
            <AlertDialogDescription>{t('payPlans.deleteEntryConfirm')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() =>
                deletingEntry &&
                deleteEntryMutation.mutate({
                  periodId: deletingEntry.period.id,
                  entryId: deletingEntry.entry.id,
                })
              }
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
