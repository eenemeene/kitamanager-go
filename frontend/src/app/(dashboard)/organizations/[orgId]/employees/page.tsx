'use client';

import { useState, useCallback, useMemo, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Plus, Download, Upload } from 'lucide-react';
import { MonthStepper } from '@/components/ui/month-stepper';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
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
import { useCrudDialogs } from '@/lib/hooks/use-crud-dialogs';
import { useContractMutation } from '@/lib/hooks/use-contract-mutation';
import { useImportMutation } from '@/lib/hooks/use-import-mutation';
import { useResourceListFilters } from '@/lib/hooks/use-resource-list-filters';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import {
  type Employee,
  type EmployeeContract,
  type EmployeeContractCreateRequest,
  type EmployeeContractUpdateRequest,
  type PayPlan,
  LOOKUP_FETCH_LIMIT,
} from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { formatDateForInput, formatDateForApi, toLocalDateString } from '@/lib/utils/formatting';
import { getActiveContract } from '@/lib/utils/contracts';
import { Pagination } from '@/components/ui/pagination';
import { DeleteConfirmDialog } from '@/components/crud/delete-confirm-dialog';
import { QueryError } from '@/components/crud/query-error';
import { PersonFormDialog } from '@/components/crud/person-form-dialog';
import { EmployeesTable } from '@/components/employees/employees-table';
import { EmployeeContractDialog } from '@/components/employees/employee-contract-dialog';
import { useToast } from '@/lib/hooks/use-toast';
import {
  employeeSchema,
  employeeContractSchema,
  type EmployeeFormData,
  type EmployeeContractFormData,
} from '@/lib/schemas';

