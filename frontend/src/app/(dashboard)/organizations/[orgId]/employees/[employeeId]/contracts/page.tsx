'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { DeleteConfirmDialog } from '@/components/crud/delete-confirm-dialog';
import { QueryError } from '@/components/crud/query-error';
import { ContractTimeline } from '@/components/contracts/contract-timeline';
import { useResourceMutation } from '@/lib/hooks/use-resource-mutation';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import {
  type EmployeeContract,
  type EmployeeContractCreateRequest,
  type EmployeeContractCorrectRequest,
  type ContractBoundaryMoveRequest,
  LOOKUP_FETCH_LIMIT,
} from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { formatDateForInput, formatDateForApi } from '@/lib/utils/formatting';
import { getContractStatus, compareDates, getDayBefore } from '@/lib/utils/contracts';
import { EmployeeContractDialog } from '@/components/employees/employee-contract-dialog';
import { employeeContractSchema, type EmployeeContractFormData } from '@/lib/schemas';
import { useToast } from '@/lib/hooks/use-toast';
import { showErrorToast } from '@/lib/utils/show-error-toast';
import { validationTiming } from '@/lib/forms/validation-timing';
import { useProblemFormErrors, suppressesToast } from '@/lib/forms/use-problem-form-errors';
import { useResetOnReopen } from '@/lib/forms/use-reset-on-reopen';
import { useFormatters } from '@/hooks/use-formatters';

export default function EmployeeContractsPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const employeeId = Number(params.employeeId);
  const t = useTranslations();
  const fmt = useFormatters();
  const queryClient = useQueryClient();
  const { toast } = useToast();

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
    queryKeys.employees.all(orgId),
    queryKeys.employees.allUnpaginated(orgId),
    queryKeys.statistics.staffingHours(orgId),
  ];

  const createMutation = useResourceMutation({
    onMutationError: suppressesToast,
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

  const correctMutation = useResourceMutation({
    onMutationError: suppressesToast,
    // The whole contract, not just its id: correcting it needs the version as an
    // If-Match precondition, so an edit cannot silently overwrite someone else's.
    mutationFn: ({
      contract,
      data,
    }: {
      contract: EmployeeContract;
      data: EmployeeContractCorrectRequest;
    }) => apiClient.correctEmployeeContract(orgId, employeeId, contract.id, contract.version, data),
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
    mutationFn: (contract: EmployeeContract) =>
      apiClient.deleteEmployeeContract(orgId, employeeId, contract.id, contract.version),
    invalidateQueryKey: invalidateKeys,
    successMessage: t('contracts.deleteSuccess'),
    errorMessage: t('common.failedToDelete', { resource: 'contract' }),
    onSuccess: () => {
      setIsDeleteDialogOpen(false);
      setDeletingContract(null);
    },
  });

  const contractsQueryKey = queryKeys.employees.contracts(orgId, employeeId);

  const boundaryMutation = useMutation({
    mutationFn: (move: ContractBoundaryMoveRequest) =>
      apiClient.moveEmployeeContractBoundary(orgId, employeeId, move),
    onMutate: async (move) => {
      await queryClient.cancelQueries({ queryKey: contractsQueryKey });
      const previous = queryClient.getQueryData<EmployeeContract[]>(contractsQueryKey);
      // Mirror what the server will do: the later contract starts at the seam,
      // the earlier one ends the day before. Only these two dates change, which
      // is the point of sending one date instead of four.
      //
      // Calendar arithmetic, not `parseISO` + local `format`: `move.at` is a
      // UTC-midnight timestamp, and rendering it in a zone behind UTC lands on
      // the previous day before the subtraction has even happened, so the seam
      // jumped two days on the way past.
      const dayBefore = formatDateForApi(getDayBefore(move.at)) ?? undefined;
      queryClient.setQueryData<EmployeeContract[]>(contractsQueryKey, (old) =>
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
    watch,
    setValue,
    setError,
    clearErrors,
    getValues,
    formState: { errors },
  } = useForm<EmployeeContractFormData>({
    ...validationTiming,
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

  // Create and correct both submit this one dialog.
  // A dialog that just opened has nothing pending -- react-query keeps a
  // mutation's rejection until the next attempt, so without this the form
  // reopened showing the previous submit's summary.
  useResetOnReopen(isContractDialogOpen, createMutation, correctMutation);

  const unmappedViolations = useProblemFormErrors([createMutation.error, correctMutation.error], {
    setError,
    clearErrors,
    getValues,
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
      correctMutation.mutate({
        contract: editingContract,
        // section_id is included because the dialog renders a section select whose
        // value the old payload dropped. `to` is sent as null rather than omitted
        // when empty: omitting now means "leave alone".
        data: {
          from: formatDateForApi(data.from) || undefined,
          to: formatDateForApi(data.to),
          section_id: data.section_id,
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
        to: formatDateForApi(data.to) ?? undefined,
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
      <div className="flex flex-col gap-4 md:flex-row md:items-center">
        <div className="min-w-0 flex-1">
          <Breadcrumb
            items={[
              { label: t('nav.employees'), href: `/organizations/${orgId}/employees` },
              {
                label: employee ? `${employee.first_name} ${employee.last_name}` : '...',
              },
              { label: t('employees.contractHistory') },
            ]}
          />
          <h1 className="mt-1 text-3xl font-bold tracking-tight">
            {t('employees.contractHistory')}
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
          refetchEmployee();
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
                    ? t('employees.contractHistory')
                    : t('employees.noContractsFound')}
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
                            <TableCell>{fmt.date(contract.from)}</TableCell>
                            <TableCell>
                              {contract.to ? fmt.date(contract.to) : t('common.ongoing')}
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
                        <div className="flex flex-wrap gap-1">
                          <Badge variant="outline" className="text-xs">
                            {t(`employees.staffCategory.${contract.staff_category}`)}
                          </Badge>
                          <Badge variant="outline" className="text-xs">
                            {contract.grade} / {contract.step}
                          </Badge>
                          <Badge variant="outline" className="text-xs">
                            {contract.weekly_hours}h
                          </Badge>
                        </div>
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

      <EmployeeContractDialog
        open={isContractDialogOpen}
        onOpenChange={setIsContractDialogOpen}
        title={editingContract ? t('contracts.edit') : t('contracts.create')}
        register={register}
        onSubmit={handleSubmit(onSubmit)}
        errors={errors}
        watch={watch}
        setValue={setValue}
        unmapped={unmappedViolations}
        isSaving={createMutation.isPending || correctMutation.isPending}
        payPlans={payPlans}
        sections={sections}
      />

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
