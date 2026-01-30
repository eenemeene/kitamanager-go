'use client';

import { useMemo, useState } from 'react';
import { useParams } from 'next/navigation';
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
import { Switch } from '@/components/ui/switch';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { apiClient } from '@/lib/api/client';
import type { Group, GroupCreateRequest, GroupUpdateRequest } from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useCrudMutations } from '@/lib/hooks/use-crud-mutations';
import { useCrudDialogs } from '@/lib/hooks/use-crud-dialogs';
import { CrudPageHeader, ResourceTable, DeleteConfirmDialog, Column } from '@/components/crud';
import { Pagination } from '@/components/ui/pagination';

const groupSchema = z.object({
  name: z.string().min(1).max(255),
  active: z.boolean().default(true),
});

type GroupFormData = z.infer<typeof groupSchema>;

const defaultValues: GroupFormData = {
  name: '',
  active: true,
};

export default function GroupsPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const [page, setPage] = useState(1);

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    formState: { errors },
  } = useForm<GroupFormData>({
    resolver: zodResolver(groupSchema),
    defaultValues,
  });

  const { data: paginatedData, isLoading } = useQuery({
    queryKey: ['groups', orgId, page],
    queryFn: () => apiClient.getGroups(orgId, { page }),
    enabled: !!orgId,
  });

  const groups = paginatedData?.data;

  const dialogs = useCrudDialogs<Group, GroupFormData>({
    reset,
    itemToFormData: (group) => ({ name: group.name, active: group.active }),
    defaultValues,
  });

  const mutations = useCrudMutations<Group, GroupCreateRequest, GroupUpdateRequest>({
    resourceName: 'groups',
    queryKey: ['groups', orgId],
    createFn: (data) => apiClient.createGroup(orgId, data),
    updateFn: (id, data) => apiClient.updateGroup(orgId, id, data),
    deleteFn: (id) => apiClient.deleteGroup(orgId, id),
    onSuccess: dialogs.closeDialog,
    onDeleteSuccess: dialogs.closeDeleteDialog,
  });

  const onSubmit = (data: GroupFormData) => {
    if (dialogs.editingItem) {
      mutations.updateMutation.mutate({ id: dialogs.editingItem.id, data });
    } else {
      mutations.createMutation.mutate(data);
    }
  };

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

  const activeValue = watch('active');

  return (
    <div className="space-y-6">
      <CrudPageHeader
        title="groups.title"
        onNew={dialogs.handleCreate}
        newButtonText="groups.newGroup"
      />

      <Card>
        <CardHeader>
          <CardTitle>{t('groups.title')}</CardTitle>
        </CardHeader>
        <CardContent>
          <ResourceTable
            items={groups}
            columns={columns}
            getItemKey={(group) => group.id}
            isLoading={isLoading}
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
            <DialogTitle>{dialogs.isEditing ? t('groups.edit') : t('groups.create')}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">{t('common.name')}</Label>
              <Input id="name" {...register('name')} />
              {errors.name && (
                <p className="text-sm text-destructive">{t('validation.nameRequired')}</p>
              )}
            </div>

            <div className="flex items-center space-x-2">
              <Switch
                id="active"
                checked={activeValue}
                onCheckedChange={(checked) => setValue('active', checked)}
              />
              <Label htmlFor="active">{t('common.active')}</Label>
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
        resourceName="groups"
      />
    </div>
  );
}