export default function EmployeesPage() {
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
    importFn: (file) => apiClient.importEmployees(orgId, file),
    invalidateQueryKey: queryKeys.employees.all(orgId),
    resourceNameKey: 'employees.title',
    errorMessageKey: 'employees.importError',
  });

  const [isContractDialogOpen, setIsContractDialogOpen] = useState(false);
  const [contractEmployee, setContractEmployee] = useState<Employee | null>(null);
  const [endCurrentContract, setEndCurrentContract] = useState(true);
  const { page, setPage, searchInput, setSearchInput, search, activeOn, setActiveOn } =
    useResourceListFilters();
  const [staffCategoryFilter, setStaffCategoryFilter] = useState<string>('');

  const {
    data: paginatedData,
    isLoading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: queryKeys.employees.list(
      orgId,
      page,
      search,
      staffCategoryFilter,
      toLocalDateString(activeOn)
    ),
    queryFn: () =>
      apiClient.getEmployees(orgId, {
        page,
        search: search || undefined,
        staff_category: staffCategoryFilter || undefined,
        active_on: toLocalDateString(activeOn),
      }),
    enabled: !!orgId,
  });

  const employees = paginatedData?.data;

  const { data: payPlansData, error: payPlansError } = useQuery({
    queryKey: queryKeys.payPlans.all(orgId),
    queryFn: () => apiClient.getPayPlans(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });
  const payPlans = useMemo(() => payPlansData?.data ?? [], [payPlansData?.data]);

  const { data: sectionsData, error: sectionsError } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });
  const sections = useMemo(() => sectionsData?.data ?? [], [sectionsData?.data]);

  // Show toast on secondary query failures
  useEffect(() => {
    const err = payPlansError || sectionsError;
    if (err) {
      toast({
        title: t('common.error'),
        description: t('common.failedToLoad', { resource: t('common.data') }),
        variant: 'destructive',
      });
    }
  }, [payPlansError, sectionsError, toast, t]);

  // Collect unique payplan IDs from employee contracts to fetch details for salary calc
  const payPlanIds = useMemo(
    () =>
      Array.from(
        new Set(
          (employees ?? [])
            .flatMap((e) => e.contracts ?? [])
            .map((c) => c.payplan_id)
            .filter((id) => id > 0)
        )
      ),
    [employees]
  );

  const payPlanDetailsQueries = useQuery({
    queryKey: queryKeys.payPlans.details(orgId, payPlanIds),
    queryFn: async () => {
      const results = await Promise.all(payPlanIds.map((id) => apiClient.getPayPlan(orgId, id)));
      const map = new Map<number, PayPlan>();
      for (const pp of results) {
        map.set(pp.id, pp);
      }
      return map;
    },
    enabled: payPlanIds.length > 0,
  });
  const payPlanMap = payPlanDetailsQueries.data ?? new Map<number, PayPlan>();

  const createContractMutation = useContractMutation<
    EmployeeContractCreateRequest,
    EmployeeContractUpdateRequest,
    EmployeeContract
  >({
    createFn: (employeeId, data) => apiClient.createEmployeeContract(orgId, employeeId, data),
    updateFn: (employeeId, contractId, data) =>
      apiClient.updateEmployeeContract(orgId, employeeId, contractId, data),
    toUpdateData: ({ from, ...rest }) => rest,
    invalidateQueryKeys: [
      queryKeys.employees.all(orgId),
      queryKeys.employees.allUnpaginated(orgId),
      queryKeys.statistics.staffingHours(orgId),
    ],
    extraInvalidateKeys: (employeeId) => [
      queryKeys.employees.contracts(orgId, employeeId),
      queryKeys.employees.detail(orgId, employeeId),
    ],
    onSuccess: () => {
      setIsContractDialogOpen(false);
      setContractEmployee(null);
      setEndCurrentContract(true);
      resetContract();
    },
  });

  const {
    register: registerEmployee,
    handleSubmit: handleSubmitEmployee,
    reset: resetEmployee,
    setValue: setValueEmployee,
    watch: watchEmployee,
    formState: { errors: errorsEmployee },
  } = useForm<EmployeeFormData>({
    resolver: zodResolver(employeeSchema),
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
    setValue: setValueContract,
    formState: { errors: errorsContract },
  } = useForm<EmployeeContractFormData>({
    resolver: zodResolver(employeeContractSchema),
    defaultValues: {
      from: '',
      to: '',
      section_id: 0,
      payplan_id: 0,
      staff_category: 'qualified',
      grade: '',
      step: 1,
      weekly_hours: 39,
    },
  });

  const activeContract = contractEmployee ? getActiveContract(contractEmployee.contracts) : null;

  const dialogs = useCrudDialogs<Employee, EmployeeFormData>({
    reset: resetEmployee,
    itemToFormData: (emp) => ({
      first_name: emp.first_name,
      last_name: emp.last_name,
      gender: emp.gender,
      birthdate: formatDateForInput(emp.birthdate),
    }),
    defaultValues: { first_name: '', last_name: '', gender: 'male', birthdate: '' },
  });

  const mutations = useCrudMutations<
    Employee,
    Omit<EmployeeFormData, 'organization_id'>,
    Partial<EmployeeFormData>
  >({
    resourceName: 'employees',
    queryKey: queryKeys.employees.all(orgId),
    createFn: (data) => apiClient.createEmployee(orgId, data),
    updateFn: (id, data) => apiClient.updateEmployee(orgId, id, data),
    deleteFn: (id) => apiClient.deleteEmployee(orgId, id),
    onSuccess: () => dialogs.closeDialog(),
    onDeleteSuccess: () => dialogs.closeDeleteDialog(),
  });

  const handleAddContract = useCallback(
    (employee: Employee) => {
      setContractEmployee(employee);
      setEndCurrentContract(true);

      const defaultPayPlanId = payPlans.length === 1 ? payPlans[0].id : 0;
      const active = getActiveContract(employee.contracts);
      if (active) {
        const tomorrow = new Date();
        tomorrow.setDate(tomorrow.getDate() + 1);
        const tomorrowStr = toLocalDateString(tomorrow);

        resetContract({
          from: tomorrowStr,
          to: '',
          section_id: active.section_id,
          payplan_id: active.payplan_id || defaultPayPlanId,
          staff_category: active.staff_category as
            | 'qualified'
            | 'supplementary'
            | 'non_pedagogical',
          grade: active.grade,
          step: active.step,
          weekly_hours: active.weekly_hours,
        });
      } else {
        resetContract({
          from: '',
          to: '',
          section_id: 0,
          payplan_id: defaultPayPlanId,
          staff_category: 'qualified',
          grade: '',
          step: 1,
          weekly_hours: 39,
        });
      }
      setIsContractDialogOpen(true);
    },
    [resetContract, payPlans]
  );

  const handleViewContractHistory = useCallback(
    (employee: Employee) => {
      router.push(`/organizations/${orgId}/employees/${employee.id}/contracts`);
    },
    [router, orgId]
  );

  const onSubmitEmployee = useCallback(
    (data: EmployeeFormData) => {
      if (dialogs.editingItem) {
        mutations.updateMutation.mutate({ id: dialogs.editingItem.id, data });
      } else {
        mutations.createMutation.mutate(data);
      }
    },
    [dialogs.editingItem, mutations.updateMutation, mutations.createMutation]
  );

  const onSubmitContract = useCallback(
    (data: EmployeeContractFormData) => {
      if (contractEmployee) {
        createContractMutation.mutate({
          entityId: contractEmployee.id,
          data: {
            from: formatDateForApi(data.from) || data.from,
            to: formatDateForApi(data.to),
            section_id: data.section_id,
            staff_category: data.staff_category,
            grade: data.grade,
            step: data.step,
            weekly_hours: data.weekly_hours,
            payplan_id: data.payplan_id,
          },
          entity: contractEmployee,
          endCurrentContract,
        });
      }
    },
    [contractEmployee, createContractMutation, endCurrentContract]
  );

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t('employees.title')}</h1>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button
            variant="outline"
            onClick={() => {
              window.open(
                apiClient.getEmployeesExportUrl(orgId, {
                  search: search || undefined,
                  staff_category: staffCategoryFilter || undefined,
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
            onClick={() => window.open(apiClient.getEmployeesExportYamlUrl(orgId))}
          >
            <Download className="mr-2 h-4 w-4" />
            {t('employees.exportYaml')}
          </Button>
          <Button variant="outline" onClick={triggerFileInput} disabled={isImporting}>
            <Upload className="mr-2 h-4 w-4" />
            {isImporting ? t('employees.importing') : t('employees.importYaml')}
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
            {t('employees.newEmployee')}
          </Button>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2 md:gap-4">
        <MonthStepper value={activeOn} onChange={setActiveOn} />
        <SearchInput id="search-employees" value={searchInput} onChange={setSearchInput} />
        <Select
          value={staffCategoryFilter}
          onValueChange={(value) => {
            setStaffCategoryFilter(value === 'all' ? '' : value);
            setPage(1);
          }}
        >
          <SelectTrigger className="w-full md:w-[200px]">
            <SelectValue placeholder={t('employees.filterByCategory')} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t('common.all')}</SelectItem>
            <SelectItem value="qualified">{t('employees.staffCategory.qualified')}</SelectItem>
            <SelectItem value="supplementary">
              {t('employees.staffCategory.supplementary')}
            </SelectItem>
            <SelectItem value="non_pedagogical">
              {t('employees.staffCategory.non_pedagogical')}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('employees.title')}</CardTitle>
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
            <EmployeesTable
              employees={employees ?? []}
              payPlanMap={payPlanMap}
              onViewHistory={handleViewContractHistory}
              onAddContract={handleAddContract}
              onEdit={dialogs.handleEdit}
              onDelete={dialogs.handleDelete}
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

      <PersonFormDialog
        open={dialogs.isDialogOpen}
        onOpenChange={dialogs.setIsDialogOpen}
        isEditing={dialogs.isEditing}
        register={registerEmployee}
        onSubmit={handleSubmitEmployee(onSubmitEmployee)}
        errors={errorsEmployee}
        watch={watchEmployee}
        setValue={setValueEmployee}
        isSaving={mutations.isMutating}
        translationPrefix="employees"
      />

      <EmployeeContractDialog
        open={isContractDialogOpen}
        onOpenChange={setIsContractDialogOpen}
        title={t('contracts.newContractFor', {
          name: contractEmployee
            ? `${contractEmployee.first_name} ${contractEmployee.last_name}`
            : '',
        })}
        register={registerContract}
        onSubmit={handleSubmitContract(onSubmitContract)}
        errors={errorsContract}
        watch={watchContract}
        setValue={setValueContract}
        isSaving={createContractMutation.isPending}
        payPlans={payPlans}
        sections={sections}
        activeContractInfo={
          activeContract
            ? {
                contract: activeContract,
                endCurrentContract,
                onEndCurrentContractChange: setEndCurrentContract,
              }
            : undefined
        }
      />

      <DeleteConfirmDialog
        open={dialogs.isDeleteDialogOpen}
        onOpenChange={dialogs.setIsDeleteDialogOpen}
        onConfirm={() =>
          dialogs.deletingItem && mutations.deleteMutation.mutate(dialogs.deletingItem.id)
        }
        isLoading={mutations.deleteMutation.isPending}
        resourceName="employees"
        description={t('employees.confirmDeleteMessage', {
          name: dialogs.deletingItem
            ? `${dialogs.deletingItem.first_name} ${dialogs.deletingItem.last_name}`
            : '',
        })}
      />
    </div>
  );
}
