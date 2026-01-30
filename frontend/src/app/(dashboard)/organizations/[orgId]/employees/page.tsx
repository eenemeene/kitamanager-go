'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Pencil, Trash2, FileText } from 'lucide-react';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useToast } from '@/lib/hooks/use-toast';
import { apiClient, getErrorMessage } from '@/lib/api/client';
import type {
  Employee,
  EmployeeContract,
  EmployeeContractCreateRequest,
  Gender,
} from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { formatDate, calculateAge, formatDateForInput } from '@/lib/utils/formatting';
import { Pagination } from '@/components/ui/pagination';

const employeeSchema = z.object({
  first_name: z.string().min(1),
  last_name: z.string().min(1),
  gender: z.enum(['male', 'female', 'diverse']),
  birthdate: z.string().min(1),
});

const contractSchema = z.object({
  from: z.string().min(1),
  to: z.string().optional(),
  position: z.string().min(1),
  grade: z.string().min(1),
  step: z.number().min(1).max(6),
  weekly_hours: z.number().min(0).max(168),
});

type EmployeeFormData = z.infer<typeof employeeSchema>;
type ContractFormData = z.infer<typeof contractSchema>;

export default function EmployeesPage() {
  const params = useParams();
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
  const [page, setPage] = useState(1);

  const { data: paginatedData, isLoading } = useQuery({
    queryKey: ['employees', orgId, page],
    queryFn: () => apiClient.getEmployees(orgId, { page }),
    enabled: !!orgId,
  });

  const employees = paginatedData?.data;

  const createMutation = useMutation({
    mutationFn: (data: Omit<EmployeeFormData, 'organization_id'>) =>
      apiClient.createEmployee(orgId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['employees', orgId] });
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
      queryClient.invalidateQueries({ queryKey: ['employees', orgId] });
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
      queryClient.invalidateQueries({ queryKey: ['employees', orgId] });
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
    mutationFn: ({
      employeeId,
      data,
    }: {
      employeeId: number;
      data: EmployeeContractCreateRequest;
    }) => apiClient.createEmployeeContract(orgId, employeeId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['employees', orgId] });
      toast({ title: t('contracts.createSuccess') });
      setIsContractDialogOpen(false);
      setContractEmployee(null);
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
    setValue: setValueContract,
    watch: watchContract,
    formState: { errors: errorsContract },
  } = useForm<ContractFormData>({
    resolver: zodResolver(contractSchema),
    defaultValues: {
      from: '',
      to: '',
      position: '',
      grade: '',
      step: 1,
      weekly_hours: 39,
    },
  });

  const handleCreateEmployee = () => {
    setEditingEmployee(null);
    resetEmployee({ first_name: '', last_name: '', gender: 'male', birthdate: '' });
    setIsEmployeeDialogOpen(true);
  };

  const handleEditEmployee = (employee: Employee) => {
    setEditingEmployee(employee);
    resetEmployee({
      first_name: employee.first_name,
      last_name: employee.last_name,
      gender: employee.gender,
      birthdate: formatDateForInput(employee.birthdate),
    });
    setIsEmployeeDialogOpen(true);
  };

  const handleDeleteEmployee = (employee: Employee) => {
    setDeletingEmployee(employee);
    setIsDeleteDialogOpen(true);
  };

  const handleAddContract = (employee: Employee) => {
    setContractEmployee(employee);
    resetContract({ from: '', to: '', position: '', grade: '', step: 1, weekly_hours: 39 });
    setIsContractDialogOpen(true);
  };

  const onSubmitEmployee = (data: EmployeeFormData) => {
    if (editingEmployee) {
      updateMutation.mutate({ id: editingEmployee.id, data });
    } else {
      createMutation.mutate(data);
    }
  };

  const onSubmitContract = (data: ContractFormData) => {
    if (contractEmployee) {
      createContractMutation.mutate({
        employeeId: contractEmployee.id,
        data: {
          ...data,
          to: data.to || null,
        },
      });
    }
  };

  const getCurrentContract = (contracts?: EmployeeContract[]): EmployeeContract | null => {
    if (!contracts || contracts.length === 0) return null;
    const today = new Date().toISOString().split('T')[0];
    return (
      contracts.find((c) => c.from <= today && (!c.to || c.to >= today)) ||
      contracts.sort((a, b) => b.from.localeCompare(a.from))[0]
    );
  };

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

      <Card>
        <CardHeader>
          <CardTitle>{t('employees.title')}</CardTitle>
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
                  <TableHead>{t('common.name')}</TableHead>
                  <TableHead>{t('gender.label')}</TableHead>
                  <TableHead>{t('employees.birthdate')}</TableHead>
                  <TableHead>{t('employees.age')}</TableHead>
                  <TableHead>{t('employees.currentPosition')}</TableHead>
                  <TableHead>{t('employees.grade')}</TableHead>
                  <TableHead>{t('employees.weeklyHours')}</TableHead>
                  <TableHead className="text-right">{t('common.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {employees?.map((employee) => {
                  const currentContract = getCurrentContract(employee.contracts);
                  return (
                    <TableRow key={employee.id}>
                      <TableCell className="font-medium">
                        {employee.first_name} {employee.last_name}
                      </TableCell>
                      <TableCell>{t(`gender.${employee.gender}`)}</TableCell>
                      <TableCell>{formatDate(employee.birthdate)}</TableCell>
                      <TableCell>{calculateAge(employee.birthdate)}</TableCell>
                      <TableCell>
                        {currentContract?.position || (
                          <span className="text-muted-foreground">{t('employees.noContract')}</span>
                        )}
                      </TableCell>
                      <TableCell>
                        {currentContract
                          ? `${currentContract.grade} / ${currentContract.step}`
                          : '-'}
                      </TableCell>
                      <TableCell>{currentContract?.weekly_hours || '-'}</TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleAddContract(employee)}
                          title={t('employees.addContract')}
                        >
                          <FileText className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleEditEmployee(employee)}
                        >
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleDeleteEmployee(employee)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
                {employees?.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={8} className="text-center text-muted-foreground">
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

      {/* Employee Create/Edit Dialog */}
      <Dialog open={isEmployeeDialogOpen} onOpenChange={setIsEmployeeDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editingEmployee ? t('employees.edit') : t('employees.create')}
            </DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmitEmployee(onSubmitEmployee)} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="first_name">{t('employees.firstName')}</Label>
                <Input id="first_name" {...registerEmployee('first_name')} />
                {errorsEmployee.first_name && (
                  <p className="text-sm text-destructive">{t('validation.firstNameRequired')}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="last_name">{t('employees.lastName')}</Label>
                <Input id="last_name" {...registerEmployee('last_name')} />
                {errorsEmployee.last_name && (
                  <p className="text-sm text-destructive">{t('validation.lastNameRequired')}</p>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="gender">{t('gender.label')}</Label>
              <Select
                value={watchEmployee('gender')}
                onValueChange={(value: Gender) => setValueEmployee('gender', value)}
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
              {errorsEmployee.gender && (
                <p className="text-sm text-destructive">{t('validation.genderRequired')}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="birthdate">{t('employees.birthdate')}</Label>
              <Input id="birthdate" type="date" {...registerEmployee('birthdate')} />
              {errorsEmployee.birthdate && (
                <p className="text-sm text-destructive">{t('validation.birthdateRequired')}</p>
              )}
            </div>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setIsEmployeeDialogOpen(false)}
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

      {/* Contract Create Dialog */}
      <Dialog open={isContractDialogOpen} onOpenChange={setIsContractDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {t('contracts.newContractFor', {
                name: contractEmployee
                  ? `${contractEmployee.first_name} ${contractEmployee.last_name}`
                  : '',
              })}
            </DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmitContract(onSubmitContract)} className="space-y-4">
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
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="position">{t('employees.position')}</Label>
              <Input id="position" {...registerContract('position')} />
              {errorsContract.position && (
                <p className="text-sm text-destructive">{t('validation.positionRequired')}</p>
              )}
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="grade">{t('employees.grade')}</Label>
                <Input id="grade" {...registerContract('grade')} placeholder="S8a" />
                {errorsContract.grade && (
                  <p className="text-sm text-destructive">{t('payPlans.gradeRequired')}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="step">{t('employees.step')}</Label>
                <Input
                  id="step"
                  type="number"
                  min={1}
                  max={6}
                  {...registerContract('step', { valueAsNumber: true })}
                />
                {errorsContract.step && (
                  <p className="text-sm text-destructive">{t('payPlans.stepRequired')}</p>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="weekly_hours">{t('employees.weeklyHours')}</Label>
              <Input
                id="weekly_hours"
                type="number"
                min={0}
                max={168}
                step={0.5}
                {...registerContract('weekly_hours', { valueAsNumber: true })}
              />
              {errorsContract.weekly_hours && (
                <p className="text-sm text-destructive">{t('validation.weeklyHoursRequired')}</p>
              )}
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
      <AlertDialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('common.confirmDelete')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('employees.confirmDeleteMessage', {
                name: deletingEmployee
                  ? `${deletingEmployee.first_name} ${deletingEmployee.last_name}`
                  : '',
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deletingEmployee && deleteMutation.mutate(deletingEmployee.id)}
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
