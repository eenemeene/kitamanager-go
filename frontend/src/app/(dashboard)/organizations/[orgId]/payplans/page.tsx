'use client';

import { useMemo, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Button } from '@/components/ui/button';
import { apiClient } from '@/lib/api/client';
import type { PayPlan, PayPlanCreateRequest, PayPlanUpdateRequest } from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useCrudMutations } from '@/lib/hooks/use-crud-mutations';
import { useCrudDialogs } from '@/lib/hooks/use-crud-dialogs';
import { CrudPageHeader, ResourceTable, DeleteConfirmDialog, Column } from '@/components/crud';
import { Pagination } from '@/components/ui/pagination';

const payPlanSchema = z.object({
  name: z.string().min(1).max(255),
});

type PayPlanFormData = z.infer<typeof payPlanSchema>;

const defaultValues: PayPlanFormData = {
  name: '',
};

export default function PayPlansPage() {
  const params = useParams();
  const router = useRouter();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const [page, setPage] = useState(1);

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<PayPlanFormData>({
    resolver: zodResolver(payPlanSchema),
    defaultValues,
  });

  const { data: paginatedData, isLoading } = useQuery({
    queryKey: ['payplans', orgId, page],
    queryFn: () => apiClient.getPayPlans(orgId, { page }),
    enabled: !!orgId,
  });

  const payPlans = paginatedData?.data;

  const dialogs = useCrudDialogs<PayPlan, PayPlanFormData>({
    reset,
    itemToFormData: (payPlan) => ({ name: payPlan.name }),
    defaultValues,
  });

  const mutations = useCrudMutations<PayPlan, PayPlanCreateRequest, PayPlanUpdateRequest>({
    resourceName: 'payPlans',
    queryKey: ['payplans', orgId],
    createFn: (data) => apiClient.createPayPlan(orgId, data),
    updateFn: (id, data) => apiClient.updatePayPlan(orgId, id, data),
    deleteFn: (id) => apiClient.deletePayPlan(orgId, id),
    onSuccess: dialogs.closeDialog,
    onDeleteSuccess: dialogs.closeDeleteDialog,
  });

  const onSubmit = (data: PayPlanFormData) => {
    if (dialogs.editingItem) {
      mutations.updateMutation.mutate({ id: dialogs.editingItem.id, data });
    } else {
      mutations.createMutation.mutate(data);
    }
  };

  const handleView = (payPlan: PayPlan) => {
    router.push(`/organizations/${orgId}/payplans/${payPlan.id}`);
  };

  const columns = useMemo<Column<PayPlan>[]>(
    () => [
      { key: 'id', header: 'common.id', render: (payPlan) => payPlan.id },
      {
        key: 'name',
        header: 'common.name',
        render: (payPlan) => payPlan.name,
        className: 'font-medium',
      },
      {
        key: 'periods',
        header: 'governmentFundings.periods',
        render: (payPlan) => payPlan.total_periods || payPlan.periods?.length || 0,
      },
    ],
    []
  );

  return (
    <div className="space-y-6">
      <CrudPageHeader
        title="payPlans.title"
        onNew={dialogs.handleCreate}
        newButtonText="payPlans.newPayPlan"
      />

      <Card>
        <CardHeader>
          <CardTitle>{t('payPlans.title')}</CardTitle>
        </CardHeader>
        <CardContent>
          <ResourceTable
            items={payPlans}
            columns={columns}
            getItemKey={(payPlan) => payPlan.id}
            isLoading={isLoading}
            onView={handleView}
            onEdit={dialogs.handleEdit}
            onDelete={dialogs.handleDelete}
          />
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

      <Dialog open={dialogs.isDialogOpen} onOpenChange={dialogs.setIsDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {dialogs.isEditing ? t('payPlans.edit') : t('payPlans.create')}
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

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => dialogs.setIsDialogOpen(false)}
              >
                {t('common.cancel')}
              </Button>
              <Button type="submit" disabled={mutations.isMutating}>
                {t('common.save')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <DeleteConfirmDialog
        open={dialogs.isDeleteDialogOpen}
        onOpenChange={dialogs.setIsDeleteDialogOpen}
        onConfirm={() =>
          dialogs.deletingItem && mutations.deleteMutation.mutate(dialogs.deletingItem.id)
        }
        isLoading={mutations.deleteMutation.isPending}
        resourceName="payPlans"
      />
    </div>
  );
}
