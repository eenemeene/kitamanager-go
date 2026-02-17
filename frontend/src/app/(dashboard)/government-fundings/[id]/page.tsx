'use client';

import { useMemo, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, Plus, Trash2 } from 'lucide-react';
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
import { useResourceMutation } from '@/lib/hooks/use-resource-mutation';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import type {
  GovernmentFundingPeriod,
  GovernmentFundingProperty,
  GovernmentFundingPeriodCreateRequest,
  GovernmentFundingPropertyCreateRequest,
} from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  formatDate,
  formatDateForApi,
  formatCurrency,
  formatPeriod,
  formatAgeRange,
  formatFte,
  eurosToCents,
} from '@/lib/utils/formatting';
import {
  governmentFundingPeriodSchema,
  governmentFundingPropertySchema,
  type GovernmentFundingPeriodFormData,
  type GovernmentFundingPropertyFormData,
} from '@/lib/schemas';

function PropertiesGroupedByKey({
  period,
  onDeleteProperty,
  t,
}: {
  period: GovernmentFundingPeriod;
  onDeleteProperty: (property: GovernmentFundingProperty) => void;
  t: (key: string) => string;
}) {
  // Group by key, then by value within each key
  const groups = useMemo(() => {
    const keyMap = new Map<string, Map<string, GovernmentFundingProperty[]>>();
    for (const prop of period.properties ?? []) {
      let valueMap = keyMap.get(prop.key);
      if (!valueMap) {
        valueMap = new Map<string, GovernmentFundingProperty[]>();
        keyMap.set(prop.key, valueMap);
      }
      const list = valueMap.get(prop.value);
      if (list) {
        list.push(prop);
      } else {
        valueMap.set(prop.value, [prop]);
      }
    }
    return Array.from(keyMap.entries()).map(
      ([key, valueMap]) =>
        [key, Array.from(valueMap.entries())] as [string, [string, GovernmentFundingProperty[]][]]
    );
  }, [period.properties]);

  if (!period.properties?.length) {
    return (
      <p className="py-4 text-center text-muted-foreground">
        {t('governmentFundings.noPropertiesDefined')}
      </p>
    );
  }

  return (
    <div className="space-y-6">
      {groups.map(([key, valueGroups]) => (
        <div key={key}>
          <h4 className="text-sm font-semibold">{key}</h4>
          <div className="mt-2 space-y-3 pl-4">
            {valueGroups.map(([value, properties]) => (
              <div key={value}>
                <p className="mb-1 text-sm text-muted-foreground">
                  {properties[0]?.label || value}
                </p>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('governmentFundings.ageRange')}</TableHead>
                      <TableHead>{t('governmentFundings.payment')}</TableHead>
                      <TableHead>{t('governmentFundings.requirementFte')}</TableHead>
                      <TableHead className="text-right">{t('common.actions')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {properties.map((property) => (
                      <TableRow key={property.id}>
                        <TableCell>
                          {formatAgeRange(property.min_age, property.max_age, t('common.years'))}
                        </TableCell>
                        <TableCell>{formatCurrency(property.payment)}</TableCell>
                        <TableCell>{formatFte(property.requirement)}</TableCell>
                        <TableCell className="text-right">
                          <Button
                            size="icon"
                            variant="ghost"
                            onClick={() => onDeleteProperty(property)}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

export default function GovernmentFundingDetailPage() {
  const params = useParams();
  const router = useRouter();
  const fundingId = Number(params.id);
  const t = useTranslations();

  const [isPeriodDialogOpen, setIsPeriodDialogOpen] = useState(false);
  const [isPropertyDialogOpen, setIsPropertyDialogOpen] = useState(false);
  const [isDeletePeriodDialogOpen, setIsDeletePeriodDialogOpen] = useState(false);
  const [isDeletePropertyDialogOpen, setIsDeletePropertyDialogOpen] = useState(false);
  const [selectedPeriod, setSelectedPeriod] = useState<GovernmentFundingPeriod | null>(null);
  const [deletingPeriod, setDeletingPeriod] = useState<GovernmentFundingPeriod | null>(null);
  const [deletingProperty, setDeletingProperty] = useState<{
    period: GovernmentFundingPeriod;
    property: GovernmentFundingProperty;
  } | null>(null);

  const { data: funding, isLoading } = useQuery({
    queryKey: queryKeys.governmentFundings.detail(fundingId),
    queryFn: () => apiClient.getGovernmentFunding(fundingId, 100),
    enabled: !!fundingId,
  });

  const detailQueryKey = queryKeys.governmentFundings.detail(fundingId);

  // Period mutations
  const createPeriodMutation = useResourceMutation({
    mutationFn: (data: GovernmentFundingPeriodCreateRequest) =>
      apiClient.createGovernmentFundingPeriod(fundingId, data),
    invalidateQueryKey: detailQueryKey,
    successMessage: t('governmentFundings.periodCreated'),
    errorMessage: t('governmentFundings.failedToSavePeriod'),
    onSuccess: () => {
      setIsPeriodDialogOpen(false);
      resetPeriod();
    },
  });

  const deletePeriodMutation = useResourceMutation({
    mutationFn: (periodId: number) => apiClient.deleteGovernmentFundingPeriod(fundingId, periodId),
    invalidateQueryKey: detailQueryKey,
    successMessage: t('governmentFundings.periodDeleted'),
    errorMessage: t('governmentFundings.failedToDeletePeriod'),
    onSuccess: () => {
      setIsDeletePeriodDialogOpen(false);
      setDeletingPeriod(null);
    },
  });

  // Property mutations
  const createPropertyMutation = useResourceMutation({
    mutationFn: ({
      periodId,
      data,
    }: {
      periodId: number;
      data: GovernmentFundingPropertyCreateRequest;
    }) => apiClient.createGovernmentFundingProperty(fundingId, periodId, data),
    invalidateQueryKey: detailQueryKey,
    successMessage: t('governmentFundings.propertyCreated'),
    errorMessage: t('governmentFundings.failedToSaveProperty'),
    onSuccess: () => {
      setIsPropertyDialogOpen(false);
      resetProperty();
    },
  });

  const deletePropertyMutation = useResourceMutation({
    mutationFn: ({ periodId, propertyId }: { periodId: number; propertyId: number }) =>
      apiClient.deleteGovernmentFundingProperty(fundingId, periodId, propertyId),
    invalidateQueryKey: detailQueryKey,
    successMessage: t('governmentFundings.propertyDeleted'),
    errorMessage: t('governmentFundings.failedToDeleteProperty'),
    onSuccess: () => {
      setIsDeletePropertyDialogOpen(false);
      setDeletingProperty(null);
    },
  });

  const {
    register: registerPeriod,
    handleSubmit: handleSubmitPeriod,
    reset: resetPeriod,
    formState: { errors: errorsPeriod },
  } = useForm<GovernmentFundingPeriodFormData>({
    resolver: zodResolver(governmentFundingPeriodSchema),
    defaultValues: { from: '', to: '', full_time_weekly_hours: 39, comment: '' },
  });

  const {
    register: registerProperty,
    handleSubmit: handleSubmitProperty,
    reset: resetProperty,
    formState: { errors: errorsProperty },
  } = useForm<GovernmentFundingPropertyFormData>({
    resolver: zodResolver(governmentFundingPropertySchema),
    defaultValues: {
      key: '',
      value: '',
      label: '',
      payment_euros: 0,
      requirement: 0,
      min_age: null,
      max_age: null,
      comment: '',
    },
  });

  const handleAddPeriod = () => {
    resetPeriod({ from: '', to: '', full_time_weekly_hours: 39, comment: '' });
    setIsPeriodDialogOpen(true);
  };

  const handleAddProperty = (period: GovernmentFundingPeriod) => {
    setSelectedPeriod(period);
    resetProperty({
      key: '',
      value: '',
      label: '',
      payment_euros: 0,
      requirement: 0,
      min_age: null,
      max_age: null,
      comment: '',
    });
    setIsPropertyDialogOpen(true);
  };

  const onSubmitPeriod = (data: GovernmentFundingPeriodFormData) => {
    createPeriodMutation.mutate({
      ...data,
      from: formatDateForApi(data.from)!,
      to: formatDateForApi(data.to) || null,
    });
  };

  const onSubmitProperty = (data: GovernmentFundingPropertyFormData) => {
    if (selectedPeriod) {
      createPropertyMutation.mutate({
        periodId: selectedPeriod.id,
        data: {
          key: data.key,
          value: data.value,
          label: data.label,
          payment: eurosToCents(data.payment_euros),
          requirement: data.requirement,
          min_age: data.min_age ?? null,
          max_age: data.max_age ?? null,
          comment: data.comment,
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

  if (!funding) {
    return (
      <div className="text-center text-muted-foreground">
        {t('governmentFundings.failedToLoadFunding')}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => router.back()}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{funding.name}</h1>
          <p className="text-muted-foreground">{t(`states.${funding.state}`)}</p>
        </div>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>{t('governmentFundings.periods')}</CardTitle>
          <Button size="sm" onClick={handleAddPeriod}>
            <Plus className="mr-2 h-4 w-4" />
            {t('governmentFundings.addPeriod')}
          </Button>
        </CardHeader>
        <CardContent>
          {funding.periods?.length === 0 ? (
            <p className="py-8 text-center text-muted-foreground">
              {t('governmentFundings.noPeriodsDefined')}
            </p>
          ) : (
            <div className="space-y-6">
              {funding.periods?.map((period) => (
                <Card key={period.id}>
                  <CardHeader className="flex flex-row items-center justify-between py-3">
                    <div>
                      <CardTitle className="text-base">
                        {formatPeriod(period.from, period.to, 'en', t('common.ongoing'))}{' '}
                        <span className="text-sm font-normal text-muted-foreground">
                          ({period.full_time_weekly_hours}h/FTE)
                        </span>
                      </CardTitle>
                      {period.comment && (
                        <p className="text-sm text-muted-foreground">{period.comment}</p>
                      )}
                    </div>
                    <div className="flex gap-2">
                      <Button size="sm" variant="outline" onClick={() => handleAddProperty(period)}>
                        <Plus className="mr-2 h-4 w-4" />
                        {t('governmentFundings.addProperty')}
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
                    <PropertiesGroupedByKey
                      period={period}
                      onDeleteProperty={(property) => {
                        setDeletingProperty({ period, property });
                        setIsDeletePropertyDialogOpen(true);
                      }}
                      t={t}
                    />
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
            <DialogTitle>{t('governmentFundings.addPeriod')}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmitPeriod(onSubmitPeriod)} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="from">{t('governmentFundings.fromDate')}</Label>
                <Input id="from" type="date" {...registerPeriod('from')} />
                {errorsPeriod.from && (
                  <p className="text-sm text-destructive">{t('validation.fromDateRequired')}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="to">{t('governmentFundings.toDateOptional')}</Label>
                <Input id="to" type="date" {...registerPeriod('to')} />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="full_time_weekly_hours">
                {t('governmentFundings.fullTimeWeeklyHours')}
              </Label>
              <Input
                id="full_time_weekly_hours"
                type="number"
                min={0.1}
                max={80}
                step={0.5}
                {...registerPeriod('full_time_weekly_hours', { valueAsNumber: true })}
              />
              {errorsPeriod.full_time_weekly_hours && (
                <p className="text-sm text-destructive">
                  {t('validation.fullTimeWeeklyHoursRequired')}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="comment">{t('common.comment')}</Label>
              <Input id="comment" {...registerPeriod('comment')} />
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setIsPeriodDialogOpen(false)}>
                {t('common.cancel')}
              </Button>
              <Button type="submit" disabled={createPeriodMutation.isPending}>
                {t('common.save')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Property Dialog */}
      <Dialog open={isPropertyDialogOpen} onOpenChange={setIsPropertyDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('governmentFundings.addProperty')}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmitProperty(onSubmitProperty)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="label">{t('governmentFundings.label')}</Label>
              <Input id="label" placeholder="Ganztag" {...registerProperty('label')} />
              {errorsProperty.label && (
                <p className="text-sm text-destructive">{t('validation.labelRequired')}</p>
              )}
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="key">{t('governmentFundings.key')}</Label>
                <Input id="key" placeholder="care_type" {...registerProperty('key')} />
                {errorsProperty.key && (
                  <p className="text-sm text-destructive">{t('validation.keyRequired')}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="value">{t('governmentFundings.value')}</Label>
                <Input id="value" placeholder="ganztag" {...registerProperty('value')} />
                {errorsProperty.value && (
                  <p className="text-sm text-destructive">{t('validation.valueRequired')}</p>
                )}
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="payment_euros">{t('governmentFundings.paymentInEuros')}</Label>
                <Input
                  id="payment_euros"
                  type="number"
                  min={0}
                  step={0.01}
                  {...registerProperty('payment_euros', { valueAsNumber: true })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="requirement">{t('governmentFundings.requirement')}</Label>
                <Input
                  id="requirement"
                  type="number"
                  min={0}
                  step={0.01}
                  {...registerProperty('requirement', { valueAsNumber: true })}
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="min_age">{t('governmentFundings.minAge')}</Label>
                <Input
                  id="min_age"
                  type="number"
                  min={0}
                  {...registerProperty('min_age', { valueAsNumber: true })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="max_age">{t('governmentFundings.maxAge')}</Label>
                <Input
                  id="max_age"
                  type="number"
                  min={0}
                  {...registerProperty('max_age', { valueAsNumber: true })}
                />
              </div>
            </div>
            <p className="text-xs text-muted-foreground">{t('governmentFundings.ageRangeHelp')}</p>

            <div className="space-y-2">
              <Label htmlFor="comment">{t('common.comment')}</Label>
              <Input id="comment" {...registerProperty('comment')} />
            </div>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setIsPropertyDialogOpen(false)}
              >
                {t('common.cancel')}
              </Button>
              <Button type="submit" disabled={createPropertyMutation.isPending}>
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
            <AlertDialogDescription>
              {t('governmentFundings.deletePeriodConfirm')}
            </AlertDialogDescription>
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

      {/* Delete Property Confirmation */}
      <AlertDialog open={isDeletePropertyDialogOpen} onOpenChange={setIsDeletePropertyDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('common.confirmDelete')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('governmentFundings.deletePropertyConfirm')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() =>
                deletingProperty &&
                deletePropertyMutation.mutate({
                  periodId: deletingProperty.period.id,
                  propertyId: deletingProperty.property.id,
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
