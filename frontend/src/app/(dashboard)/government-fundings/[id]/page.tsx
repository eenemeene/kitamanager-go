'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
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
import { useToast } from '@/lib/hooks/use-toast';
import { apiClient, getErrorMessage } from '@/lib/api/client';
import type {
  GovernmentFundingPeriod,
  GovernmentFundingProperty,
  GovernmentFundingPeriodCreateRequest,
  GovernmentFundingPropertyCreateRequest,
} from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  formatDate,
  formatCurrency,
  formatPeriod,
  formatAgeRange,
  formatFte,
} from '@/lib/utils/formatting';

const periodSchema = z.object({
  from: z.string().min(1),
  to: z.string().optional(),
  comment: z.string().optional(),
});

const propertySchema = z.object({
  key: z.string().min(1),
  value: z.string().min(1),
  payment: z.number().min(0),
  requirement: z.number().min(0),
  min_age: z.number().optional().nullable(),
  max_age: z.number().optional().nullable(),
  comment: z.string().optional(),
});

type PeriodFormData = z.infer<typeof periodSchema>;
type PropertyFormData = z.infer<typeof propertySchema>;

export default function GovernmentFundingDetailPage() {
  const params = useParams();
  const router = useRouter();
  const fundingId = Number(params.id);
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();

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
    queryKey: ['government-funding', fundingId],
    queryFn: () => apiClient.getGovernmentFunding(fundingId, 100),
    enabled: !!fundingId,
  });

  // Period mutations
  const createPeriodMutation = useMutation({
    mutationFn: (data: GovernmentFundingPeriodCreateRequest) =>
      apiClient.createGovernmentFundingPeriod(fundingId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['government-funding', fundingId] });
      toast({ title: t('governmentFundings.periodCreated') });
      setIsPeriodDialogOpen(false);
      resetPeriod();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('governmentFundings.failedToSavePeriod')),
        variant: 'destructive',
      });
    },
  });

  const deletePeriodMutation = useMutation({
    mutationFn: (periodId: number) => apiClient.deleteGovernmentFundingPeriod(fundingId, periodId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['government-funding', fundingId] });
      toast({ title: t('governmentFundings.periodDeleted') });
      setIsDeletePeriodDialogOpen(false);
      setDeletingPeriod(null);
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('governmentFundings.failedToDeletePeriod')),
        variant: 'destructive',
      });
    },
  });

  // Property mutations
  const createPropertyMutation = useMutation({
    mutationFn: ({
      periodId,
      data,
    }: {
      periodId: number;
      data: GovernmentFundingPropertyCreateRequest;
    }) => apiClient.createGovernmentFundingProperty(fundingId, periodId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['government-funding', fundingId] });
      toast({ title: t('governmentFundings.propertyCreated') });
      setIsPropertyDialogOpen(false);
      resetProperty();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('governmentFundings.failedToSaveProperty')),
        variant: 'destructive',
      });
    },
  });

  const deletePropertyMutation = useMutation({
    mutationFn: ({ periodId, propertyId }: { periodId: number; propertyId: number }) =>
      apiClient.deleteGovernmentFundingProperty(fundingId, periodId, propertyId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['government-funding', fundingId] });
      toast({ title: t('governmentFundings.propertyDeleted') });
      setIsDeletePropertyDialogOpen(false);
      setDeletingProperty(null);
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('governmentFundings.failedToDeleteProperty')),
        variant: 'destructive',
      });
    },
  });

  const {
    register: registerPeriod,
    handleSubmit: handleSubmitPeriod,
    reset: resetPeriod,
    formState: { errors: errorsPeriod },
  } = useForm<PeriodFormData>({
    resolver: zodResolver(periodSchema),
    defaultValues: { from: '', to: '', comment: '' },
  });

  const {
    register: registerProperty,
    handleSubmit: handleSubmitProperty,
    reset: resetProperty,
    formState: { errors: errorsProperty },
  } = useForm<PropertyFormData>({
    resolver: zodResolver(propertySchema),
    defaultValues: {
      key: '',
      value: '',
      payment: 0,
      requirement: 0,
      min_age: null,
      max_age: null,
      comment: '',
    },
  });

  const handleAddPeriod = () => {
    resetPeriod({ from: '', to: '', comment: '' });
    setIsPeriodDialogOpen(true);
  };

  const handleAddProperty = (period: GovernmentFundingPeriod) => {
    setSelectedPeriod(period);
    resetProperty({
      key: '',
      value: '',
      payment: 0,
      requirement: 0,
      min_age: null,
      max_age: null,
      comment: '',
    });
    setIsPropertyDialogOpen(true);
  };

  const onSubmitPeriod = (data: PeriodFormData) => {
    createPeriodMutation.mutate({
      ...data,
      to: data.to || null,
    });
  };

  const onSubmitProperty = (data: PropertyFormData) => {
    if (selectedPeriod) {
      createPropertyMutation.mutate({
        periodId: selectedPeriod.id,
        data: {
          ...data,
          min_age: data.min_age ?? null,
          max_age: data.max_age ?? null,
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
                        {formatPeriod(period.from, period.to, 'en', t('common.ongoing'))}
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
                    {period.properties?.length === 0 ? (
                      <p className="py-4 text-center text-muted-foreground">
                        {t('governmentFundings.noPropertiesDefined')}
                      </p>
                    ) : (
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>{t('governmentFundings.key')}</TableHead>
                            <TableHead>{t('governmentFundings.value')}</TableHead>
                            <TableHead>{t('governmentFundings.ageRange')}</TableHead>
                            <TableHead>{t('governmentFundings.payment')}</TableHead>
                            <TableHead>{t('governmentFundings.requirementFte')}</TableHead>
                            <TableHead className="text-right">{t('common.actions')}</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {period.properties?.map((property) => (
                            <TableRow key={property.id}>
                              <TableCell className="font-medium">{property.key}</TableCell>
                              <TableCell>{property.value}</TableCell>
                              <TableCell>
                                {formatAgeRange(property.min_age, property.max_age)}
                              </TableCell>
                              <TableCell>{formatCurrency(property.payment)}</TableCell>
                              <TableCell>{formatFte(property.requirement)}</TableCell>
                              <TableCell className="text-right">
                                <Button
                                  size="icon"
                                  variant="ghost"
                                  onClick={() => {
                                    setDeletingProperty({ period, property });
                                    setIsDeletePropertyDialogOpen(true);
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
                <Label htmlFor="payment">{t('governmentFundings.paymentInCents')}</Label>
                <Input
                  id="payment"
                  type="number"
                  min={0}
                  {...registerProperty('payment', { valueAsNumber: true })}
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
