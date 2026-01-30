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
import type { Child, ChildContract, ChildContractCreateRequest, Gender } from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { formatDate, calculateAge, formatDateForInput } from '@/lib/utils/formatting';
import { Pagination } from '@/components/ui/pagination';

const childSchema = z.object({
  first_name: z.string().min(1),
  last_name: z.string().min(1),
  gender: z.enum(['male', 'female', 'diverse']),
  birthdate: z.string().min(1),
});

const contractSchema = z.object({
  from: z.string().min(1),
  to: z.string().optional(),
  attributes: z.string().optional(),
});

type ChildFormData = z.infer<typeof childSchema>;
type ContractFormData = z.infer<typeof contractSchema>;

export default function ChildrenPage() {
  const params = useParams();
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
  const [page, setPage] = useState(1);

  const { data: paginatedData, isLoading } = useQuery({
    queryKey: ['children', orgId, page],
    queryFn: () => apiClient.getChildren(orgId, { page }),
    enabled: !!orgId,
  });

  const children = paginatedData?.data;

  const createMutation = useMutation({
    mutationFn: (data: Omit<ChildFormData, 'organization_id'>) =>
      apiClient.createChild(orgId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['children', orgId] });
      toast({ title: t('children.createSuccess') });
      setIsChildDialogOpen(false);
      resetChild();
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
      queryClient.invalidateQueries({ queryKey: ['children', orgId] });
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
      queryClient.invalidateQueries({ queryKey: ['children', orgId] });
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

  const createContractMutation = useMutation({
    mutationFn: ({ childId, data }: { childId: number; data: ChildContractCreateRequest }) =>
      apiClient.createChildContract(orgId, childId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['children', orgId] });
      toast({ title: t('contracts.createSuccess') });
      setIsContractDialogOpen(false);
      setContractChild(null);
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
    formState: { errors: errorsContract },
  } = useForm<ContractFormData>({
    resolver: zodResolver(contractSchema),
    defaultValues: {
      from: '',
      to: '',
      attributes: '',
    },
  });

  const handleCreateChild = () => {
    setEditingChild(null);
    resetChild({ first_name: '', last_name: '', gender: 'male', birthdate: '' });
    setIsChildDialogOpen(true);
  };

  const handleEditChild = (child: Child) => {
    setEditingChild(child);
    resetChild({
      first_name: child.first_name,
      last_name: child.last_name,
      gender: child.gender,
      birthdate: formatDateForInput(child.birthdate),
    });
    setIsChildDialogOpen(true);
  };

  const handleDeleteChild = (child: Child) => {
    setDeletingChild(child);
    setIsDeleteDialogOpen(true);
  };

  const handleAddContract = (child: Child) => {
    setContractChild(child);
    resetContract({ from: '', to: '', attributes: '' });
    setIsContractDialogOpen(true);
  };

  const onSubmitChild = (data: ChildFormData) => {
    if (editingChild) {
      updateMutation.mutate({ id: editingChild.id, data });
    } else {
      createMutation.mutate(data);
    }
  };

  const onSubmitContract = (data: ContractFormData) => {
    if (contractChild) {
      const attributes = data.attributes
        ? data.attributes
            .split(',')
            .map((a) => a.trim())
            .filter(Boolean)
        : [];
      createContractMutation.mutate({
        childId: contractChild.id,
        data: {
          from: data.from,
          to: data.to || null,
          attributes: attributes.length > 0 ? attributes : undefined,
        },
      });
    }
  };

  const getCurrentContract = (contracts?: ChildContract[]): ChildContract | null => {
    if (!contracts || contracts.length === 0) return null;
    const today = new Date().toISOString().split('T')[0];
    return (
      contracts.find((c) => c.from <= today && (!c.to || c.to >= today)) ||
      contracts.sort((a, b) => b.from.localeCompare(a.from))[0]
    );
  };

  const getContractStatus = (
    contract: ChildContract | null
  ): 'active' | 'upcoming' | 'ended' | null => {
    if (!contract) return null;
    const today = new Date().toISOString().split('T')[0];
    if (contract.from > today) return 'upcoming';
    if (contract.to && contract.to < today) return 'ended';
    return 'active';
  };

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

      <Card>
        <CardHeader>
          <CardTitle>{t('children.title')}</CardTitle>
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
                  <TableHead>{t('children.birthdate')}</TableHead>
                  <TableHead>{t('children.age')}</TableHead>
                  <TableHead>{t('children.currentContract')}</TableHead>
                  <TableHead>{t('children.attributes')}</TableHead>
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
                        {currentContract?.attributes && currentContract.attributes.length > 0 ? (
                          <div className="flex flex-wrap gap-1">
                            {currentContract.attributes.slice(0, 3).map((attr) => (
                              <Badge key={attr} variant="outline" className="text-xs">
                                {attr}
                              </Badge>
                            ))}
                            {currentContract.attributes.length > 3 && (
                              <Badge variant="outline" className="text-xs">
                                +{currentContract.attributes.length - 3}
                              </Badge>
                            )}
                          </div>
                        ) : (
                          <span className="text-sm text-muted-foreground">
                            {t('contracts.noAttributes')}
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleAddContract(child)}
                          title={t('children.addContract')}
                        >
                          <FileText className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="icon" onClick={() => handleEditChild(child)}>
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleDeleteChild(child)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
                {children?.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center text-muted-foreground">
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

      {/* Child Create/Edit Dialog */}
      <Dialog open={isChildDialogOpen} onOpenChange={setIsChildDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingChild ? t('children.edit') : t('children.create')}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmitChild(onSubmitChild)} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="first_name">{t('children.firstName')}</Label>
                <Input id="first_name" {...registerChild('first_name')} />
                {errorsChild.first_name && (
                  <p className="text-sm text-destructive">{t('validation.firstNameRequired')}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="last_name">{t('children.lastName')}</Label>
                <Input id="last_name" {...registerChild('last_name')} />
                {errorsChild.last_name && (
                  <p className="text-sm text-destructive">{t('validation.lastNameRequired')}</p>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="gender">{t('gender.label')}</Label>
              <Select
                value={watchChild('gender')}
                onValueChange={(value: Gender) => setValueChild('gender', value)}
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
              {errorsChild.gender && (
                <p className="text-sm text-destructive">{t('validation.genderRequired')}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="birthdate">{t('children.birthdate')}</Label>
              <Input id="birthdate" type="date" {...registerChild('birthdate')} />
              {errorsChild.birthdate && (
                <p className="text-sm text-destructive">{t('validation.birthdateRequired')}</p>
              )}
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setIsChildDialogOpen(false)}>
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
                name: contractChild ? `${contractChild.first_name} ${contractChild.last_name}` : '',
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
              <Label htmlFor="attributes">{t('contracts.attributesLabel')}</Label>
              <Input
                id="attributes"
                {...registerContract('attributes')}
                placeholder="ganztags, ndh, integration_a"
              />
              <p className="text-xs text-muted-foreground">{t('contracts.attributesHelp')}</p>
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
              {t('children.confirmDeleteMessage', {
                name: deletingChild ? `${deletingChild.first_name} ${deletingChild.last_name}` : '',
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deletingChild && deleteMutation.mutate(deletingChild.id)}
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
