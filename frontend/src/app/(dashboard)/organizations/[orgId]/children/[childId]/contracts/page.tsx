'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Pencil, Trash2, Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Breadcrumb } from '@/components/ui/breadcrumb';
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
import { DeleteConfirmDialog } from '@/components/crud/delete-confirm-dialog';
import { CrudFormDialog } from '@/components/crud/crud-form-dialog';
import { QueryError } from '@/components/crud/query-error';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { PropertyTagInput } from '@/components/ui/tag-input';
import { useResourceMutation } from '@/lib/hooks/use-resource-mutation';
import { useFundingAttributes } from '@/lib/hooks/use-funding-attributes';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import {
  type ChildContract,
  type ChildContractCreateRequest,
  type ChildContractUpdateRequest,
  type ContractProperties,
  LOOKUP_FETCH_LIMIT,
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
  const orgId = Number(params.orgId);
  const childId = Number(params.childId);
  const t = useTranslations();

  const [isContractDialogOpen, setIsContractDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [editingContract, setEditingContract] = useState<ChildContract | null>(null);
  const [deletingContract, setDeletingContract] = useState<ChildContract | null>(null);

  // Fetch child data
  const {
    data: child,
    isLoading: childLoading,
    error: childError,
    refetch: refetchChild,
  } = useQuery({
    queryKey: queryKeys.children.detail(orgId, childId),
    queryFn: () => apiClient.getChild(orgId, childId),
    enabled: !!orgId && !!childId,
  });

  // Fetch contracts
  const {
    data: contracts,
    isLoading: contractsLoading,
    error: contractsError,
    refetch: refetchContracts,
  } = useQuery({
    queryKey: queryKeys.children.contracts(orgId, childId),
    queryFn: () => apiClient.getChildContracts(orgId, childId),
    enabled: !!orgId && !!childId,
  });

  // Fetch sections for section selector
  const { data: sectionsData } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });

  const invalidateKeys = [
    queryKeys.children.contracts(orgId, childId),
    queryKeys.children.detail(orgId, childId),
    queryKeys.children.all(orgId),
    queryKeys.children.allUnpaginated(orgId),
    queryKeys.statistics.contractProperties(orgId),
  ];

  const createMutation = useResourceMutation({
    mutationFn: (data: ChildContractCreateRequest) =>
      apiClient.createChildContract(orgId, childId, data),
    invalidateQueryKey: invalidateKeys,
    successMessage: t('contracts.createSuccess'),
    errorMessage: t('common.failedToCreate', { resource: 'contract' }),
    onSuccess: () => {
      setIsContractDialogOpen(false);
      setEditingContract(null);
      reset();
    },
  });

  const updateMutation = useResourceMutation({
    mutationFn: ({ contractId, data }: { contractId: number; data: ChildContractUpdateRequest }) =>
      apiClient.updateChildContract(orgId, childId, contractId, data),
    invalidateQueryKey: invalidateKeys,
    successMessage: t('contracts.updateSuccess'),
    errorMessage: t('common.failedToSave', { resource: 'contract' }),
    onSuccess: () => {
      setIsContractDialogOpen(false);
      setEditingContract(null);
      reset();
    },
  });

  const deleteMutation = useResourceMutation({
    mutationFn: (contractId: number) => apiClient.deleteChildContract(orgId, childId, contractId),
    invalidateQueryKey: invalidateKeys,
    successMessage: t('contracts.deleteSuccess'),
    errorMessage: t('common.failedToDelete', { resource: 'contract' }),
    onSuccess: () => {
      setIsDeleteDialogOpen(false);
      setDeletingContract(null);
    },
  });

  const {
    register,
    handleSubmit,
    reset,
    control,
    watch,
    setValue,
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
    reset({ from: '', to: suggestedTo, section_id: 0, properties: undefined });
    setIsContractDialogOpen(true);
  };

  const handleEdit = (contract: ChildContract) => {
    setEditingContract(contract);
    reset({
      from: formatDateForInput(contract.from),
      to: contract.to ? formatDateForInput(contract.to) : '',
      section_id: contract.section_id,
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
          properties: data.properties as ContractProperties | undefined,
        },
      });
    } else {
      createMutation.mutate({
        from: formatDateForApi(data.from) || data.from,
        to: formatDateForApi(data.to),
        section_id: data.section_id,
        properties: data.properties as ContractProperties | undefined,
      });
    }
  };

  const isLoading = childLoading || contractsLoading;
  const queryError = childError || contractsError;

  // Sort contracts by start date descending (most recent first)
  const sortedContracts = contracts
    ? [...contracts].sort((a, b) => compareDates(b.from, a.from))
    : [];

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 md:flex-row md:items-center">
        <div className="min-w-0 flex-1">
          <Breadcrumb
            items={[
              { label: t('nav.children'), href: `/organizations/${orgId}/children` },
              {
                label: child ? `${child.first_name} ${child.last_name}` : '...',
              },
              { label: t('children.contractHistory') },
            ]}
          />
          <h1 className="mt-1 text-3xl font-bold tracking-tight">
            {t('children.contractHistory')}
          </h1>
        </div>
        <div className="shrink-0">
          <Button onClick={handleCreate}>
            <Plus className="mr-2 h-4 w-4" />
            {t('contracts.newContract')}
          </Button>
        </div>
      </div>

      <QueryError
        error={queryError}
        onRetry={() => {
          refetchChild();
          refetchContracts();
        }}
      />

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
                          <span className="text-muted-foreground text-sm">
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
                          <span className="text-muted-foreground text-sm">
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
                    <TableCell colSpan={6} className="text-muted-foreground text-center">
                      {t('children.noContractsFound')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <CrudFormDialog
        open={isContractDialogOpen}
        onOpenChange={setIsContractDialogOpen}
        isEditing={!!editingContract}
        translationPrefix="contracts"
        onSubmit={handleSubmit(onSubmit)}
        isSaving={createMutation.isPending || updateMutation.isPending}
      >
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="from">{t('contracts.startDate')}</Label>
            <Input id="from" type="date" {...register('from')} />
            {errors.from && (
              <p className="text-destructive text-sm">{t('contracts.startDateRequired')}</p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="to">{t('contracts.endDateOptional')}</Label>
            <Input id="to" type="date" {...register('to')} />
            {!editingContract && child && orgState && (
              <p className="text-muted-foreground text-xs">{t('children.contractEndHint')}</p>
            )}
          </div>
        </div>

        {sectionsData && sectionsData.data.length > 0 && (
          <div className="space-y-2">
            <Label htmlFor="contract_section">{t('sections.title')} *</Label>
            <Select
              value={watch('section_id')?.toString() || ''}
              onValueChange={(val) => setValue('section_id', val ? Number(val) : 0)}
            >
              <SelectTrigger id="contract_section">
                <SelectValue placeholder={t('sections.selectSection')} />
              </SelectTrigger>
              <SelectContent>
                {sectionsData.data.map((section) => (
                  <SelectItem key={section.id} value={section.id.toString()}>
                    {section.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {errors.section_id && (
              <p className="text-destructive text-sm">{t('validation.sectionRequired')}</p>
            )}
          </div>
        )}

        <div className="space-y-2">
          <Label htmlFor="properties">{t('contracts.propertiesLabel')}</Label>
          <Controller
            name="properties"
            control={control}
            render={({ field }) => (
              <PropertyTagInput
                id="properties"
                value={field.value as Record<string, string> | undefined}
                onChange={field.onChange}
                fundingAttributes={fundingAttributes}
                attributesByKey={attributesByKey}
                placeholder={t('contracts.propertiesPlaceholder')}
                suggestionsLabel={t('contracts.suggestedProperties')}
              />
            )}
          />
          <p className="text-muted-foreground text-xs">{t('contracts.propertiesHelp')}</p>
        </div>
      </CrudFormDialog>

      <DeleteConfirmDialog
        open={isDeleteDialogOpen}
        onOpenChange={setIsDeleteDialogOpen}
        onConfirm={() => deletingContract && deleteMutation.mutate(deletingContract.id)}
        isLoading={deleteMutation.isPending}
        resourceName="contracts"
      />
    </div>
  );
}
