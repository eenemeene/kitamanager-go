'use client';

import { useMemo, useRef } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Upload } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { apiClient, getErrorMessage } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import type { PayPlan, PayPlanCreateRequest, PayPlanUpdateRequest } from '@/lib/api/types';
import { useCrudPage } from '@/lib/hooks/use-crud-page';
import {
  CrudPageHeader,
  ResourceTable,
  DeleteConfirmDialog,
  CrudFormDialog,
  QueryError,
  Column,
} from '@/components/crud';
import { Pagination } from '@/components/ui/pagination';
import { SearchInput } from '@/components/ui/search-input';
import { payPlanSchema, type PayPlanFormData } from '@/lib/schemas';
import { useToast } from '@/lib/hooks/use-toast';

const defaultValues: PayPlanFormData = {
  name: '',
};

export default function PayPlansPage() {
  const router = useRouter();
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const importMutation = useMutation({
    mutationFn: (file: File) => apiClient.importPayPlan(orgId, file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.payPlans.all(orgId) });
      if (fileInputRef.current) fileInputRef.current.value = '';
      toast({ title: t('common.success') });
    },
    onError: (error) => {
      toast({
        title: t('payPlans.importError'),
        description: getErrorMessage(error, t('payPlans.importError')),
        variant: 'destructive',
      });
    },
  });

  const handleImportFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) importMutation.mutate(file);
  };

  const crud = useCrudPage<PayPlan, PayPlanFormData, PayPlanCreateRequest, PayPlanUpdateRequest>({
    resourceName: 'payPlans',
    schema: payPlanSchema,
    defaultValues,
    searchable: true,
    itemToFormData: (payPlan) => ({ name: payPlan.name }),
    listFn: (orgId, params) => apiClient.getPayPlans(orgId, params),
    createFn: (orgId, data) => apiClient.createPayPlan(orgId, data),
    updateFn: (orgId, id, data) => apiClient.updatePayPlan(orgId, id, data),
    deleteFn: (orgId, id) => apiClient.deletePayPlan(orgId, id),
    queryKeys: {
      list: (orgId, page, search) => queryKeys.payPlans.list(orgId, page, search),
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
        // TODO(api-types-migration): the PayPlan list DTO doesn't expose
        // total_periods/periods. Fetch the detail response or extend the
        // backend list response to include the count.
        render: (payPlan) =>
          (payPlan as PayPlan & { total_periods?: number; periods?: unknown[] }).total_periods ||
          (payPlan as PayPlan & { total_periods?: number; periods?: unknown[] }).periods?.length ||
          0,
      },
    ],
    []
  );

  return (
    <div className="space-y-6">
      <CrudPageHeader
        title="payPlans.title"
        description="payPlans.description"
        onNew={crud.dialogs.handleCreate}
        newButtonText="payPlans.newPayPlan"
      >
        <>
          <input
            ref={fileInputRef}
            type="file"
            accept=".yaml,.yml"
            className="hidden"
            onChange={handleImportFile}
          />
          <Button
            variant="outline"
            onClick={() => fileInputRef.current?.click()}
            disabled={importMutation.isPending}
          >
            <Upload className="mr-2 h-4 w-4" />
            {importMutation.isPending ? t('payPlans.importing') : t('payPlans.importYaml')}
          </Button>
        </>
      </CrudPageHeader>

      <Card>
        <CardHeader>
          <CardTitle>{t('payPlans.title')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <SearchInput
            id="search-payplans"
            value={crud.searchInput}
            onChange={crud.setSearchInput}
          />
          <QueryError error={crud.error} onRetry={crud.refetch} />
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
