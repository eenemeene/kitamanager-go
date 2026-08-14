'use client';

import { useState, useCallback, useEffect, useMemo } from 'react';
import { useCrudDialogs } from '@/lib/hooks/use-crud-dialogs';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Download, Upload, Baby } from 'lucide-react';
import { MonthStepper } from '@/components/ui/month-stepper';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ChildrenTable } from '@/components/children/children-table';
import { Skeleton } from '@/components/ui/skeleton';
import { SearchInput } from '@/components/ui/search-input';
import { SectionFilter } from '@/components/ui/section-filter';
import { useCrudMutations } from '@/lib/hooks/use-crud-mutations';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import {
  type Child,
  type ChildContract,
  type ChildContractCreateRequest,
  type ChildContractAmendRequest,
  type ChildContractAmendResponse,
  type ChildFundingResponse,
  type ChildBillingSummaryEntry,
  type ContractProperties,
  LOOKUP_FETCH_LIMIT,
} from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  formatDate,
  formatDateForInput,
  formatDateForApi,
  toLocalDateString,
} from '@/lib/utils/formatting';
import { showErrorToast } from '@/lib/utils/show-error-toast';
import { useContractMutation } from '@/lib/hooks/use-contract-mutation';
import { useImportMutation } from '@/lib/hooks/use-import-mutation';
import { useResourceListFilters } from '@/lib/hooks/use-resource-list-filters';
import { Pagination } from '@/components/ui/pagination';
import { DeleteConfirmDialog } from '@/components/crud/delete-confirm-dialog';
import { QueryError } from '@/components/crud/query-error';
import { EmptyState } from '@/components/crud/empty-state';
import { PersonFormDialog } from '@/components/crud/person-form-dialog';
import { ChildCreateDialog } from '@/components/children/child-create-dialog';
import { ChildContractCreateDialog } from '@/components/children/child-contract-create-dialog';
import { VouchersDialog } from '@/components/children/vouchers-dialog';
import { useToast } from '@/lib/hooks/use-toast';
import { useUiStore } from '@/stores/ui-store';
import { validationTiming } from '@/lib/forms/validation-timing';
import {
  childSchema,
  type ChildFormData,
  type ChildContractFormData,
  type ChildWithContractFormData,
} from '@/lib/schemas';

