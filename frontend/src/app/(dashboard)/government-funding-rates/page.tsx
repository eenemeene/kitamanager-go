'use client';

import { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import {
  type GovernmentFunding,
  type GovernmentFundingCreateRequest,
  type GovernmentFundingUpdateRequest,
  VALID_STATES,
} from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Pagination } from '@/components/ui/pagination';
import {
  CrudPageHeader,
  ResourceTable,
  DeleteConfirmDialog,
  CrudFormDialog,
  Column,
} from '@/components/crud';
import { useCrudDialogs } from '@/lib/hooks/use-crud-dialogs';
import { useCrudMutations } from '@/lib/hooks/use-crud-mutations';
import { governmentFundingSchema, type GovernmentFundingFormData } from '@/lib/schemas';

const defaultValues: GovernmentFundingFormData = {
  name: '',
  state: VALID_STATES[0],
};

export default function GovernmentFundingsPage() {
  const router = useRouter();
  const t = useTranslations();
  const [page, setPage] = useState(1);

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    formState: { errors },
  } = useForm<GovernmentFundingFormData>({
    resolver: zodResolver(governmentFundingSchema),
    defaultValues,
  });

  const dialogs = useCrudDialogs<GovernmentFunding, GovernmentFundingFormData>({
    reset,
    itemToFormData: (funding) => ({ name: funding.name, state: funding.state }),
    defaultValues,
  });

  const mutations = useCrudMutations<
    GovernmentFunding,
    GovernmentFundingCreateRequest,
    GovernmentFundingUpdateRequest
  >({
    resourceName: 'governmentFundings',
    queryKey: queryKeys.governmentFundings.all(),
    createFn: (data) => apiClient.createGovernmentFunding(data),
    updateFn: (id, data) => apiClient.updateGovernmentFunding(id, data),
    deleteFn: (id) => apiClient.deleteGovernmentFunding(id),
    onSuccess: () => dialogs.closeDialog(),
    onDeleteSuccess: () => dialogs.closeDeleteDialog(),
  });

  const { data: paginatedData, isLoading } = useQuery({
    queryKey: queryKeys.governmentFundings.list(page),
    queryFn: () => apiClient.getGovernmentFundings({ page }),
  });

  const handleView = (funding: GovernmentFunding) => {
    router.push(`/government-funding-rates/${funding.id}`);
  };

  const onSubmit = (data: GovernmentFundingFormData) => {
    if (dialogs.editingItem) {
      mutations.updateMutation.mutate({ id: dialogs.editingItem.id, data: { name: data.name } });
    } else {
      mutations.createMutation.mutate(data);
    }
  };

  const columns = useMemo<Column<GovernmentFunding>[]>(
    () => [
      { key: 'id', header: 'common.id', render: (funding) => funding.id },
      {
        key: 'name',
        header: 'common.name',
        render: (funding) => funding.name,
        className: 'font-medium',
      },
      {
        key: 'state',
        header: 'states.state',
        render: (funding) => t(`states.${funding.state}`),
      },
    ],
    [t]
  );

  return (
    <div className="space-y-6">
      <CrudPageHeader
        title="governmentFundings.title"
        onNew={dialogs.handleCreate}
        newButtonText="governmentFundings.newGovernmentFunding"
      />

      <Card>
        <CardHeader>
          <CardTitle>{t('governmentFundings.title')}</CardTitle>
        </CardHeader>
        <CardContent>
          <ResourceTable
            items={paginatedData?.data}
            columns={columns}
            getItemKey={(funding) => funding.id}
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

      <CrudFormDialog
        open={dialogs.isDialogOpen}
        onOpenChange={dialogs.setIsDialogOpen}
        isEditing={dialogs.isEditing}
        translationPrefix="governmentFundings"
        onSubmit={handleSubmit(onSubmit)}
        isSaving={mutations.isMutating}
      >
        <div className="space-y-2">
          <Label htmlFor="name">{t('common.name')}</Label>
          <Input id="name" {...register('name')} />
          {errors.name && (
            <p className="text-destructive text-sm">{t('validation.nameRequired')}</p>
          )}
        </div>

        {!dialogs.isEditing && (
          <div className="space-y-2">
            <Label htmlFor="state">{t('states.state')}</Label>
            <Select value={watch('state')} onValueChange={(value) => setValue('state', value)}>
              <SelectTrigger>
                <SelectValue placeholder={t('states.selectState')} />
              </SelectTrigger>
              <SelectContent>
                {VALID_STATES.map((state) => (
                  <SelectItem key={state} value={state}>
                    {t(`states.${state}`)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {errors.state && (
              <p className="text-destructive text-sm">{t('validation.stateRequired')}</p>
            )}
          </div>
        )}
      </CrudFormDialog>

      <DeleteConfirmDialog
        open={dialogs.isDeleteDialogOpen}
        onOpenChange={dialogs.setIsDeleteDialogOpen}
        onConfirm={() =>
          dialogs.deletingItem && mutations.deleteMutation.mutate(dialogs.deletingItem.id)
        }
        isLoading={mutations.deleteMutation.isPending}
        resourceName="governmentFundings"
      />
    </div>
  );
}
