'use client';

import { useMemo } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import type { PayPlan, PayPlanCreateRequest, PayPlanUpdateRequest } from '@/lib/api/types';
import { useCrudPage } from '@/lib/hooks/use-crud-page';
import {
  CrudPageHeader,
  ResourceTable,
  DeleteConfirmDialog,
  CrudFormDialog,
  Column,
} from '@/components/crud';
import { Pagination } from '@/components/ui/pagination';
import { payPlanSchema, type PayPlanFormData } from '@/lib/schemas';

const defaultValues: PayPlanFormData = {
  name: '',
};

export default function PayPlansPage() {
  const router = useRouter();
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();

  const crud = useCrudPage<PayPlan, PayPlanFormData, PayPlanCreateRequest, PayPlanUpdateRequest>({
    resourceName: 'payPlans',
    schema: payPlanSchema,
    defaultValues,
    itemToFormData: (payPlan) => ({ name: payPlan.name }),
    listFn: (orgId, params) => apiClient.getPayPlans(orgId, params),
    createFn: (orgId, data) => apiClient.createPayPlan(orgId, data),
    updateFn: (orgId, id, data) => apiClient.updatePayPlan(orgId, id, data),
    deleteFn: (orgId, id) => apiClient.deletePayPlan(orgId, id),
    queryKeys: {
      list: (orgId, page) => queryKeys.payPlans.list(orgId, page),
      invalidate: (orgId) => queryKeys.payPlans.all(orgId),
    },
  });

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
        onNew={crud.dialogs.handleCreate}
        newButtonText="payPlans.newPayPlan"
      />

      <Card>
        <CardHeader>
          <CardTitle>{t('payPlans.title')}</CardTitle>
        </CardHeader>
        <CardContent>
          <ResourceTable
            items={crud.items}
            columns={columns}
            getItemKey={(payPlan) => payPlan.id}
            isLoading={crud.isLoading}
            onView={handleView}
            onEdit={crud.dialogs.handleEdit}
            onDelete={crud.dialogs.handleDelete}
          />
          {crud.paginatedData && (
            <Pagination
              page={crud.paginatedData.page}
              totalPages={crud.paginatedData.total_pages}
              total={crud.paginatedData.total}
              limit={crud.paginatedData.limit}
              onPageChange={crud.setPage}
              isLoading={crud.isLoading}
            />
          )}
        </CardContent>
      </Card>

      <CrudFormDialog
        open={crud.dialogs.isDialogOpen}
        onOpenChange={crud.dialogs.setIsDialogOpen}
        isEditing={crud.dialogs.isEditing}
        translationPrefix="payPlans"
        onSubmit={crud.handleSubmit(crud.onSubmit)}
        isSaving={crud.mutations.isMutating}
      >
        <div className="space-y-2">
          <Label htmlFor="name">{t('common.name')}</Label>
          <Input id="name" {...crud.register('name')} />
          {crud.errors.name && (
            <p className="text-destructive text-sm">{t('validation.nameRequired')}</p>
          )}
        </div>
      </CrudFormDialog>

      <DeleteConfirmDialog
        open={crud.dialogs.isDeleteDialogOpen}
        onOpenChange={crud.dialogs.setIsDeleteDialogOpen}
        onConfirm={() =>
          crud.dialogs.deletingItem &&
          crud.mutations.deleteMutation.mutate(crud.dialogs.deletingItem.id)
        }
        isLoading={crud.mutations.deleteMutation.isPending}
        resourceName="payPlans"
      />
    </div>
  );
}
