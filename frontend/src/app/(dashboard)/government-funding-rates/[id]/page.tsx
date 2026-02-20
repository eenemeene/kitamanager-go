'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, Plus, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
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
import { useResourceMutation } from '@/lib/hooks/use-resource-mutation';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import type {
  GovernmentFundingPeriod,
  GovernmentFundingProperty,
  GovernmentFundingPeriodCreateRequest,
  GovernmentFundingPropertyCreateRequest,
} from '@/lib/api/types';
import { formatDateForApi, formatPeriod, eurosToCents } from '@/lib/utils/formatting';
import type {
  GovernmentFundingPeriodFormData,
  GovernmentFundingPropertyFormData,
} from '@/lib/schemas';
import { PropertiesGroupedByKey } from '@/components/government-funding-rates/properties-grouped-by-key';
import { PeriodFormDialog } from '@/components/government-funding-rates/period-form-dialog';
import { PropertyFormDialog } from '@/components/government-funding-rates/property-form-dialog';

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
    onSuccess: () => setIsPeriodDialogOpen(false),
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
    onSuccess: () => setIsPropertyDialogOpen(false),
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

  const handleAddProperty = (period: GovernmentFundingPeriod) => {
    setSelectedPeriod(period);
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
      <div className="text-muted-foreground text-center">
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
          <Button size="sm" onClick={() => setIsPeriodDialogOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t('governmentFundings.addPeriod')}
          </Button>
        </CardHeader>
        <CardContent>
          {funding.periods?.length === 0 ? (
            <p className="text-muted-foreground py-8 text-center">
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
                        <span className="text-muted-foreground text-sm font-normal">
                          ({period.full_time_weekly_hours}h/FTE)
                        </span>
                      </CardTitle>
                      {period.comment && (
                        <p className="text-muted-foreground text-sm">{period.comment}</p>
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

      <PeriodFormDialog
        open={isPeriodDialogOpen}
        onOpenChange={setIsPeriodDialogOpen}
        onSubmit={onSubmitPeriod}
        isSaving={createPeriodMutation.isPending}
      />

      <PropertyFormDialog
        open={isPropertyDialogOpen}
        onOpenChange={setIsPropertyDialogOpen}
        onSubmit={onSubmitProperty}
        isSaving={createPropertyMutation.isPending}
      />

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
