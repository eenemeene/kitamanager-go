'use client';

import { useState, useCallback } from 'react';
import { useCrudDialogs } from '@/lib/hooks/use-crud-dialogs';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Plus, Pencil, Trash2, FileText, History } from 'lucide-react';
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
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { SearchInput } from '@/components/ui/search-input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useCrudMutations } from '@/lib/hooks/use-crud-mutations';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import {
  type Child,
  type ChildContract,
  type ChildContractCreateRequest,
  type ChildContractUpdateRequest,
  type ChildFundingResponse,
  type ContractProperties,
  LOOKUP_FETCH_LIMIT,
} from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  formatDate,
  calculateAge,
  formatDateForInput,
  formatDateForApi,
  formatCurrency,
  formatFte,
  propertiesToValues,
} from '@/lib/utils/formatting';
import { getCurrentContract } from '@/lib/utils/contracts';
import { useContractMutation } from '@/lib/hooks/use-contract-mutation';
import { Pagination } from '@/components/ui/pagination';
import { useDebouncedValue } from '@/lib/hooks/use-debounced-value';
import { DeleteConfirmDialog } from '@/components/crud/delete-confirm-dialog';
import { QueryError } from '@/components/crud/query-error';
import { PersonFormDialog } from '@/components/crud/person-form-dialog';
import { ChildCreateDialog } from '@/components/children/child-create-dialog';
import { ChildContractCreateDialog } from '@/components/children/child-contract-create-dialog';
import { useUiStore } from '@/stores/ui-store';
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
  const [isContractDialogOpen, setIsContractDialogOpen] = useState(false);
  const [contractChild, setContractChild] = useState<Child | null>(null);
  const [page, setPage] = useState(1);
  const [searchInput, setSearchInput] = useState('');
  const search = useDebouncedValue(searchInput, 300);
  const [sectionFilter, setSectionFilter] = useState<number | undefined>(undefined);

  const {
    data: paginatedData,
    isLoading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: queryKeys.children.list(orgId, page, search, sectionFilter),
    queryFn: () =>
      apiClient.getChildren(orgId, {
        page,
        search: search || undefined,
        section_id: sectionFilter,
      }),
    enabled: !!orgId,
  });

  const children = paginatedData?.data;

  // Fetch funding data for all children
  const { data: fundingData } = useQuery({
    queryKey: queryKeys.children.funding(orgId),
    queryFn: () => apiClient.getChildrenFunding(orgId),
    enabled: !!orgId,
    staleTime: 5 * 60 * 1000, // 5 minutes - funding data changes infrequently
  });

  // Create a map for quick lookup of funding by child ID
  const fundingByChildId = new Map<number, ChildFundingResponse>(
    fundingData?.children?.map((f) => [f.child_id, f]) ?? []
  );

  // Fetch sections for section selector in dialogs
  const { data: sectionsData } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });

  const sections = sectionsData?.data ?? [];

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
        to: formatDateForApi(data.contract_to),
        section_id: data.section_id,
        properties: data.properties,
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
    ChildContractUpdateRequest,
    ChildContract
  >({
    createFn: (childId, data) => apiClient.createChildContract(orgId, childId, data),
    updateFn: (childId, contractId, data) =>
      apiClient.updateChildContract(orgId, childId, contractId, data),
    toUpdateData: ({ from, ...rest }) => rest,
    invalidateQueryKeys: [queryKeys.children.all(orgId)],
    extraInvalidateKeys: (childId) => [queryKeys.children.contracts(orgId, childId)],
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

  const handleViewContractHistory = useCallback(
    (child: Child) => {
      router.push(`/organizations/${orgId}/children/${child.id}/contracts`);
    },
    [router, orgId]
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
          to: formatDateForApi(data.to),
          section_id: data.section_id,
          properties: data.properties,
        },
        entity: child,
        endCurrentContract,
      });
    },
    [createContractMutation]
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t('children.title')}</h1>
        </div>
        <Button onClick={dialogs.handleCreate}>
          <Plus className="mr-2 h-4 w-4" />
          {t('children.newChild')}
        </Button>
      </div>

      <div className="flex items-center gap-4">
        <SearchInput
          id="search-children"
          value={searchInput}
          onChange={(value) => {
            setSearchInput(value);
            setPage(1);
          }}
        />
        <Select
          value={sectionFilter ? String(sectionFilter) : 'all'}
          onValueChange={(value) => {
            setSectionFilter(value === 'all' ? undefined : Number(value));
            setPage(1);
          }}
        >
          <SelectTrigger className="w-[200px]">
            <SelectValue placeholder={t('statistics.filterBySection')} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t('statistics.allSections')}</SelectItem>
            {sections.map((section) => (
              <SelectItem key={section.id} value={String(section.id)}>
                {section.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
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
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('common.name')}</TableHead>
                  <TableHead>{t('gender.label')}</TableHead>
                  <TableHead>{t('children.birthdate')}</TableHead>
                  <TableHead>{t('children.age')}</TableHead>
                  <TableHead>{t('sections.title')}</TableHead>
                  <TableHead>{t('children.properties')}</TableHead>
                  <TableHead className="text-right">{t('children.funding')}</TableHead>
                  <TableHead className="text-right">
                    {t('children.requirement')}
                    {fundingData?.weekly_hours_basis ? ` (${fundingData.weekly_hours_basis}h)` : ''}
                  </TableHead>
                  <TableHead className="text-right">{t('common.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {children?.map((child) => {
                  const currentContract = getCurrentContract(child.contracts);
                  return (
                    <TableRow key={child.id}>
                      <TableCell className="font-medium">
                        {child.first_name} {child.last_name}
                      </TableCell>
                      <TableCell>{t(`gender.${child.gender}`)}</TableCell>
                      <TableCell>{formatDate(child.birthdate)}</TableCell>
                      <TableCell>{calculateAge(child.birthdate)}</TableCell>
                      <TableCell>
                        {currentContract?.section_name && (
                          <span>{currentContract.section_name}</span>
                        )}
                      </TableCell>
                      <TableCell>
                        {currentContract?.properties &&
                        Object.keys(currentContract.properties).length > 0 ? (
                          <div className="flex flex-wrap gap-1">
                            {propertiesToValues(currentContract.properties as ContractProperties)
                              .slice(0, 3)
                              .map((value) => (
                                <Badge key={value} variant="outline" className="text-xs">
                                  {value}
                                </Badge>
                              ))}
                            {Object.keys(currentContract.properties).length > 3 && (
                              <Badge variant="outline" className="text-xs">
                                +{Object.keys(currentContract.properties).length - 3}
                              </Badge>
                            )}
                          </div>
                        ) : (
                          <span className="text-sm text-muted-foreground">
                            {t('contracts.noProperties')}
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        {(() => {
                          const funding = fundingByChildId.get(child.id);
                          if (!funding || funding.funding === 0) {
                            return <span className="text-sm text-muted-foreground">-</span>;
                          }
                          return (
                            <span className="font-medium">{formatCurrency(funding.funding)}</span>
                          );
                        })()}
                      </TableCell>
                      <TableCell className="text-right">
                        {(() => {
                          const funding = fundingByChildId.get(child.id);
                          if (!funding || funding.requirement === 0) {
                            return <span className="text-sm text-muted-foreground">-</span>;
                          }
                          return (
                            <span className="font-medium">{formatFte(funding.requirement)}</span>
                          );
                        })()}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleViewContractHistory(child)}
                          title={t('children.contractHistory')}
                          aria-label={t('children.contractHistory')}
                        >
                          <History className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleAddContract(child)}
                          title={t('children.addContract')}
                          aria-label={t('children.addContract')}
                        >
                          <FileText className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => dialogs.handleEdit(child)}
                          aria-label={t('common.edit')}
                        >
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => dialogs.handleDelete(child)}
                          aria-label={t('common.delete')}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
                {children?.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={9} className="text-center text-muted-foreground">
                      {t('common.noResults')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
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