export default function ChildrenPage() {
  const params = useParams();
  const router = useRouter();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const { toast } = useToast();

  const {
    fileInputRef,
    triggerFileInput,
    handleFileChange,
    isPending: isImporting,
  } = useImportMutation({
    importFn: (file) => apiClient.importChildren(orgId, file),
    invalidateQueryKeys: [queryKeys.children.all(orgId), queryKeys.statistics.all(orgId)],
    resourceNameKey: 'children.title',
    errorMessageKey: 'children.importError',
  });

  const [isContractDialogOpen, setIsContractDialogOpen] = useState(false);
  const [contractChild, setContractChild] = useState<Child | null>(null);
  const { page, setPage, searchInput, setSearchInput, search, activeOn, setActiveOn } =
    useResourceListFilters();
  const [sectionFilter, setSectionFilter] = useState<number | undefined>(undefined);

  const {
    data: paginatedData,
    isLoading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: queryKeys.children.list(
      orgId,
      page,
      search,
      sectionFilter,
      toLocalDateString(activeOn)
    ),
    queryFn: () =>
      apiClient.getChildren(orgId, {
        page,
        search: search || undefined,
        section_id: sectionFilter,
        active_on: toLocalDateString(activeOn),
      }),
    enabled: !!orgId,
  });

  const children = paginatedData?.data;

  // Fetch funding data for all children
  const { data: fundingData, error: fundingError } = useQuery({
    queryKey: queryKeys.children.funding(orgId),
    queryFn: () => apiClient.getChildrenFunding(orgId),
    enabled: !!orgId,
    staleTime: 5 * 60 * 1000, // 5 minutes - funding data changes infrequently
  });

  // Create a map for quick lookup of funding by child ID
  const fundingByChildId = useMemo(
    () =>
      new Map<number, ChildFundingResponse>(
        fundingData?.children?.map((f) => [f.child_id, f]) ?? []
      ),
    [fundingData]
  );

  // Fetch billing summary for all children
  const { data: billingSummaryData, error: billingSummaryError } = useQuery({
    queryKey: queryKeys.children.billingSummary(orgId),
    queryFn: () => apiClient.getChildrenBillingSummary(orgId),
    enabled: !!orgId,
    staleTime: 5 * 60 * 1000,
  });

  const billingSummaryByChildId = useMemo(
    () =>
      new Map<number, ChildBillingSummaryEntry>(
        billingSummaryData?.children?.map((b) => [b.child_id, b] as const) ?? []
      ),
    [billingSummaryData]
  );

  // Fetch sections for section selector in dialogs
  const { data: sectionsData, error: sectionsError } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });

  const sections = sectionsData?.data ?? [];

  // Show toast on secondary query failures
  useEffect(() => {
    const err = fundingError || sectionsError;
    if (err) {
      toast({
        title: t('common.error'),
        description: t('common.failedToLoad', { resource: t('common.data') }),
        variant: 'destructive',
      });
    }
  }, [fundingError, sectionsError, toast, t]);

  // Get org state for school enrollment date calculation
  const orgState = useUiStore((state) => state.organizations.find((o) => o.id === orgId)?.state);

  const mutations = useCrudMutations<Child, ChildWithContractFormData, Partial<ChildFormData>>({
    resourceName: 'children',
    queryKey: queryKeys.children.all(orgId),
    createFn: async (data) => {
      const child = await apiClient.createChild(orgId, {
        first_name: data.first_name,
        last_name: data.last_name,
        gender: data.gender,
        birthdate: data.birthdate,
      });
      await apiClient.createChildContract(orgId, child.id, {
        from: formatDateForApi(data.contract_from) || data.contract_from,
        to: formatDateForApi(data.contract_to) ?? undefined,
        section_id: data.section_id,
        properties: data.properties as ContractProperties | undefined,
      });
      return child;
    },
    updateFn: (id, data) => apiClient.updateChild(orgId, id, data),
    deleteFn: (id) => apiClient.deleteChild(orgId, id),
    onSuccess: () => dialogs.closeDialog(),
    onDeleteSuccess: () => dialogs.closeDeleteDialog(),
  });

  const createContractMutation = useContractMutation<
    ChildContractCreateRequest,
    ChildContractAmendRequest,
    ChildContract,
    ChildContractAmendResponse
  >({
    createFn: (childId, data) => apiClient.createChildContract(orgId, childId, data),
    amendFn: (childId, contractId, version, data) =>
      apiClient.amendChildContract(orgId, childId, contractId, version, data),
    // The date the user picked in the dialog is the date the change takes effect,
    // including in the past. The old code stripped it and the server used today.
    toAmendData: ({ from, ...rest }) => ({ effective_from: from, ...rest }),
    invalidateQueryKeys: [
      queryKeys.children.all(orgId),
      queryKeys.children.allUnpaginated(orgId),
      queryKeys.statistics.contractProperties(orgId),
    ],
    extraInvalidateKeys: (childId) => [
      queryKeys.children.contracts(orgId, childId),
      queryKeys.children.detail(orgId, childId),
    ],
    onSuccess: () => {
      setIsContractDialogOpen(false);
      setContractChild(null);
    },
  });

  const {
    register: registerChild,
    handleSubmit: handleSubmitChild,
    reset: resetChild,
    setValue: setValueChild,
    watch: watchChild,
    formState: { errors: errorsChild },
  } = useForm<ChildFormData>({
    ...validationTiming,
    resolver: zodResolver(childSchema),
    defaultValues: {
      first_name: '',
      last_name: '',
      gender: 'male',
      birthdate: '',
    },
  });

  const dialogs = useCrudDialogs<Child, ChildFormData>({
    reset: resetChild,
    itemToFormData: (child) => ({
      first_name: child.first_name,
      last_name: child.last_name,
      gender: child.gender,
      birthdate: formatDateForInput(child.birthdate),
    }),
    defaultValues: { first_name: '', last_name: '', gender: 'male', birthdate: '' },
  });

  const handleAddContract = useCallback((child: Child) => {
    setContractChild(child);
    setIsContractDialogOpen(true);
  }, []);

  const [vouchersChild, setVouchersChild] = useState<Child | null>(null);
  const handleManageVouchers = useCallback((child: Child) => {
    setVouchersChild(child);
  }, []);

  const handleViewContractHistory = useCallback(
    (child: Child) => {
      router.push(`/organizations/${orgId}/children/${child.id}/contracts`);
    },
    [router, orgId]
  );

  const handleViewBillingHistory = useCallback(
    (child: Child) => {
      router.push(`/organizations/${orgId}/children/${child.id}/billing`);
    },
    [router, orgId]
  );

  const queryClient = useQueryClient();
  const adjustContractEndMutation = useMutation({
    mutationFn: ({
      childId,
      contract,
      to,
    }: {
      childId: number;
      contract: Pick<ChildContract, 'id' | 'version'>;
      to: string;
    }) =>
      apiClient.endChildContract(orgId, childId, contract.id, contract.version, {
        to: formatDateForApi(to),
      }),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.children.all(orgId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.children.billingSummary(orgId) });
      queryClient.invalidateQueries({
        queryKey: queryKeys.children.contracts(orgId, variables.childId),
      });
      toast({
        title: t('children.contractEndAdjusted', { date: formatDate(variables.to) }),
      });
    },
    onError: (error: unknown) => {
      showErrorToast(
        t('common.error'),
        error,
        t('common.failedToSave', { resource: t('contracts.title') })
      );
    },
  });
  const handleAdjustContractEnd = useCallback(
    (child: Child, contract: Pick<ChildContract, 'id' | 'version'>, to: string) => {
      adjustContractEndMutation.mutate({ childId: child.id, contract, to });
    },
    [adjustContractEndMutation]
  );

  const onSubmitChild = useCallback(
    (data: ChildFormData) => {
      if (dialogs.editingItem) {
        mutations.updateMutation.mutate({ id: dialogs.editingItem.id, data });
      }
    },
    [dialogs.editingItem, mutations.updateMutation]
  );

  const onSubmitCreate = useCallback(
    (data: ChildWithContractFormData) => {
      mutations.createMutation.mutate(data);
    },
    [mutations.createMutation]
  );

  const onSubmitContract = useCallback(
    (data: ChildContractFormData, child: Child, endCurrentContract: boolean) => {
      createContractMutation.mutate({
        entityId: child.id,
        data: {
          from: formatDateForApi(data.from) || data.from,
          to: formatDateForApi(data.to) ?? undefined,
          section_id: data.section_id,
          properties: data.properties as ContractProperties | undefined,
        },
        entity: child,
        endCurrentContract,
      });
    },
    [createContractMutation]
  );

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div className="min-w-0">
          <h1 className="text-3xl font-bold tracking-tight">{t('children.title')}</h1>
          <p className="text-muted-foreground mt-1 max-w-3xl text-sm">
            {t('children.description')}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button
            variant="outline"
            onClick={() => {
              window.open(
                apiClient.getChildrenExportUrl(orgId, {
                  search: search || undefined,
                  section_id: sectionFilter ? String(sectionFilter) : undefined,
                  active_on: toLocalDateString(activeOn),
                })
              );
            }}
          >
            <Download className="mr-2 h-4 w-4" />
            {t('common.exportExcel')}
          </Button>
          <Button
            variant="outline"
            onClick={() => window.open(apiClient.getChildrenExportYamlUrl(orgId))}
          >
            <Download className="mr-2 h-4 w-4" />
            {t('children.exportYaml')}
          </Button>
          <Button variant="outline" onClick={triggerFileInput} disabled={isImporting}>
            <Upload className="mr-2 h-4 w-4" />
            {isImporting ? t('children.importing') : t('children.importYaml')}
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            accept=".yaml,.yml"
            className="hidden"
            onChange={handleFileChange}
          />
          <Button onClick={dialogs.handleCreate}>
            <Plus className="mr-2 h-4 w-4" />
            {t('children.newChild')}
          </Button>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2 md:gap-4">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-muted-foreground text-sm">{t('common.activeOn')}</span>
          <MonthStepper value={activeOn} onChange={setActiveOn} />
        </div>
        <SearchInput id="search-children" value={searchInput} onChange={setSearchInput} />
        <SectionFilter
          sections={sections}
          value={sectionFilter}
          onChange={(id) => {
            setSectionFilter(id);
            setPage(1);
          }}
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('children.title')}</CardTitle>
        </CardHeader>
        <CardContent>
          {queryError ? (
            <QueryError error={queryError} onRetry={() => refetch()} />
          ) : isLoading ? (
            <div className="space-y-2">
              {[...Array(3)].map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : !search &&
            !sectionFilter &&
            paginatedData &&
            paginatedData.total === 0 &&
            (children?.length ?? 0) === 0 ? (
            <EmptyState
              icon={Baby}
              title="children.emptyTitle"
              description="children.emptyDescription"
              action={
                <>
                  <Button onClick={dialogs.handleCreate}>
                    <Plus className="mr-2 h-4 w-4" />
                    {t('children.emptyAction')}
                  </Button>
                  <Button variant="outline" onClick={triggerFileInput} disabled={isImporting}>
                    <Upload className="mr-2 h-4 w-4" />
                    {isImporting ? t('children.importing') : t('children.importYaml')}
                  </Button>
                </>
              }
            />
          ) : (
            <ChildrenTable
              items={children ?? []}
              fundingByChildId={fundingByChildId}
              weeklyHoursBasis={fundingData?.weekly_hours_basis}
              billingSummaryByChildId={billingSummaryByChildId}
              orgState={orgState}
              onViewHistory={handleViewContractHistory}
              onViewBilling={handleViewBillingHistory}
              onAddContract={handleAddContract}
              onEdit={dialogs.handleEdit}
              onDelete={dialogs.handleDelete}
              onManageVouchers={handleManageVouchers}
              onAdjustContractEnd={handleAdjustContractEnd}
              isAdjustingContractEnd={adjustContractEndMutation.isPending}
            />
          )}
          {paginatedData && (
            <Pagination
              page={paginatedData.page}
              totalPages={paginatedData.total_pages}
              total={paginatedData.total}
              limit={paginatedData.limit}
              onPageChange={setPage}
              isLoading={isLoading}
            />
          )}
        </CardContent>
      </Card>

      {/* Child Edit Dialog (uses PersonFormDialog) */}
      {dialogs.editingItem && (
        <PersonFormDialog
          open={dialogs.isDialogOpen}
          onOpenChange={dialogs.setIsDialogOpen}
          isEditing={true}
          register={registerChild}
          onSubmit={handleSubmitChild(onSubmitChild)}
          errors={errorsChild}
          watch={watchChild}
          setValue={setValueChild}
          isSaving={mutations.updateMutation.isPending}
          translationPrefix="children"
        />
      )}

      {/* Child Create Dialog (with initial contract) */}
      {!dialogs.editingItem && (
        <ChildCreateDialog
          open={dialogs.isDialogOpen}
          onOpenChange={dialogs.setIsDialogOpen}
          orgId={orgId}
          orgState={orgState}
          sections={sections}
          isSaving={mutations.createMutation.isPending}
          onSubmit={onSubmitCreate}
        />
      )}

      {/* Vouchers Dialog */}
      <VouchersDialog
        open={vouchersChild !== null}
        onOpenChange={(next) => {
          if (!next) setVouchersChild(null);
        }}
        orgId={orgId}
        child={vouchersChild}
      />

      {/* Contract Create Dialog */}
      <ChildContractCreateDialog
        open={isContractDialogOpen}
        onOpenChange={setIsContractDialogOpen}
        orgId={orgId}
        orgState={orgState}
        child={contractChild}
        sections={sections}
        isSaving={createContractMutation.isPending}
        onSubmit={onSubmitContract}
      />

      {/* Delete Confirmation Dialog */}
      <DeleteConfirmDialog
        open={dialogs.isDeleteDialogOpen}
        onOpenChange={dialogs.setIsDeleteDialogOpen}
        onConfirm={() =>
          dialogs.deletingItem && mutations.deleteMutation.mutate(dialogs.deletingItem.id)
        }
        isLoading={mutations.deleteMutation.isPending}
        resourceName="children"
        description={t('children.confirmDeleteMessage', {
          name: dialogs.deletingItem
            ? `${dialogs.deletingItem.first_name} ${dialogs.deletingItem.last_name}`
            : '',
        })}
      />
    </div>
  );
}
