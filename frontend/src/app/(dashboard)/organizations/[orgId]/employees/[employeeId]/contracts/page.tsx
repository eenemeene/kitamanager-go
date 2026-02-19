'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
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
import { DeleteConfirmDialog } from '@/components/crud/delete-confirm-dialog';
import { QueryError } from '@/components/crud/query-error';
import { useResourceMutation } from '@/lib/hooks/use-resource-mutation';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import {
  type EmployeeContract,
  type EmployeeContractCreateRequest,
  type EmployeeContractUpdateRequest,
  LOOKUP_FETCH_LIMIT,
} from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { formatDate, formatDateForInput, formatDateForApi } from '@/lib/utils/formatting';
import { getContractStatus, compareDates } from '@/lib/utils/contracts';
import { EmployeeContractDialog } from '@/components/employees/employee-contract-dialog';
import { employeeContractSchema, type EmployeeContractFormData } from '@/lib/schemas';

export default function EmployeeContractsPage() {
  const params = useParams();
  const router = useRouter();
  const orgId = Number(params.orgId);
  const employeeId = Number(params.employeeId);
  const t = useTranslations();

  const [isContractDialogOpen, setIsContractDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [editingContract, setEditingContract] = useState<EmployeeContract | null>(null);
  const [deletingContract, setDeletingContract] = useState<EmployeeContract | null>(null);

  const {
    data: employee,
    isLoading: employeeLoading,
    error: employeeError,
    refetch: refetchEmployee,
  } = useQuery({
    queryKey: queryKeys.employees.detail(orgId, employeeId),
    queryFn: () => apiClient.getEmployee(orgId, employeeId),
    enabled: !!orgId && !!employeeId,
  });

  const {
    data: contracts,
    isLoading: contractsLoading,
    error: contractsError,
    refetch: refetchContracts,
  } = useQuery({
    queryKey: queryKeys.employees.contracts(orgId, employeeId),
    queryFn: () => apiClient.getEmployeeContracts(orgId, employeeId),
    enabled: !!orgId && !!employeeId,
  });

  const { data: payPlansData } = useQuery({
    queryKey: queryKeys.payPlans.all(orgId),
    queryFn: () => apiClient.getPayPlans(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });
  const payPlans = payPlansData?.data ?? [];

  const { data: sectionsData } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: LOOKUP_FETCH_LIMIT }),
    enabled: !!orgId,
  });
  const sections = sectionsData?.data ?? [];

  const invalidateKeys = [
    queryKeys.employees.contracts(orgId, employeeId),
    queryKeys.employees.detail(orgId, employeeId),
  ];

  const createMutation = useResourceMutation({
    mutationFn: (data: EmployeeContractCreateRequest) =>
      apiClient.createEmployeeContract(orgId, employeeId, data),
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
    mutationFn: ({
      contractId,
      data,
    }: {
      contractId: number;
      data: EmployeeContractUpdateRequest;
    }) => apiClient.updateEmployeeContract(orgId, employeeId, contractId, data),
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
    mutationFn: (contractId: number) =>
      apiClient.deleteEmployeeContract(orgId, employeeId, contractId),
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
    watch,
    setValue,
    formState: { errors },
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

  const handleCreate = () => {
    setEditingContract(null);
    const defaultPayPlanId = payPlans.length === 1 ? payPlans[0].id : 0;
    reset({
      from: '',
      to: '',
      section_id: 0,
      payplan_id: defaultPayPlanId,
      staff_category: 'qualified',
      grade: '',
      step: 1,
      weekly_hours: 39,
    });
    setIsContractDialogOpen(true);
  };

  const handleEdit = (contract: EmployeeContract) => {
    setEditingContract(contract);
    reset({
      from: formatDateForInput(contract.from),
      to: contract.to ? formatDateForInput(contract.to) : '',
      section_id: contract.section_id,
      payplan_id: contract.payplan_id || 0,
      staff_category: contract.staff_category as 'qualified' | 'supplementary' | 'non_pedagogical',
      grade: contract.grade,
      step: contract.step,
      weekly_hours: contract.weekly_hours,
    });
    setIsContractDialogOpen(true);
  };

  const handleDelete = (contract: EmployeeContract) => {
    setDeletingContract(contract);
    setIsDeleteDialogOpen(true);
  };

  const onSubmit = (data: EmployeeContractFormData) => {
    if (editingContract) {
      updateMutation.mutate({
        contractId: editingContract.id,
        data: {
          from: formatDateForApi(data.from) || undefined,
          to: formatDateForApi(data.to) || undefined,
          payplan_id: data.payplan_id,
          staff_category: data.staff_category,
          grade: data.grade,
          step: data.step,
          weekly_hours: data.weekly_hours,
        },
      });
    } else {
      createMutation.mutate({
        from: formatDateForApi(data.from) || data.from,
        to: formatDateForApi(data.to),
        section_id: data.section_id,
        payplan_id: data.payplan_id,
        staff_category: data.staff_category,
        grade: data.grade,
        step: data.step,
        weekly_hours: data.weekly_hours,
      });
    }
  };

  const isLoading = employeeLoading || contractsLoading;
  const queryError = employeeError || contractsError;

  const sortedContracts = contracts
    ? [...contracts].sort((a, b) => compareDates(b.from, a.from))
    : [];

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => router.push(`/organizations/${orgId}/employees`)}
        >
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t('employees.contractHistory')}</h1>
          {employee && (
            <p className="text-muted-foreground">
              {employee.first_name} {employee.last_name}
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

      <QueryError
        error={queryError}
        onRetry={() => {
          refetchEmployee();
          refetchContracts();
        }}
      />

      <Card>
        <CardHeader>
          <CardTitle>{t('contracts.title')}</CardTitle>
          <CardDescription>
            {sortedContracts.length > 0
              ? t('employees.contractHistory')
              : t('employees.noContractsFound')}
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
                  <TableHead>{t('employees.staffCategory.label')}</TableHead>
                  <TableHead>{t('employees.grade')}</TableHead>
                  <TableHead>{t('employees.weeklyHours')}</TableHead>
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
                        {t(`employees.staffCategory.${contract.staff_category}`)}
                      </TableCell>
                      <TableCell>
                        {contract.grade} / {contract.step}
                      </TableCell>
                      <TableCell>{contract.weekly_hours}h</TableCell>
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
                    <TableCell colSpan={8} className="text-muted-foreground text-center">
                      {t('employees.noContractsFound')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <EmployeeContractDialog
        open={isContractDialogOpen}
        onOpenChange={setIsContractDialogOpen}
        title={editingContract ? t('contracts.edit') : t('contracts.create')}
        register={register}
        onSubmit={handleSubmit(onSubmit)}
        errors={errors}
        watch={watch}
        setValue={setValue}
        isSaving={createMutation.isPending || updateMutation.isPending}
        payPlans={payPlans}
        sections={sections}
      />

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
