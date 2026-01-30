'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Pencil, Trash2, Eye } from 'lucide-react';
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
  GovernmentFunding,
  GovernmentFundingCreateRequest,
  GovernmentFundingUpdateRequest,
} from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Pagination } from '@/components/ui/pagination';

const fundingSchema = z.object({
  name: z.string().min(1).max(255),
  state: z.string().min(1),
});

type FundingFormData = z.infer<typeof fundingSchema>;

const states = [{ value: 'berlin', label: 'Berlin' }];

export default function GovernmentFundingsPage() {
  const router = useRouter();
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [editingFunding, setEditingFunding] = useState<GovernmentFunding | null>(null);
  const [deletingFunding, setDeletingFunding] = useState<GovernmentFunding | null>(null);
  const [page, setPage] = useState(1);

  const { data: paginatedData, isLoading } = useQuery({
    queryKey: ['government-fundings', page],
    queryFn: () => apiClient.getGovernmentFundings({ page }),
  });

  const fundings = paginatedData?.data;

  const createMutation = useMutation({
    mutationFn: (data: GovernmentFundingCreateRequest) => apiClient.createGovernmentFunding(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['government-fundings'] });
      toast({ title: t('governmentFundings.createSuccess') });
      setIsDialogOpen(false);
      reset();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToCreate', { resource: 'funding' })),
        variant: 'destructive',
      });
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: GovernmentFundingUpdateRequest }) =>
      apiClient.updateGovernmentFunding(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['government-fundings'] });
      toast({ title: t('governmentFundings.updateSuccess') });
      setIsDialogOpen(false);
      setEditingFunding(null);
      reset();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToSave', { resource: 'funding' })),
        variant: 'destructive',
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => apiClient.deleteGovernmentFunding(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['government-fundings'] });
      toast({ title: t('governmentFundings.deleteSuccess') });
      setIsDeleteDialogOpen(false);
      setDeletingFunding(null);
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToDelete', { resource: 'funding' })),
        variant: 'destructive',
      });
    },
  });

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    formState: { errors },
  } = useForm<FundingFormData>({
    resolver: zodResolver(fundingSchema),
    defaultValues: {
      name: '',
      state: 'berlin',
    },
  });

  const handleCreate = () => {
    setEditingFunding(null);
    reset({ name: '', state: 'berlin' });
    setIsDialogOpen(true);
  };

  const handleEdit = (funding: GovernmentFunding) => {
    setEditingFunding(funding);
    reset({ name: funding.name, state: funding.state });
    setIsDialogOpen(true);
  };

  const handleDelete = (funding: GovernmentFunding) => {
    setDeletingFunding(funding);
    setIsDeleteDialogOpen(true);
  };

  const handleView = (funding: GovernmentFunding) => {
    router.push(`/government-fundings/${funding.id}`);
  };

  const onSubmit = (data: FundingFormData) => {
    if (editingFunding) {
      updateMutation.mutate({ id: editingFunding.id, data: { name: data.name } });
    } else {
      createMutation.mutate(data);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t('governmentFundings.title')}</h1>
        </div>
        <Button onClick={handleCreate}>
          <Plus className="mr-2 h-4 w-4" />
          {t('governmentFundings.newGovernmentFunding')}
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('governmentFundings.title')}</CardTitle>
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
                  <TableHead>{t('common.id')}</TableHead>
                  <TableHead>{t('common.name')}</TableHead>
                  <TableHead>{t('states.state')}</TableHead>
                  <TableHead>{t('governmentFundings.periods')}</TableHead>
                  <TableHead className="text-right">{t('common.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {fundings?.map((funding) => (
                  <TableRow key={funding.id}>
                    <TableCell>{funding.id}</TableCell>
                    <TableCell className="font-medium">{funding.name}</TableCell>
                    <TableCell>{t(`states.${funding.state}`)}</TableCell>
                    <TableCell>{funding.total_periods || funding.periods?.length || 0}</TableCell>
                    <TableCell className="text-right">
                      <Button variant="ghost" size="icon" onClick={() => handleView(funding)}>
                        <Eye className="h-4 w-4" />
                      </Button>
                      <Button variant="ghost" size="icon" onClick={() => handleEdit(funding)}>
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button variant="ghost" size="icon" onClick={() => handleDelete(funding)}>
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
                {fundings?.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center text-muted-foreground">
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

      {/* Create/Edit Dialog */}
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editingFunding ? t('governmentFundings.edit') : t('governmentFundings.create')}
            </DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">{t('common.name')}</Label>
              <Input id="name" {...register('name')} />
              {errors.name && (
                <p className="text-sm text-destructive">{t('validation.nameRequired')}</p>
              )}
            </div>

            {!editingFunding && (
              <div className="space-y-2">
                <Label htmlFor="state">{t('states.state')}</Label>
                <Select value={watch('state')} onValueChange={(value) => setValue('state', value)}>
                  <SelectTrigger>
                    <SelectValue placeholder={t('states.selectState')} />
                  </SelectTrigger>
                  <SelectContent>
                    {states.map((state) => (
                      <SelectItem key={state.value} value={state.value}>
                        {t(`states.${state.value}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {errors.state && (
                  <p className="text-sm text-destructive">{t('validation.stateRequired')}</p>
                )}
              </div>
            )}

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setIsDialogOpen(false)}>
                {t('common.cancel')}
              </Button>
              <Button type="submit" disabled={createMutation.isPending || updateMutation.isPending}>
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
            <AlertDialogDescription>{t('governmentFundings.deleteConfirm')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deletingFunding && deleteMutation.mutate(deletingFunding.id)}
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
