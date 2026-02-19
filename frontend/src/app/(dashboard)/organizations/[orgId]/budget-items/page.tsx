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
  Column,
} from '@/components/crud';
import { Pagination } from '@/components/ui/pagination';
import { budgetItemWithEntrySchema, type BudgetItemWithEntryFormData } from '@/lib/schemas';
import { formatDateForApi, eurosToCents, formatCurrency } from '@/lib/utils/formatting';

const today = new Date().toISOString().slice(0, 10);

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

  const crud = useCrudPage<
    BudgetItem,
    BudgetItemWithEntryFormData,
    BudgetItemCreateRequest,
    BudgetItemUpdateRequest
  >({
    resourceName: 'budgetItems',
    schema: budgetItemWithEntrySchema,
    defaultValues,
    itemToFormData: (item) => ({
      name: item.name,
      category: item.category,
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
          to: formatDateForApi(entryData.entry_to) || null,
          amount_cents: eurosToCents(entryData.entry_amount_euros),
          notes: entryData.entry_notes || '',
        });
      }
      return budgetItem;
    },
    updateFn: (orgId, id, data) => apiClient.updateBudgetItem(orgId, id, data),
    deleteFn: (orgId, id) => apiClient.deleteBudgetItem(orgId, id),
    queryKeys: {
      list: (orgId, page) => queryKeys.budgetItems.list(orgId, page),
      invalidate: (orgId) => queryKeys.budgetItems.all(orgId),
      extraInvalidate: (orgId) => [['financials', orgId]],
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
          item.active_amount_cents != null ? formatCurrency(item.active_amount_cents) : '—',
      },
    ],
    [t]
  );

  const isEditing = crud.dialogs.isEditing;

  const onSubmit = (data: BudgetItemWithEntryFormData) => {
    if (!isEditing && data.entry_amount_euros <= 0) {
      crud.setError('entry_amount_euros', {
        type: 'manual',
        message: t('budgetItems.amountMustBePositive'),
      });
      return;
    }
    crud.onSubmit(data);
  };

  return (
    <div className="space-y-6">
      <CrudPageHeader
        title="budgetItems.title"
        onNew={crud.dialogs.handleCreate}
        newButtonText="budgetItems.newBudgetItem"
      />

      <Card>
        <CardHeader>
          <CardTitle>{t('budgetItems.title')}</CardTitle>
        </CardHeader>
        <CardContent>
          <ResourceTable
            items={crud.items}
            columns={columns}
            getItemKey={(item) => item.id}
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
        isEditing={isEditing}
        translationPrefix="budgetItems"
        onSubmit={crud.handleSubmit(onSubmit)}
        isSaving={crud.mutations.isMutating}
      >
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">{t('common.name')}</Label>
            <Input id="name" {...crud.register('name')} />
            {crud.errors.name && (
              <p className="text-destructive text-sm">{t('validation.nameRequired')}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="category">{t('budgetItems.category')}</Label>
            <Select
              value={crud.watch('category')}
              onValueChange={(value) =>
                crud.setValue('category', value as 'income' | 'expense', { shouldValidate: true })
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
              checked={crud.watch('per_child')}
              onCheckedChange={(checked) =>
                crud.setValue('per_child', checked === true, { shouldValidate: true })
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
                  <Input id="entry_from" type="date" {...crud.register('entry_from')} />
                  {crud.errors.entry_from && (
                    <p className="text-destructive text-sm">{t('validation.fromDateRequired')}</p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="entry_to">{t('budgetItems.toDateOptional')}</Label>
                  <Input id="entry_to" type="date" {...crud.register('entry_to')} />
                  {crud.errors.entry_to && (
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
                  {...crud.register('entry_amount_euros', { valueAsNumber: true })}
                />
                {crud.errors.entry_amount_euros && (
                  <p className="text-destructive text-sm">
                    {t('budgetItems.amountMustBePositive')}
                  </p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="entry_notes">{t('budgetItems.notes')}</Label>
                <Input id="entry_notes" {...crud.register('entry_notes')} />
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
