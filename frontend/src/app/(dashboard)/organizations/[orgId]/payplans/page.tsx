'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
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
import { useToast } from '@/lib/hooks/use-toast';
import { apiClient, getErrorMessage } from '@/lib/api/client';
import type { PayPlan, PayPlanCreateRequest, PayPlanUpdateRequest } from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';

const payPlanSchema = z.object({
  name: z.string().min(1).max(255),
});

type PayPlanFormData = z.infer<typeof payPlanSchema>;

export default function PayPlansPage() {
  const params = useParams();
  const router = useRouter();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [editingPayPlan, setEditingPayPlan] = useState<PayPlan | null>(null);
  const [deletingPayPlan, setDeletingPayPlan] = useState<PayPlan | null>(null);

  const { data: payPlans, isLoading } = useQuery({
    queryKey: ['payplans', orgId],
    queryFn: () => apiClient.getPayPlans(orgId),
    enabled: !!orgId,
  });

  const createMutation = useMutation({
    mutationFn: (data: PayPlanCreateRequest) => apiClient.createPayPlan(orgId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['payplans', orgId] });
      toast({ title: t('payPlans.createSuccess') });
      setIsDialogOpen(false);
      reset();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToCreate', { resource: 'pay plan' })),
        variant: 'destructive',
      });
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: PayPlanUpdateRequest }) =>
      apiClient.updatePayPlan(orgId, id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['payplans', orgId] });
      toast({ title: t('payPlans.updateSuccess') });
      setIsDialogOpen(false);
      setEditingPayPlan(null);
      reset();
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToSave', { resource: 'pay plan' })),
        variant: 'destructive',
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => apiClient.deletePayPlan(orgId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['payplans', orgId] });
      toast({ title: t('payPlans.deleteSuccess') });
      setIsDeleteDialogOpen(false);
      setDeletingPayPlan(null);
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToDelete', { resource: 'pay plan' })),
        variant: 'destructive',
      });
    },
  });

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<PayPlanFormData>({
    resolver: zodResolver(payPlanSchema),
    defaultValues: {
      name: '',
    },
  });

  const handleCreate = () => {
    setEditingPayPlan(null);
    reset({ name: '' });
    setIsDialogOpen(true);
  };

  const handleEdit = (payPlan: PayPlan) => {
    setEditingPayPlan(payPlan);
    reset({ name: payPlan.name });
    setIsDialogOpen(true);
  };

  const handleDelete = (payPlan: PayPlan) => {
    setDeletingPayPlan(payPlan);
    setIsDeleteDialogOpen(true);
  };

  const handleView = (payPlan: PayPlan) => {
    router.push(`/organizations/${orgId}/payplans/${payPlan.id}`);
  };

  const onSubmit = (data: PayPlanFormData) => {
    if (editingPayPlan) {
      updateMutation.mutate({ id: editingPayPlan.id, data });
    } else {
      createMutation.mutate(data);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{t('payPlans.title')}</h1>
        </div>
        <Button onClick={handleCreate}>
          <Plus className="mr-2 h-4 w-4" />
          {t('payPlans.newPayPlan')}
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('payPlans.title')}</CardTitle>
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
                  <TableHead>{t('governmentFundings.periods')}</TableHead>
                  <TableHead className="text-right">{t('common.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {payPlans?.map((payPlan) => (
                  <TableRow key={payPlan.id}>
                    <TableCell>{payPlan.id}</TableCell>
                    <TableCell className="font-medium">{payPlan.name}</TableCell>
                    <TableCell>{payPlan.total_periods || payPlan.periods?.length || 0}</TableCell>
                    <TableCell className="text-right">
                      <Button variant="ghost" size="icon" onClick={() => handleView(payPlan)}>
                        <Eye className="h-4 w-4" />
                      </Button>
                      <Button variant="ghost" size="icon" onClick={() => handleEdit(payPlan)}>
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button variant="ghost" size="icon" onClick={() => handleDelete(payPlan)}>
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
                {payPlans?.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center text-muted-foreground">
                      {t('common.noResults')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Create/Edit Dialog */}
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingPayPlan ? t('payPlans.edit') : t('payPlans.create')}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">{t('common.name')}</Label>
              <Input id="name" {...register('name')} />
              {errors.name && (
                <p className="text-sm text-destructive">{t('validation.nameRequired')}</p>
              )}
            </div>

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
            <AlertDialogDescription>{t('payPlans.deleteConfirm')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deletingPayPlan && deleteMutation.mutate(deletingPayPlan.id)}
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
