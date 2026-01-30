'use client';

import { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
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
  Child,
  ChildContract,
  ChildContractCreateRequest,
  ChildContractUpdateRequest,
  Gender,
} from '@/lib/api/types';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Checkbox } from '@/components/ui/checkbox';
import { TagInput } from '@/components/ui/tag-input';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  formatDate,
  calculateAge,
  formatDateForInput,
  formatDateForApi,
} from '@/lib/utils/formatting';
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
  attributes: z.array(z.string()).default([]),
});

type ChildFormData = z.infer<typeof childSchema>;
type ContractFormData = z.infer<typeof contractSchema>;

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
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['children', orgId] });
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
        const activeContract = getActiveContract(contractChild.contracts);
        if (activeContract && data.from) {
          const endDate = getDayBefore(data.from);
          await apiClient.updateChildContract(orgId, childId, activeContract.id, {
            to: formatDateForApi(endDate),
          });
        }
      }
      return apiClient.createChildContract(orgId, childId, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['children', orgId] });
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
  } = useForm<ContractFormData>({
    resolver: zodResolver(contractSchema),
    defaultValues: {
      from: '',
      to: '',
      attributes: [],
    },
  });

  const contractFromDate = watchContract('from');

  // Helper to get a truly active contract (currently in effect, not ended)
  const getActiveContract = (contracts?: ChildContract[]): ChildContract | null => {
    if (!contracts || contracts.length === 0) return null;
    const today = new Date().toISOString().split('T')[0];
    return contracts.find((c) => c.from <= today && (!c.to || c.to >= today)) || null;
  };

  // Helper to get day before a date string
  const getDayBefore = (dateStr: string): string => {
    const date = new Date(dateStr);
    date.setDate(date.getDate() - 1);
    return date.toISOString().split('T')[0];
  };

  // Calculate end date preview based on contract from date
  const activeContract = contractChild ? getActiveContract(contractChild.contracts) : null;
  const endDatePreview = contractFromDate ? getDayBefore(contractFromDate) : null;

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
    setEndCurrentContract(true);

    // Prefill from active contract if exists
    const active = getActiveContract(child.contracts);
    if (active) {
      // Suggest start date as tomorrow
      const tomorrow = new Date();
      tomorrow.setDate(tomorrow.getDate() + 1);
      const tomorrowStr = tomorrow.toISOString().split('T')[0];

      resetContract({
        from: tomorrowStr,
        to: '',
        attributes: active.attributes || [],
      });
    } else {
      resetContract({ from: '', to: '', attributes: [] });
    }
    setIsContractDialogOpen(true);
  };

  const handleViewContractHistory = (child: Child) => {
    router.push(`/organizations/${orgId}/children/${child.id}/contracts`);
  };

  const onSubmitChild = (data: ChildFormData) => {
    if (editingChild) {
      updateMutation.mutate({ id: editingChild.id, data });
    } else {
      createMutation.mutate(data);
    }
  };

  // Helper to check if attributes have changed
  const attributesChanged = (newAttrs: string[], oldAttrs: string[] | undefined): boolean => {
    const oldSorted = [...(oldAttrs || [])].sort();
    const newSorted = [...newAttrs].sort();
    if (oldSorted.length !== newSorted.length) return true;
    return oldSorted.some((attr, i) => attr !== newSorted[i]);
  };

  const onSubmitContract = (data: ContractFormData) => {
    if (contractChild) {
      const attributes = data.attributes.filter(Boolean);

      // If there's an active contract and we're ending it, check if something actually changed
      if (activeContract && endCurrentContract) {
        const hasChanges = attributesChanged(attributes, activeContract.attributes);
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
                      attrs: activeContract.attributes?.join(', ') || t('contracts.noAttributes'),
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
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="attributes">{t('contracts.attributesLabel')}</Label>
              <Controller
                name="attributes"
                control={controlContract}
                render={({ field }) => (
                  <TagInput
                    id="attributes"
                    value={field.value}
                    onChange={field.onChange}
                    placeholder={t('contracts.attributesPlaceholder')}
                  />
                )}
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
