'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Pencil, Trash2, Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
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
import { PropertyTagInput } from '@/components/ui/tag-input';
import { useToast } from '@/lib/hooks/use-toast';
import { useFundingAttributes } from '@/lib/hooks/use-funding-attributes';
import { apiClient, getErrorMessage } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import type {
  ChildContract,
  ChildContractCreateRequest,
  ChildContractUpdateRequest,
  ContractProperties,
} from '@/lib/api/types';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  formatDate,
  formatDateForInput,
  formatDateForApi,
  propertiesToValues,
} from '@/lib/utils/formatting';
import { calculateContractEndDate } from '@/lib/utils/school-enrollment';
import { getContractStatus, compareDates } from '@/lib/utils/contracts';
import { childContractSchema, type ChildContractFormData } from '@/lib/schemas';
import { useUiStore } from '@/stores/ui-store';

export default function ChildContractsPage() {
  const params = useParams();
  const router = useRouter();
  const orgId = Number(params.orgId);
  const childId = Number(params.childId);
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const [isContractDialogOpen, setIsContractDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [editingContract, setEditingContract] = useState<ChildContract | null>(null);
  const [deletingContract, setDeletingContract] = useState<ChildContract | null>(null);

  // Fetch child data
  const { data: child, isLoading: childLoading } = useQuery({
    queryKey: queryKeys.children.detail(orgId, childId),
    queryFn: () => apiClient.getChild(orgId, childId),
    enabled: !!orgId && !!childId,
  });

  // Fetch contracts
  const { data: contracts, isLoading: contractsLoading } = useQuery({
    queryKey: queryKeys.children.contracts(orgId, childId),
    queryFn: () => apiClient.getChildContracts(orgId, childId),
    enabled: !!orgId && !!childId,
  });

  const createMutation = useMutation({
    mutationFn: (data: ChildContractCreateRequest) =>
      apiClient.createChildContract(orgId, childId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.children.contracts(orgId, childId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.children.detail(orgId, childId) });
      toast({ title: t('contracts.createSuccess') });
      setIsContractDialogOpen(false);
      setEditingContract(null);
      reset();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToCreate', { resource: 'contract' })),
        variant: 'destructive',
      });
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ contractId, data }: { contractId: number; data: ChildContractUpdateRequest }) =>
      apiClient.updateChildContract(orgId, childId, contractId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.children.contracts(orgId, childId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.children.detail(orgId, childId) });
      toast({ title: t('contracts.updateSuccess') });
      setIsContractDialogOpen(false);
      setEditingContract(null);
      reset();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToSave', { resource: 'contract' })),
        variant: 'destructive',
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (contractId: number) => apiClient.deleteChildContract(orgId, childId, contractId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.children.contracts(orgId, childId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.children.detail(orgId, childId) });
      toast({ title: t('contracts.deleteSuccess') });
      setIsDeleteDialogOpen(false);
      setDeletingContract(null);
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToDelete', { resource: 'contract' })),
        variant: 'destructive',
      });
    },
  });

  const {
    register,
    handleSubmit,
    reset,
    control,
    watch,
    formState: { errors },
  } = useForm<ChildContractFormData>({
    resolver: zodResolver(childContractSchema),
    defaultValues: {
      from: '',
      to: '',
      properties: undefined,
    },
  });

  // Get org state for school enrollment date calculation
  const orgState = useUiStore((state) => state.organizations.find((o) => o.id === orgId)?.state);

  // Watch date fields for funding attribute suggestions
  const watchedFrom = watch('from');
  const watchedTo = watch('to');

  // Get funding attributes from government funding
  const { fundingAttributes, attributesByKey } = useFundingAttributes(
    orgId,
    watchedFrom,
    watchedTo
  );

  const handleCreate = () => {
    setEditingContract(null);
    // Auto-fill end date based on birthdate + org state
    let suggestedTo = '';
    if (child && orgState) {
      const birthdate = formatDateForInput(child.birthdate);
      if (birthdate) {
        suggestedTo = calculateContractEndDate(birthdate, orgState) || '';
      }
    }
    reset({ from: '', to: suggestedTo, properties: undefined });
    setIsContractDialogOpen(true);
  };

  const handleEdit = (contract: ChildContract) => {
    setEditingContract(contract);
    reset({
      from: formatDateForInput(contract.from),
      to: contract.to ? formatDateForInput(contract.to) : '',
      properties: contract.properties as Record<string, string> | undefined,
    });
    setIsContractDialogOpen(true);
  };

  const handleDelete = (contract: ChildContract) => {
    setDeletingContract(contract);
    setIsDeleteDialogOpen(true);
  };

  const onSubmit = (data: ChildContractFormData) => {
    if (editingContract) {
      updateMutation.mutate({
        contractId: editingContract.id,
        data: {
          from: formatDateForApi(data.from) || undefined,
          to: formatDateForApi(data.to) || undefined,
          properties: data.properties,
        },
      });
    } else {
      createMutation.mutate({
        from: formatDateForApi(data.from) || data.from,
        to: formatDateForApi(data.to),
        properties: data.properties,
      });
    }
  };

  const isLoading = childLoading || contractsLoading;

  // Sort contracts by start date descending (most recent first)
  const sortedContracts = contracts
    ? [...contracts].sort((a, b) => compareDates(b.from, a.from))
    : [];

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => router.push(`/organizations/${orgId}/children`)}
        >
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t('children.contractHistory')}</h1>
          {child && (
            <p className="text-muted-foreground">
              {child.first_name} {child.last_name}
            </p>
          )}
        </div>
        <div className="ml-auto">
          <Button onClick={handleCreate}>
            <Plus className="mr-2 h-4 w-4" />
            {t('contracts.newContract')}
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('contracts.title')}</CardTitle>
          <CardDescription>
            {sortedContracts.length > 0
              ? t('children.contractHistory')
              : t('children.noContractsFound')}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              {[...Array(3)].map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('common.status')}</TableHead>
                  <TableHead>{t('sections.title')}</TableHead>
                  <TableHead>{t('contracts.from')}</TableHead>
                  <TableHead>{t('contracts.to')}</TableHead>
                  <TableHead>{t('children.properties')}</TableHead>
                  <TableHead className="text-right">{t('common.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sortedContracts.map((contract) => {
                  const status = getContractStatus(contract);
                  return (
                    <TableRow key={contract.id}>
                      <TableCell>
                        <Badge
                          variant={
                            status === 'active'
                              ? 'success'
                              : status === 'upcoming'
                                ? 'warning'
                                : 'secondary'
                          }
                        >
                          {status === 'active'
                            ? t('common.active')
                            : status === 'upcoming'
                              ? t('common.upcoming')
                              : t('common.ended')}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {contract.section_name ? (
                          <Badge variant="outline">{contract.section_name}</Badge>
                        ) : (
                          <span className="text-sm text-muted-foreground">
                            {t('sections.unassigned')}
                          </span>
                        )}
                      </TableCell>
                      <TableCell>{formatDate(contract.from)}</TableCell>
                      <TableCell>
                        {contract.to ? formatDate(contract.to) : t('common.ongoing')}
                      </TableCell>
                      <TableCell>
                        {contract.properties && Object.keys(contract.properties).length > 0 ? (
                          <div className="flex flex-wrap gap-1">
                            {propertiesToValues(contract.properties as ContractProperties).map(
                              (value) => (
                                <Badge key={value} variant="outline" className="text-xs">
                                  {value}
                                </Badge>
                              )
                            )}
                          </div>
                        ) : (
                          <span className="text-sm text-muted-foreground">
                            {t('contracts.noProperties')}
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleEdit(contract)}
                          aria-label={t('common.edit')}
                        >
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleDelete(contract)}
                          aria-label={t('common.delete')}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
                {sortedContracts.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center text-muted-foreground">
                      {t('children.noContractsFound')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Contract Create/Edit Dialog */}
      <Dialog open={isContractDialogOpen} onOpenChange={setIsContractDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editingContract ? t('contracts.edit') : t('contracts.create')}
            </DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="from">{t('contracts.startDate')}</Label>
                <Input id="from" type="date" {...register('from')} />
                {errors.from && (
                  <p className="text-sm text-destructive">{t('contracts.startDateRequired')}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="to">{t('contracts.endDateOptional')}</Label>
                <Input id="to" type="date" {...register('to')} />
                {!editingContract && child && orgState && (
                  <p className="text-xs text-muted-foreground">{t('children.contractEndHint')}</p>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="properties">{t('contracts.propertiesLabel')}</Label>
              <Controller
                name="properties"
                control={control}
                render={({ field }) => (
                  <PropertyTagInput
                    id="properties"
                    value={field.value}
                    onChange={field.onChange}
                    fundingAttributes={fundingAttributes}
                    attributesByKey={attributesByKey}
                    placeholder={t('contracts.propertiesPlaceholder')}
                    suggestionsLabel={t('contracts.suggestedProperties')}
                  />
                )}
              />
              <p className="text-xs text-muted-foreground">{t('contracts.propertiesHelp')}</p>
            </div>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setIsContractDialogOpen(false)}
              >
                {t('common.cancel')}
              </Button>
              <Button type="submit" disabled={createMutation.isPending || updateMutation.isPending}>
                {t('common.save')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('common.confirmDelete')}</AlertDialogTitle>
            <AlertDialogDescription>{t('contracts.deleteConfirm')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deletingContract && deleteMutation.mutate(deletingContract.id)}
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
