'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Badge } from '@/components/ui/badge';
import { apiClient } from '@/lib/api/client';
import type { Group, GroupCreateRequest, GroupUpdateRequest } from '@/lib/api/types';
import { useCrudPage } from '@/lib/hooks/use-crud-page';
import {
  CrudPageHeader,
  ResourceTable,
  DeleteConfirmDialog,
  CrudFormDialog,
  Column,
} from '@/components/crud';
import { Pagination } from '@/components/ui/pagination';
import { groupSchema, type GroupFormData } from '@/lib/schemas';

const defaultValues: GroupFormData = {
  name: '',
  active: true,
};

export default function GroupsPage() {
  const t = useTranslations();
  const crud = useCrudPage<Group, GroupFormData, GroupCreateRequest, GroupUpdateRequest>({
    resourceName: 'groups',
    schema: groupSchema,
    defaultValues,
    itemToFormData: (group) => ({ name: group.name, active: group.active }),
    listFn: (orgId, params) => apiClient.getGroups(orgId, params),
    createFn: (orgId, data) => apiClient.createGroup(orgId, data),
    updateFn: (orgId, id, data) => apiClient.updateGroup(orgId, id, data),
    deleteFn: (orgId, id) => apiClient.deleteGroup(orgId, id),
  });

  const columns = useMemo<Column<Group>[]>(
    () => [
      { key: 'id', header: 'common.id', render: (group) => group.id },
      {
        key: 'name',
        header: 'common.name',
        render: (group) => group.name,
        className: 'font-medium',
      },
      {
        key: 'status',
        header: 'common.status',
        render: (group) => (
          <Badge variant={group.active ? 'success' : 'secondary'}>
            {group.active ? t('common.active') : t('common.inactive')}
          </Badge>
        ),
      },
    ],
    [t]
  );

  const activeValue = crud.watch('active');

  return (
    <div className="space-y-6">
      <CrudPageHeader
        title="groups.title"
        onNew={crud.dialogs.handleCreate}
        newButtonText="groups.newGroup"
      />

      <Card>
        <CardHeader>
          <CardTitle>{t('groups.title')}</CardTitle>
        </CardHeader>
        <CardContent>
          <ResourceTable
            items={crud.items}
            columns={columns}
            getItemKey={(group) => group.id}
            isLoading={crud.isLoading}
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
        translationPrefix="groups"
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

        <div className="flex items-center space-x-2">
          <Switch
            id="active"
            checked={activeValue}
            onCheckedChange={(checked) => crud.setValue('active', checked)}
          />
          <Label htmlFor="active">{t('common.active')}</Label>
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
        resourceName="groups"
      />
    </div>
  );
}
