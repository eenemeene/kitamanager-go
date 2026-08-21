'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { format, parseISO, subDays } from 'date-fns';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ContractTimeline } from '@/components/contracts/contract-timeline';
import { useResourceMutation } from '@/lib/hooks/use-resource-mutation';
import { useFundingAttributes } from '@/lib/hooks/use-funding-attributes';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import {
  type ChildContract,
  type ChildContractCreateRequest,
  type ChildContractCorrectRequest,
  type ContractProperties,
  type ContractBoundaryMoveRequest,
  LOOKUP_FETCH_LIMIT,
} from '@/lib/api/types';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { formatDateForInput, formatDateForApi } from '@/lib/utils/formatting';
import { propertiesToLabelKeys } from '@/lib/utils/contract-properties';
import { suggestContractEnd } from '@/lib/utils/school-enrollment';
import { getContractStatus, compareDates } from '@/lib/utils/contracts';
import { childContractSchema, type ChildContractFormData } from '@/lib/schemas';
import { useToast } from '@/lib/hooks/use-toast';
import { showErrorToast } from '@/lib/utils/show-error-toast';
import { useUiStore } from '@/stores/ui-store';
import { validationTiming } from '@/lib/forms/validation-timing';
import { useProblemFormErrors, suppressesToast } from '@/lib/forms/use-problem-form-errors';
import { FormErrorSummary } from '@/components/forms/form-error-summary';
import { useFormatters } from '@/hooks/use-formatters';

