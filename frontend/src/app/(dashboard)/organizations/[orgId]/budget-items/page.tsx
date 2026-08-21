'use client';

import { useMemo } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { Check } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import { Separator } from '@/components/ui/separator';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import type { BudgetItem, BudgetItemCreateRequest, BudgetItemUpdateRequest } from '@/lib/api/types';
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
import { budgetItemWithEntrySchema, type BudgetItemWithEntryFormData } from '@/lib/schemas';
import { FormErrorSummary } from '@/components/forms/form-error-summary';
import { formatDateForApi, eurosToCents } from '@/lib/utils/formatting';
import { useFormatters } from '@/hooks/use-formatters';
import { todayBerlinString } from '@/lib/utils/contracts';

const today = todayBerlinString();

const defaultValues: BudgetItemWithEntryFormData = {
  name: '',
  category: 'expense',
  per_child: false,
  entry_from: today,
  entry_to: undefined,
  entry_amount_euros: 0,
  entry_notes: undefined,
};

export default function BudgetItemsPage() {
  const router = useRouter();
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();

  const fmt = useFormatters();
  const crud = useCrudPage<
    BudgetItem,
    BudgetItemWithEntryFormData,
    BudgetItemCreateRequest,
    BudgetItemUpdateRequest
  >({
    resourceName: 'budgetItems',
    schema: budgetItemWithEntrySchema,
    defaultValues,
    searchable: true,
    itemToFormData: (item) => ({
      name: item.name,
      category: item.category as 'income' | 'expense',
      per_child: item.per_child,
      // Entry fields are not used for editing — provide defaults
      entry_from: today,
      entry_to: undefined,
      entry_amount_euros: 0,
      entry_notes: undefined,
    }),
    listFn: (orgId, params) => apiClient.getBudgetItems(orgId, params),
    createFn: async (orgId, data) => {
      // Create the budget item first
      const budgetItem = await apiClient.createBudgetItem(orgId, {
        name: data.name,
        category: data.category,
        per_child: data.per_child,
      });
      // Create the first entry
      const entryData = data as unknown as BudgetItemWithEntryFormData;
      if (entryData.entry_from) {
        await apiClient.createBudgetItemEntry(orgId, budgetItem.id, {
          from: formatDateForApi(entryData.entry_from) || entryData.entry_from,
          to: formatDateForApi(entryData.entry_to) || undefined,
          amount_cents: eurosToCents(entryData.entry_amount_euros),
          notes: entryData.entry_notes || '',
        });
      }
      return budgetItem;
    },
    // The form schema includes the entry_* fields used by the create flow,
    // but BudgetItemUpdateRequest accepts only {name, category, per_child}.
    // Strip the entry fields here — sending them through verbatim used to
    // be tolerated by the API but is now rejected by the strict JSON
    // binding (security audit G7 / I-M-6).
    updateFn: (orgId, id, data) =>
      apiClient.updateBudgetItem(orgId, id, {
        name: data.name,
        category: data.category,
        per_child: data.per_child,
      }),
    deleteFn: (orgId, id) => apiClient.deleteBudgetItem(orgId, id),
    queryKeys: {
      list: (orgId, page, search) => queryKeys.budgetItems.list(orgId, page, search),
      invalidate: (orgId) => queryKeys.budgetItems.all(orgId),
      // Editing a budget item changes every financials data point that
      // includes it, so invalidate the whole statistics namespace via
      // the prefix key. The previous `[['financials', orgId]]` matched
      // nothing — actual financials keys live under
      // `['statistics', orgId, 'financials', from, to]`. With this,
      // dashboards refetch automatically after a budget-item edit.
      // Spread the readonly key into a mutable tuple so it satisfies
      // the extraInvalidate type signature (which doesn't accept
      // `readonly` tuples).
      extraInvalidate: (orgId) => [[...queryKeys.statistics.all(orgId)]],
    },
  });

  const handleView = (item: BudgetItem) => {
    router.push(`/organizations/${orgId}/budget-items/${item.id}`);
  };

  const columns = useMemo<Column<BudgetItem>[]>(
    () => [
      { key: 'id', header: 'common.id', render: (item) => item.id },
      {
        key: 'name',
        header: 'common.name',
        render: (item) => item.name,
        className: 'font-medium',
      },
      {
        key: 'category',
        header: 'budgetItems.category',
        render: (item) => (
          <Badge variant={item.category === 'income' ? 'default' : 'secondary'}>
            {t(`budgetItems.category${item.category === 'income' ? 'Income' : 'Expense'}`)}
          </Badge>
        ),
      },
      {
        key: 'per_child',
        header: 'budgetItems.perChild',
        render: (item) =>
          item.per_child ? <Check className="text-muted-foreground h-4 w-4" /> : null,
      },
      {
        key: 'active_amount',
        header: 'budgetItems.activeAmount',
        render: (item) =>
          item.active_amount_cents != null ? fmt.currency(item.active_amount_cents) : '—',
      },
    ],
    [t, fmt]
  );

  const isEditing = crud.dialogs.isEditing;

  const onSubmit = (data: BudgetItemWithEntryFormData) => {
    if (!isEditing && data.entry_amount_euros <= 0) {
      crud.form.setError('entry_amount_euros', {
        type: 'manual',
        message: t('budgetItems.amountMustBePositive'),
      });
      return;
    }
    crud.form.onSubmit(data);
  };

  return (
    <div className="space-y-6">
      <CrudPageHeader
        title="budgetItems.title"
        description="budgetItems.description"
        onNew={crud.dialogs.handleCreate}
        newButtonText="budgetItems.newBudgetItem"
      />

      <Card>
        <CardHeader>
          <CardTitle>{t('budgetItems.title')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <SearchInput
            id="search-budget-items"
            value={crud.list.searchInput}
            onChange={crud.list.setSearchInput}
          />
          <QueryError error={crud.list.error} onRetry={crud.list.refetch} />
          <ResourceTable
            items={crud.list.items}
            columns={columns}
            getItemKey={(item) => item.id}
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
        </CardContent>
      </Card>

      <CrudFormDialog
        open={crud.dialogs.isDialogOpen}
        onOpenChange={crud.dialogs.setIsDialogOpen}
        isEditing={isEditing}
        translationPrefix="budgetItems"
        onSubmit={crud.form.handleSubmit(onSubmit)}
        isSaving={crud.mutations.isMutating}
      >
        <FormErrorSummary
          errors={crud.form.errors}
          unmapped={crud.form.unmappedViolations}
          labels={{ name: t('common.name'), description: t('common.description') }}
        />
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">{t('common.name')}</Label>
            <Input id="name" {...crud.form.register('name')} />
            {crud.form.errors.name && (
              <p className="text-destructive text-sm">{t('validation.nameRequired')}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="category">{t('budgetItems.category')}</Label>
            <Select
              value={crud.form.watch('category')}
              onValueChange={(value) =>
                crud.form.setValue('category', value as 'income' | 'expense', {
                  shouldValidate: true,
                })
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="income">{t('budgetItems.categoryIncome')}</SelectItem>
                <SelectItem value="expense">{t('budgetItems.categoryExpense')}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="flex items-center space-x-2">
            <Checkbox
              id="per_child"
              checked={crud.form.watch('per_child')}
              onCheckedChange={(checked) =>
                crud.form.setValue('per_child', checked === true, { shouldValidate: true })
              }
            />
            <Label htmlFor="per_child">{t('budgetItems.perChild')}</Label>
          </div>

          {!isEditing && (
            <>
              <Separator />
              <h4 className="text-sm font-medium">{t('budgetItems.firstEntry')}</h4>

              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="entry_from">{t('budgetItems.fromDate')}</Label>
                  <Input id="entry_from" type="date" {...crud.form.register('entry_from')} />
                  {crud.form.errors.entry_from && (
                    <p className="text-destructive text-sm">{t('validation.fromDateRequired')}</p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="entry_to">{t('budgetItems.toDateOptional')}</Label>
                  <Input id="entry_to" type="date" {...crud.form.register('entry_to')} />
                  {crud.form.errors.entry_to && (
                    <p className="text-destructive text-sm">
                      {t('validation.toDateMustBeAfterFromDate')}
                    </p>
                  )}
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="entry_amount_euros">{t('budgetItems.amountInEuros')}</Label>
                <Input
                  id="entry_amount_euros"
                  type="number"
                  min={0.01}
                  step={0.01}
                  {...crud.form.register('entry_amount_euros', { valueAsNumber: true })}
                />
                {crud.form.errors.entry_amount_euros && (
                  <p className="text-destructive text-sm">
                    {t('budgetItems.amountMustBePositive')}
                  </p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="entry_notes">{t('budgetItems.notes')}</Label>
                <Input id="entry_notes" {...crud.form.register('entry_notes')} />
              </div>
            </>
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
        resourceName="budgetItems"
      />
    </div>
  );
}
