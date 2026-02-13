'use client';

import { useState, useCallback, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Pencil, Trash2, FileText, History, Search } from 'lucide-react';
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
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useToast } from '@/lib/hooks/use-toast';
import { apiClient, getErrorMessage } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import type {
  Child,
  Gender,
  ChildContractCreateRequest,
  ChildContractUpdateRequest,
  ChildFundingResponse,
  ContractProperties,
} from '@/lib/api/types';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Checkbox } from '@/components/ui/checkbox';
import { PropertyTagInput } from '@/components/ui/tag-input';
import { useFundingAttributes } from '@/lib/hooks/use-funding-attributes';
import { useForm, Controller } from 'react-hook-form';
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
import {
  getActiveContract,
  getCurrentContract,
  getDayBefore,
  getContractStatus,
  isDateBefore,
} from '@/lib/utils/contracts';
import { calculateContractEndDate } from '@/lib/utils/school-enrollment';
import { Pagination } from '@/components/ui/pagination';
import { useDebouncedValue } from '@/lib/hooks/use-debounced-value';
import { DeleteConfirmDialog } from '@/components/crud/delete-confirm-dialog';
import { QueryError } from '@/components/crud/query-error';
import { PersonFormDialog } from '@/components/crud/person-form-dialog';
import { useUiStore } from '@/stores/ui-store';
import {
  childSchema,
  childContractSchema,
  childWithContractSchema,
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
  const queryClient = useQueryClient();

  const [isChildDialogOpen, setIsChildDialogOpen] = useState(false);
  const [isContractDialogOpen, setIsContractDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [editingChild, setEditingChild] = useState<Child | null>(null);
  const [deletingChild, setDeletingChild] = useState<Child | null>(null);
  const [contractChild, setContractChild] = useState<Child | null>(null);
  const [endCurrentContract, setEndCurrentContract] = useState(true);
  const [page, setPage] = useState(1);
  const [searchInput, setSearchInput] = useState('');
  const search = useDebouncedValue(searchInput, 300);

  const {
    data: paginatedData,
    isLoading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: queryKeys.children.list(orgId, page, search),
    queryFn: () => apiClient.getChildren(orgId, { page, search: search || undefined }),
    enabled: !!orgId,
  });

  const children = paginatedData?.data;

  // Fetch funding data for all children
  const { data: fundingData } = useQuery({
    queryKey: queryKeys.children.funding(orgId),
    queryFn: () => apiClient.getChildrenFunding(orgId),
    enabled: !!orgId,
    staleTime: 60 * 1000, // 1 minute - funding doesn't change often
  });

  // Create a map for quick lookup of funding by child ID
  const fundingByChildId = new Map<number, ChildFundingResponse>(
    fundingData?.children?.map((f) => [f.child_id, f]) ?? []
  );

  // Get org state for school enrollment date calculation
  const orgState = useUiStore((state) => state.organizations.find((o) => o.id === orgId)?.state);

  const createWithContractMutation = useMutation({
    mutationFn: async (data: ChildWithContractFormData) => {
      const child = await apiClient.createChild(orgId, {
        first_name: data.first_name,
        last_name: data.last_name,
        gender: data.gender as Gender,
        birthdate: data.birthdate,
      });
      await apiClient.createChildContract(orgId, child.id, {
        from: formatDateForApi(data.contract_from) || data.contract_from,
        to: formatDateForApi(data.contract_to),
        properties: data.properties,
      });
      return child;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.children.all(orgId) });
      toast({ title: t('children.createSuccess') });
      setIsChildDialogOpen(false);
      resetCreate();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToCreate', { resource: 'child' })),
        variant: 'destructive',
      });
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<ChildFormData> }) =>
      apiClient.updateChild(orgId, id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.children.all(orgId) });
      toast({ title: t('children.updateSuccess') });
      setIsChildDialogOpen(false);
      setEditingChild(null);
      resetChild();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToSave', { resource: 'child' })),
        variant: 'destructive',
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => apiClient.deleteChild(orgId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.children.all(orgId) });
      toast({ title: t('children.deleteSuccess') });
      setIsDeleteDialogOpen(false);
      setDeletingChild(null);
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToDelete', { resource: 'child' })),
        variant: 'destructive',
      });
    },
  });

  const updateContractMutation = useMutation({
    mutationFn: ({
      childId,
      contractId,
      data,
    }: {
      childId: number;
      contractId: number;
      data: ChildContractUpdateRequest;
    }) => apiClient.updateChildContract(orgId, childId, contractId, data),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.children.all(orgId) });
      queryClient.invalidateQueries({
        queryKey: queryKeys.children.contracts(orgId, variables.childId),
      });
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToSave', { resource: 'contract' })),
        variant: 'destructive',
      });
    },
  });

  const createContractMutation = useMutation({
    mutationFn: async ({
      childId,
      data,
    }: {
      childId: number;
      data: ChildContractCreateRequest;
    }) => {
      // If we need to end the current contract first
      if (contractChild && endCurrentContract) {
        const active = getActiveContract(contractChild.contracts);
        if (active && data.from) {
          const endDate = getDayBefore(data.from);
          await apiClient.updateChildContract(orgId, childId, active.id, {
            to: formatDateForApi(endDate),
          });
        }
      }
      return apiClient.createChildContract(orgId, childId, data);
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.children.all(orgId) });
      queryClient.invalidateQueries({
        queryKey: queryKeys.children.contracts(orgId, variables.childId),
      });
      toast({
        title: endCurrentContract
          ? t('contracts.previousContractEnded')
          : t('contracts.createSuccess'),
      });
      setIsContractDialogOpen(false);
      setContractChild(null);
      setEndCurrentContract(true);
      resetContract();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToCreate', { resource: 'contract' })),
        variant: 'destructive',
      });
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

  const {
    register: registerContract,
    handleSubmit: handleSubmitContract,
    reset: resetContract,
    watch: watchContract,
    control: controlContract,
    formState: { errors: errorsContract },
  } = useForm<ChildContractFormData>({
    resolver: zodResolver(childContractSchema),
    defaultValues: {
      from: '',
      to: '',
      properties: undefined,
    },
  });

  // Combined child + contract form (for creating new children)
  const {
    register: registerCreate,
    handleSubmit: handleSubmitCreate,
    reset: resetCreate,
    setValue: setValueCreate,
    watch: watchCreate,
    control: controlCreate,
    formState: { errors: errorsCreate },
  } = useForm<ChildWithContractFormData>({
    resolver: zodResolver(childWithContractSchema),
    defaultValues: {
      first_name: '',
      last_name: '',
      gender: 'male',
      birthdate: '',
      contract_from: '',
      contract_to: '',
      properties: undefined,
    },
  });

  const createBirthdate = watchCreate('birthdate');
  const createContractFrom = watchCreate('contract_from');
  const createContractTo = watchCreate('contract_to');

  // Auto-fill contract end date based on birthdate + org state
  useEffect(() => {
    if (createBirthdate && orgState) {
      const suggestedEnd = calculateContractEndDate(createBirthdate, orgState);
      if (suggestedEnd) {
        setValueCreate('contract_to', suggestedEnd);
      }
    }
  }, [createBirthdate, orgState, setValueCreate]);

  // Funding attributes for the create dialog
  const { fundingAttributes: createFundingAttributes, attributesByKey: createAttributesByKey } =
    useFundingAttributes(orgId, createContractFrom, createContractTo);

  const contractFromDate = watchContract('from');
  const contractToDate = watchContract('to');

  // Get funding attributes from government funding (for the add-contract dialog)
  const { fundingAttributes, attributesByKey } = useFundingAttributes(
    orgId,
    contractFromDate,
    contractToDate
  );

  // Calculate end date preview based on contract from date
  const activeContract = contractChild ? getActiveContract(contractChild.contracts) : null;
  const endDatePreview = contractFromDate ? getDayBefore(contractFromDate) : null;

  const handleCreateChild = useCallback(() => {
    setEditingChild(null);
    resetCreate({
      first_name: '',
      last_name: '',
      gender: 'male',
      birthdate: '',
      contract_from: '',
      contract_to: '',
      properties: undefined,
    });
    setIsChildDialogOpen(true);
  }, [resetCreate]);

  const handleEditChild = useCallback(
    (child: Child) => {
      setEditingChild(child);
      resetChild({
        first_name: child.first_name,
        last_name: child.last_name,
        gender: child.gender,
        birthdate: formatDateForInput(child.birthdate),
      });
      setIsChildDialogOpen(true);
    },
    [resetChild]
  );

  const handleDeleteChild = useCallback((child: Child) => {
    setDeletingChild(child);
    setIsDeleteDialogOpen(true);
  }, []);

  const handleAddContract = useCallback(
    (child: Child) => {
      setContractChild(child);
      setEndCurrentContract(true);

      // Auto-fill end date based on birthdate + org state
      const birthdate = formatDateForInput(child.birthdate);
      const suggestedTo =
        birthdate && orgState ? calculateContractEndDate(birthdate, orgState) || '' : '';

      // Prefill from active contract if exists
      const active = getActiveContract(child.contracts);
      if (active) {
        // Suggest start date as tomorrow
        const tomorrow = new Date();
        tomorrow.setDate(tomorrow.getDate() + 1);
        const tomorrowStr = tomorrow.toISOString().split('T')[0];

        resetContract({
          from: tomorrowStr,
          to: suggestedTo,
          properties: active.properties as Record<string, string> | undefined,
        });
      } else {
        resetContract({ from: '', to: suggestedTo, properties: undefined });
      }
      setIsContractDialogOpen(true);
    },
    [resetContract, orgState]
  );

  const handleViewContractHistory = useCallback(
    (child: Child) => {
      router.push(`/organizations/${orgId}/children/${child.id}/contracts`);
    },
    [router, orgId]
  );

  const onSubmitChild = useCallback(
    (data: ChildFormData) => {
      if (editingChild) {
        updateMutation.mutate({ id: editingChild.id, data });
      }
    },
    [editingChild, updateMutation]
  );

  const onSubmitCreate = useCallback(
    (data: ChildWithContractFormData) => {
      createWithContractMutation.mutate(data);
    },
    [createWithContractMutation]
  );

  // Helper to check if properties have changed
  const propertiesChanged = (
    newProps: ContractProperties | undefined,
    oldProps: ContractProperties | undefined
  ): boolean => {
    const newKeys = Object.keys(newProps || {}).sort();
    const oldKeys = Object.keys(oldProps || {}).sort();
    if (newKeys.length !== oldKeys.length) return true;
    if (newKeys.some((key, i) => key !== oldKeys[i])) return true;
    return newKeys.some((key) => (newProps || {})[key] !== (oldProps || {})[key]);
  };

  const onSubmitContract = useCallback(
    (data: ChildContractFormData) => {
      if (contractChild) {
        // Validate contract start date is not before birthdate
        const childBirthdate = formatDateForInput(contractChild.birthdate);
        if (childBirthdate && data.from && isDateBefore(data.from, childBirthdate)) {
          toast({
            title: t('common.error'),
            description: t('validation.contractBeforeBirthdate'),
            variant: 'destructive',
          });
          return;
        }

        // If there's an active contract and we're ending it, check if something actually changed
        if (activeContract && endCurrentContract) {
          const hasChanges = propertiesChanged(
            data.properties,
            activeContract.properties as ContractProperties | undefined
          );
          if (!hasChanges) {
            toast({
              title: t('contracts.noChangesDetected'),
              description: t('contracts.noChangesDescription'),
              variant: 'destructive',
            });
            return;
          }
        }

        createContractMutation.mutate({
          childId: contractChild.id,
          data: {
            from: formatDateForApi(data.from) || data.from,
            to: formatDateForApi(data.to),
            properties: data.properties,
          },
        });
      }
    },
    [contractChild, activeContract, endCurrentContract, createContractMutation, toast, t]
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t('children.title')}</h1>
        </div>
        <Button onClick={handleCreateChild}>
          <Plus className="mr-2 h-4 w-4" />
          {t('children.newChild')}
        </Button>
      </div>

      <div className="relative max-w-sm">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <label htmlFor="search-children" className="sr-only">
          {t('common.search')}
        </label>
        <Input
          id="search-children"
          placeholder={t('common.search')}
          value={searchInput}
          onChange={(e) => {
            setSearchInput(e.target.value);
            setPage(1);
          }}
          className="pl-9"
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
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('common.name')}</TableHead>
                  <TableHead>{t('gender.label')}</TableHead>
                  <TableHead>{t('children.birthdate')}</TableHead>
                  <TableHead>{t('children.age')}</TableHead>
                  <TableHead>{t('sections.title')}</TableHead>
                  <TableHead>{t('children.currentContract')}</TableHead>
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
                  const status = getContractStatus(currentContract);
                  return (
                    <TableRow key={child.id}>
                      <TableCell className="font-medium">
                        {child.first_name} {child.last_name}
                      </TableCell>
                      <TableCell>{t(`gender.${child.gender}`)}</TableCell>
                      <TableCell>{formatDate(child.birthdate)}</TableCell>
                      <TableCell>{calculateAge(child.birthdate)}</TableCell>
                      <TableCell>
                        {child.section ? (
                          <span>{child.section.name}</span>
                        ) : (
                          <span className="text-muted-foreground">{t('sections.unassigned')}</span>
                        )}
                      </TableCell>
                      <TableCell>
                        {currentContract ? (
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
                        ) : (
                          <span className="text-muted-foreground">{t('children.noContract')}</span>
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
                          onClick={() => handleEditChild(child)}
                          aria-label={t('common.edit')}
                        >
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleDeleteChild(child)}
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
                    <TableCell colSpan={10} className="text-center text-muted-foreground">
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
      {editingChild && (
        <PersonFormDialog
          open={isChildDialogOpen}
          onOpenChange={setIsChildDialogOpen}
          isEditing={true}
          register={registerChild}
          onSubmit={handleSubmitChild(onSubmitChild)}
          errors={errorsChild}
          watch={watchChild}
          setValue={setValueChild}
          isSaving={updateMutation.isPending}
          translationPrefix="children"
        />
      )}

      {/* Child Create Dialog (with initial contract) */}
      <Dialog open={isChildDialogOpen && !editingChild} onOpenChange={setIsChildDialogOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t('children.create')}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmitCreate(onSubmitCreate)} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="create_first_name">{t('children.firstName')}</Label>
                <Input id="create_first_name" {...registerCreate('first_name')} />
                {errorsCreate.first_name && (
                  <p className="text-sm text-destructive">{t('validation.firstNameRequired')}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="create_last_name">{t('children.lastName')}</Label>
                <Input id="create_last_name" {...registerCreate('last_name')} />
                {errorsCreate.last_name && (
                  <p className="text-sm text-destructive">{t('validation.lastNameRequired')}</p>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="create_gender">{t('gender.label')}</Label>
              <Select
                value={watchCreate('gender')}
                onValueChange={(value: Gender) => setValueCreate('gender', value)}
              >
                <SelectTrigger>
                  <SelectValue placeholder={t('gender.selectGender')} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="male">{t('gender.male')}</SelectItem>
                  <SelectItem value="female">{t('gender.female')}</SelectItem>
                  <SelectItem value="diverse">{t('gender.diverse')}</SelectItem>
                </SelectContent>
              </Select>
              {errorsCreate.gender && (
                <p className="text-sm text-destructive">{t('validation.genderRequired')}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="create_birthdate">{t('children.birthdate')}</Label>
              <Input id="create_birthdate" type="date" {...registerCreate('birthdate')} />
              {errorsCreate.birthdate && (
                <p className="text-sm text-destructive">{t('validation.birthdateRequired')}</p>
              )}
            </div>

            <div className="border-t pt-4">
              <h4 className="mb-3 text-sm font-medium">{t('children.initialContract')}</h4>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="create_contract_from">{t('contracts.startDate')}</Label>
                  <Input
                    id="create_contract_from"
                    type="date"
                    {...registerCreate('contract_from')}
                  />
                  {errorsCreate.contract_from && (
                    <p className="text-sm text-destructive">
                      {errorsCreate.contract_from.type === 'custom'
                        ? t('validation.contractBeforeBirthdate')
                        : t('contracts.startDateRequired')}
                    </p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="create_contract_to">{t('contracts.endDateOptional')}</Label>
                  <Input id="create_contract_to" type="date" {...registerCreate('contract_to')} />
                  {createBirthdate && orgState && (
                    <p className="text-xs text-muted-foreground">{t('children.contractEndHint')}</p>
                  )}
                </div>
              </div>

              <div className="mt-4 space-y-2">
                <Label htmlFor="create_properties">{t('contracts.propertiesLabel')}</Label>
                <Controller
                  name="properties"
                  control={controlCreate}
                  render={({ field }) => (
                    <PropertyTagInput
                      id="create_properties"
                      value={field.value}
                      onChange={field.onChange}
                      fundingAttributes={createFundingAttributes}
                      attributesByKey={createAttributesByKey}
                      placeholder={t('contracts.propertiesPlaceholder')}
                      suggestionsLabel={t('contracts.suggestedProperties')}
                    />
                  )}
                />
                <p className="text-xs text-muted-foreground">{t('contracts.propertiesHelp')}</p>
              </div>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setIsChildDialogOpen(false)}>
                {t('common.cancel')}
              </Button>
              <Button type="submit" disabled={createWithContractMutation.isPending}>
                {t('common.save')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Contract Create Dialog */}
      <Dialog open={isContractDialogOpen} onOpenChange={setIsContractDialogOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {t('contracts.newContractFor', {
                name: contractChild ? `${contractChild.first_name} ${contractChild.last_name}` : '',
              })}
            </DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmitContract(onSubmitContract)} className="space-y-4">
            {/* Show active contract info if exists */}
            {activeContract && (
              <Alert>
                <AlertDescription className="space-y-3">
                  <p className="font-medium">{t('contracts.hasActiveContract')}</p>
                  <p className="text-sm text-muted-foreground">
                    {t('contracts.activeSince', {
                      date: formatDate(activeContract.from),
                      attrs:
                        propertiesToValues(activeContract.properties as ContractProperties).join(
                          ', '
                        ) || t('contracts.noAttributes'),
                    })}
                  </p>
                  <div className="flex items-center space-x-2">
                    <Checkbox
                      id="endCurrentContract"
                      checked={endCurrentContract}
                      onCheckedChange={(checked) => setEndCurrentContract(checked === true)}
                    />
                    <label
                      htmlFor="endCurrentContract"
                      className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                    >
                      {endDatePreview
                        ? t('contracts.endCurrentContract', { date: formatDate(endDatePreview) })
                        : t('contracts.endCurrentContract', { date: '...' })}
                    </label>
                  </div>
                </AlertDescription>
              </Alert>
            )}

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="from">{t('contracts.startDate')}</Label>
                <Input id="from" type="date" {...registerContract('from')} />
                {errorsContract.from && (
                  <p className="text-sm text-destructive">{t('contracts.startDateRequired')}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="to">{t('contracts.endDateOptional')}</Label>
                <Input id="to" type="date" {...registerContract('to')} />
                {contractChild && orgState && (
                  <p className="text-xs text-muted-foreground">{t('children.contractEndHint')}</p>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="properties">{t('contracts.propertiesLabel')}</Label>
              <Controller
                name="properties"
                control={controlContract}
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
              <Button type="submit" disabled={createContractMutation.isPending}>
                {t('common.save')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <DeleteConfirmDialog
        open={isDeleteDialogOpen}
        onOpenChange={setIsDeleteDialogOpen}
        onConfirm={() => deletingChild && deleteMutation.mutate(deletingChild.id)}
        isLoading={deleteMutation.isPending}
        resourceName="children"
        description={t('children.confirmDeleteMessage', {
          name: deletingChild ? `${deletingChild.first_name} ${deletingChild.last_name}` : '',
        })}
      />
    </div>
  );
}
