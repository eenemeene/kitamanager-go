'use client';

import { useMemo, useRef } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Table2, Upload } from 'lucide-react';
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
  EmptyState,
  Column,
} from '@/components/crud';
import { Pagination } from '@/components/ui/pagination';
import { SearchInput } from '@/components/ui/search-input';
import { payPlanSchema, type PayPlanFormData } from '@/lib/schemas';
import { useToast } from '@/lib/hooks/use-toast';
import { FormErrorSummary } from '@/components/forms/form-error-summary';

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
        render: (payPlan) => payPlan.periods_count,
      },
    ],
    []
  );

  // The onboarding panel, not the "no results" row: it appears only when the
  // resource is genuinely unused, never when a search just failed to match.
  // Both of these feed the Financials and Forecast figures, so an operator who
  // never creates any gets plausible-looking numbers that quietly omit a whole
  // category of cost — which is what the panel exists to say.
  const showEmptyState =
    !crud.list.isLoading && !crud.list.searchInput && crud.list.paginatedData?.total === 0;

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
            aria-label={t('payPlans.importYaml')}
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
            value={crud.list.searchInput}
            onChange={crud.list.setSearchInput}
          />
          <QueryError error={crud.list.error} onRetry={crud.list.refetch} />
          {showEmptyState ? (
            <EmptyState
              icon={Table2}
              title="payPlans.emptyTitle"
              description="payPlans.emptyDescription"
              action={
                <Button onClick={crud.dialogs.handleCreate}>
                  <Plus className="mr-2 h-4 w-4" />
                  {t('payPlans.emptyAction')}
                </Button>
              }
            />
          ) : (
            <>
              <ResourceTable
                items={crud.list.items}
                columns={columns}
                getItemKey={(payPlan) => payPlan.id}
                isLoading={crud.list.isLoading}
                onView={handleView}
                onEdit={crud.dialogs.handleEdit}
                onDelete={crud.dialogs.handleDelete}
              />
              {crud.list.paginatedData && (
                <Pagination
                  page={crud.list.paginatedData.page}
                  totalPages={crud.list.paginatedData.total_pages}
                  total={crud.list.paginatedData.total}
                  limit={crud.list.paginatedData.limit}
                  onPageChange={crud.list.setPage}
                  isLoading={crud.list.isLoading}
                />
              )}
            </>
          )}
        </CardContent>
      </Card>

      <CrudFormDialog
        open={crud.dialogs.isDialogOpen}
        onOpenChange={crud.dialogs.setIsDialogOpen}
        isEditing={crud.dialogs.isEditing}
        translationPrefix="payPlans"
        onSubmit={crud.form.handleSubmit(crud.form.onSubmit)}
        isSaving={crud.mutations.isMutating}
      >
        <FormErrorSummary
          errors={crud.form.errors}
          unmapped={crud.form.unmappedViolations}
          labels={{ name: t('common.name'), description: t('common.description') }}
        />
        <div className="space-y-2">
          <Label htmlFor="name">{t('common.name')}</Label>
          <Input id="name" {...crud.form.register('name')} />
          {crud.form.errors.name && (
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
