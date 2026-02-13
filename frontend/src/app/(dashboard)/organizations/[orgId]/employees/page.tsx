'use client';

import { useState, useCallback, useMemo } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Search } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Input } from '@/components/ui/input';
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
  Employee,
  EmployeeContract,
  EmployeeContractCreateRequest,
  PayPlan,
} from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { formatDateForInput, formatDateForApi } from '@/lib/utils/formatting';
import { getActiveContract, getDayBefore } from '@/lib/utils/contracts';
import { Pagination } from '@/components/ui/pagination';
import { useDebouncedValue } from '@/lib/hooks/use-debounced-value';
import { DeleteConfirmDialog } from '@/components/crud/delete-confirm-dialog';
import { QueryError } from '@/components/crud/query-error';
import { PersonFormDialog } from '@/components/crud/person-form-dialog';
import { EmployeesTable } from '@/components/employees/employees-table';
import { EmployeeContractDialog } from '@/components/employees/employee-contract-dialog';
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
  const queryClient = useQueryClient();

  const [isEmployeeDialogOpen, setIsEmployeeDialogOpen] = useState(false);
  const [isContractDialogOpen, setIsContractDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [editingEmployee, setEditingEmployee] = useState<Employee | null>(null);
  const [deletingEmployee, setDeletingEmployee] = useState<Employee | null>(null);
  const [contractEmployee, setContractEmployee] = useState<Employee | null>(null);
  const [endCurrentContract, setEndCurrentContract] = useState(true);
  const [page, setPage] = useState(1);
  const [searchInput, setSearchInput] = useState('');
  const search = useDebouncedValue(searchInput, 300);
  const [staffCategoryFilter, setStaffCategoryFilter] = useState<string>('');

  const {
    data: paginatedData,
    isLoading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: queryKeys.employees.list(orgId, page, search, staffCategoryFilter),
    queryFn: () =>
      apiClient.getEmployees(orgId, {
        page,
        search: search || undefined,
        staff_category: staffCategoryFilter || undefined,
      }),
    enabled: !!orgId,
  });

  const employees = paginatedData?.data;

  const { data: payPlansData } = useQuery({
    queryKey: queryKeys.payPlans.all(orgId),
    queryFn: () => apiClient.getPayPlans(orgId, { limit: 100 }),
    enabled: !!orgId,
  });
  const payPlans = useMemo(() => payPlansData?.data ?? [], [payPlansData?.data]);

  // Collect unique payplan IDs from employee contracts to fetch details for salary calc
  const payPlanIds = Array.from(
    new Set(
      (employees ?? [])
        .flatMap((e) => e.contracts ?? [])
        .map((c) => c.payplan_id)
        .filter((id) => id > 0)
    )
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

  const createMutation = useMutation({
    mutationFn: (data: Omit<EmployeeFormData, 'organization_id'>) =>
      apiClient.createEmployee(orgId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.employees.all(orgId) });
      toast({ title: t('employees.createSuccess') });
      setIsEmployeeDialogOpen(false);
      resetEmployee();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToCreate', { resource: 'employee' })),
        variant: 'destructive',
      });
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<EmployeeFormData> }) =>
      apiClient.updateEmployee(orgId, id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.employees.all(orgId) });
      toast({ title: t('employees.updateSuccess') });
      setIsEmployeeDialogOpen(false);
      setEditingEmployee(null);
      resetEmployee();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToSave', { resource: 'employee' })),
        variant: 'destructive',
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => apiClient.deleteEmployee(orgId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.employees.all(orgId) });
      toast({ title: t('employees.deleteSuccess') });
      setIsDeleteDialogOpen(false);
      setDeletingEmployee(null);
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToDelete', { resource: 'employee' })),
        variant: 'destructive',
      });
    },
  });

  const createContractMutation = useMutation({
    mutationFn: async ({
      employeeId,
      data,
    }: {
      employeeId: number;
      data: EmployeeContractCreateRequest;
    }) => {
      if (contractEmployee && endCurrentContract) {
        const active = getActiveContract(contractEmployee.contracts);
        if (active && data.from) {
          const endDate = getDayBefore(data.from);
          await apiClient.updateEmployeeContract(orgId, employeeId, active.id, {
            to: formatDateForApi(endDate),
          });
        }
      }
      return apiClient.createEmployeeContract(orgId, employeeId, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.employees.all(orgId) });
      toast({
        title: endCurrentContract
          ? t('contracts.previousContractEnded')
          : t('contracts.createSuccess'),
      });
      setIsContractDialogOpen(false);
      setContractEmployee(null);
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
      payplan_id: 0,
      staff_category: 'qualified',
      grade: '',
      step: 1,
      weekly_hours: 39,
    },
  });

  const contractFromDate = watchContract('from');
  const activeContract = contractEmployee ? getActiveContract(contractEmployee.contracts) : null;
  const endDatePreview = contractFromDate ? getDayBefore(contractFromDate) : null;

  const handleCreateEmployee = useCallback(() => {
    setEditingEmployee(null);
    resetEmployee({ first_name: '', last_name: '', gender: 'male', birthdate: '' });
    setIsEmployeeDialogOpen(true);
  }, [resetEmployee]);

  const handleEditEmployee = useCallback(
    (employee: Employee) => {
      setEditingEmployee(employee);
      resetEmployee({
        first_name: employee.first_name,
        last_name: employee.last_name,
        gender: employee.gender,
        birthdate: formatDateForInput(employee.birthdate),
      });
      setIsEmployeeDialogOpen(true);
    },
    [resetEmployee]
  );

  const handleDeleteEmployee = useCallback((employee: Employee) => {
    setDeletingEmployee(employee);
    setIsDeleteDialogOpen(true);
  }, []);

  const handleAddContract = useCallback(
    (employee: Employee) => {
      setContractEmployee(employee);
      setEndCurrentContract(true);

      const defaultPayPlanId = payPlans.length === 1 ? payPlans[0].id : 0;
      const active = getActiveContract(employee.contracts);
      if (active) {
        const tomorrow = new Date();
        tomorrow.setDate(tomorrow.getDate() + 1);
        const tomorrowStr = tomorrow.toISOString().split('T')[0];

        resetContract({
          from: tomorrowStr,
          to: '',
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
      if (editingEmployee) {
        updateMutation.mutate({ id: editingEmployee.id, data });
      } else {
        createMutation.mutate(data);
      }
    },
    [editingEmployee, updateMutation, createMutation]
  );

  const contractDetailsChanged = (
    newData: {
      staff_category: string;
      grade: string;
      step: number;
      weekly_hours: number;
      payplan_id: number;
    },
    oldContract: EmployeeContract
  ): boolean => {
    return (
      newData.staff_category !== oldContract.staff_category ||
      newData.grade !== oldContract.grade ||
      newData.step !== oldContract.step ||
      newData.weekly_hours !== oldContract.weekly_hours ||
      newData.payplan_id !== oldContract.payplan_id
    );
  };

  const onSubmitContract = useCallback(
    (data: EmployeeContractFormData) => {
      if (contractEmployee) {
        if (activeContract && endCurrentContract) {
          const hasChanges = contractDetailsChanged(
            {
              staff_category: data.staff_category,
              grade: data.grade,
              step: data.step,
              weekly_hours: data.weekly_hours,
              payplan_id: data.payplan_id,
            },
            activeContract
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
          employeeId: contractEmployee.id,
          data: {
            from: formatDateForApi(data.from) || data.from,
            to: formatDateForApi(data.to),
            staff_category: data.staff_category,
            grade: data.grade,
            step: data.step,
            weekly_hours: data.weekly_hours,
            payplan_id: data.payplan_id,
          },
        });
      }
    },
    [contractEmployee, activeContract, endCurrentContract, createContractMutation, toast, t]
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t('employees.title')}</h1>
        </div>
        <Button onClick={handleCreateEmployee}>
          <Plus className="mr-2 h-4 w-4" />
          {t('employees.newEmployee')}
        </Button>
      </div>

      <div className="flex items-center gap-4">
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <label htmlFor="search-employees" className="sr-only">
            {t('common.search')}
          </label>
          <Input
            id="search-employees"
            placeholder={t('common.search')}
            value={searchInput}
            onChange={(e) => {
              setSearchInput(e.target.value);
              setPage(1);
            }}
            className="pl-9"
          />
        </div>
        <Select
          value={staffCategoryFilter}
          onValueChange={(value) => {
            setStaffCategoryFilter(value === 'all' ? '' : value);
            setPage(1);
          }}
        >
          <SelectTrigger className="w-[200px]">
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
              onEdit={handleEditEmployee}
              onDelete={handleDeleteEmployee}
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
        open={isEmployeeDialogOpen}
        onOpenChange={setIsEmployeeDialogOpen}
        isEditing={!!editingEmployee}
        register={registerEmployee}
        onSubmit={handleSubmitEmployee(onSubmitEmployee)}
        errors={errorsEmployee}
        watch={watchEmployee}
        setValue={setValueEmployee}
        isSaving={createMutation.isPending || updateMutation.isPending}
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
        activeContractInfo={
          activeContract
            ? {
                contract: activeContract,
                endCurrentContract,
                onEndCurrentContractChange: setEndCurrentContract,
                endDatePreview,
              }
            : undefined
        }
      />

      <DeleteConfirmDialog
        open={isDeleteDialogOpen}
        onOpenChange={setIsDeleteDialogOpen}
        onConfirm={() => deletingEmployee && deleteMutation.mutate(deletingEmployee.id)}
        isLoading={deleteMutation.isPending}
        resourceName="employees"
        description={t('employees.confirmDeleteMessage', {
          name: deletingEmployee
            ? `${deletingEmployee.first_name} ${deletingEmployee.last_name}`
            : '',
        })}
      />
    </div>
  );
}