export default function ChildContractsPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const childId = Number(params.childId);
  const t = useTranslations();
  const fmt = useFormatters();
  const tLabels = useTranslations('fundingLabels');
  const queryClient = useQueryClient();
  const { toast } = useToast();

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
    onMutationError: suppressesToast,
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

  const correctMutation = useResourceMutation({
    onMutationError: suppressesToast,
    // The whole contract, not just its id: correcting it needs the version as an
    // If-Match precondition, so an edit cannot silently overwrite someone else's.
    mutationFn: ({
      contract,
      data,
    }: {
      contract: ChildContract;
      data: ChildContractCorrectRequest;
    }) => apiClient.correctChildContract(orgId, childId, contract.id, contract.version, data),
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
    mutationFn: (contract: ChildContract) =>
      apiClient.deleteChildContract(orgId, childId, contract.id, contract.version),
    invalidateQueryKey: invalidateKeys,
    successMessage: t('contracts.deleteSuccess'),
    errorMessage: t('common.failedToDelete', { resource: 'contract' }),
    onSuccess: () => {
      setIsDeleteDialogOpen(false);
      setDeletingContract(null);
    },
  });

  const contractsQueryKey = queryKeys.children.contracts(orgId, childId);

  const boundaryMutation = useMutation({
    mutationFn: (move: ContractBoundaryMoveRequest) =>
      apiClient.moveChildContractBoundary(orgId, childId, move),
    onMutate: async (move) => {
      await queryClient.cancelQueries({ queryKey: contractsQueryKey });
      const previous = queryClient.getQueryData<ChildContract[]>(contractsQueryKey);
      // Mirror what the server will do: the later contract starts at the seam,
      // the earlier one ends the day before. Only these two dates change, which
      // is the point of sending one date instead of four.
      const dayBefore =
        formatDateForApi(format(subDays(parseISO(move.at), 1), 'yyyy-MM-dd')) ?? undefined;
      queryClient.setQueryData<ChildContract[]>(contractsQueryKey, (old) =>
        old?.map((c) =>
          c.id === move.later_id
            ? { ...c, from: move.at }
            : c.id === move.earlier_id
              ? { ...c, to: dayBefore ?? c.to }
              : c
        )
      );
      return { previous };
    },
    onError: (error: unknown, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(contractsQueryKey, context.previous);
      }
      showErrorToast(t('common.error'), error, t('timeline.boundaryUpdateFailed'));
    },
    onSuccess: () => {
      toast({ title: t('timeline.boundaryUpdated') });
    },
    onSettled: () => {
      for (const key of invalidateKeys) {
        queryClient.invalidateQueries({ queryKey: key });
      }
    },
  });

  const {
    register,
    handleSubmit,
    reset,
    control,
    watch,
    setValue,
    setError,
    clearErrors,
    getValues,
    formState: { errors },
  } = useForm<ChildContractFormData>({
    ...validationTiming,
    resolver: zodResolver(childContractSchema),
    defaultValues: {
      from: '',
      to: '',
      properties: undefined,
    },
  });

  // Create and correct both submit this one dialog.
  const unmappedViolations = useProblemFormErrors([createMutation.error, correctMutation.error], {
    setError,
    clearErrors,
    getValues,
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
        suggestedTo = suggestContractEnd(birthdate, orgState, undefined, child.school_entry_date);
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
      correctMutation.mutate({
        contract: editingContract,
        // section_id is included because the dialog renders a section select and
        // the old payload left it out: changing a contract's section here was
        // silently discarded. `to` is sent as null rather than omitted when the
        // field is empty, because omitting now means "leave alone" — clearing an
        // end date has to be said explicitly.
        data: {
          from: formatDateForApi(data.from) || undefined,
          to: formatDateForApi(data.to),
          section_id: data.section_id,
          properties: data.properties as ContractProperties | undefined,
        },
      });
    } else {
      createMutation.mutate({
        from: formatDateForApi(data.from) || data.from,
        to: formatDateForApi(data.to) ?? undefined,
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
        <Tabs defaultValue="table">
          <CardHeader>
            <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
              <div>
                <CardTitle>{t('contracts.title')}</CardTitle>
                <CardDescription>
                  {sortedContracts.length > 0
                    ? t('children.contractHistory')
                    : t('children.noContractsFound')}
                </CardDescription>
              </div>
              <TabsList>
                <TabsTrigger value="table">{t('timeline.tableView')}</TabsTrigger>
                <TabsTrigger value="timeline">{t('timeline.timelineView')}</TabsTrigger>
              </TabsList>
            </div>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-2">
                {[...Array(3)].map((_, i) => (
                  <Skeleton key={i} className="h-12 w-full" />
                ))}
              </div>
            ) : (
              <>
                <TabsContent value="table" className="mt-0">
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
                            <TableCell>{fmt.date(contract.from)}</TableCell>
                            <TableCell>
                              {contract.to ? fmt.date(contract.to) : t('common.ongoing')}
                            </TableCell>
                            <TableCell>
                              {contract.properties &&
                              Object.keys(contract.properties).length > 0 ? (
                                <div className="flex flex-wrap gap-1">
                                  {propertiesToLabelKeys(
                                    contract.properties as ContractProperties
                                  ).map((labelKey) => (
                                    <Badge key={labelKey} variant="outline" className="text-xs">
                                      {tLabels.has(labelKey)
                                        ? tLabels(labelKey)
                                        : labelKey.split('--').pop()}
                                    </Badge>
                                  ))}
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
                </TabsContent>
                <TabsContent value="timeline" className="mt-0">
                  <ContractTimeline
                    contracts={sortedContracts}
                    renderSegmentContent={(contract) => (
                      <div className="space-y-1.5">
                        {contract.section_name && (
                          <div className="flex items-center gap-1.5">
                            <span className="text-muted-foreground text-xs">
                              {t('sections.title')}:
                            </span>
                            <Badge variant="outline">{contract.section_name}</Badge>
                          </div>
                        )}
                        {contract.properties && Object.keys(contract.properties).length > 0 && (
                          <div className="flex flex-wrap gap-1">
                            {propertiesToLabelKeys(contract.properties as ContractProperties).map(
                              (labelKey) => (
                                <Badge key={labelKey} variant="outline" className="text-xs">
                                  {tLabels.has(labelKey)
                                    ? tLabels(labelKey)
                                    : labelKey.split('--').pop()}
                                </Badge>
                              )
                            )}
                          </div>
                        )}
                      </div>
                    )}
                    onBoundaryChange={(move) => boundaryMutation.mutateAsync(move)}
                    isUpdating={boundaryMutation.isPending}
                  />
                </TabsContent>
              </>
            )}
          </CardContent>
        </Tabs>
      </Card>

      <CrudFormDialog
        open={isContractDialogOpen}
        onOpenChange={setIsContractDialogOpen}
        isEditing={!!editingContract}
        translationPrefix="contracts"
        onSubmit={handleSubmit(onSubmit)}
        isSaving={createMutation.isPending || correctMutation.isPending}
      >
        <FormErrorSummary
          errors={errors}
          unmapped={unmappedViolations}
          labels={{
            from: t('contracts.startDate'),
            to: t('contracts.endDateOptional'),
            section_id: t('sections.title'),
            properties: t('contracts.propertiesLabel'),
          }}
        />
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="from">{t('contracts.startDate')}</Label>
            <Input id="from" type="date" aria-invalid={!!errors.from} {...register('from')} />
            {errors.from && (
              <p className="text-destructive text-sm">{t('contracts.startDateRequired')}</p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="to">{t('contracts.endDateOptional')}</Label>
            <Input id="to" type="date" aria-invalid={!!errors.to} {...register('to')} />
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
          <Label id="properties-label">{t('contracts.propertiesLabel')}</Label>
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
        onConfirm={() => deletingContract && deleteMutation.mutate(deletingContract)}
        isLoading={deleteMutation.isPending}
        resourceName="contracts"
      />
    </div>
  );
}
