'use client';

import { useMemo, useState } from 'react';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { apiClient } from '@/lib/api/client';
import type {
  Organization,
  OrganizationCreateRequest,
  OrganizationUpdateRequest,
} from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useCrudMutations } from '@/lib/hooks/use-crud-mutations';
import { useCrudDialogs } from '@/lib/hooks/use-crud-dialogs';
import { CrudPageHeader, ResourceTable, DeleteConfirmDialog, Column } from '@/components/crud';
import { Pagination } from '@/components/ui/pagination';
import { DEFAULT_PAGE_SIZE } from '@/lib/api/types';

const organizationSchema = z.object({
  name: z.string().min(1).max(255),
  state: z.string().min(1),
  active: z.boolean().default(true),
});

type OrganizationFormData = z.infer<typeof organizationSchema>;

const states = [{ value: 'berlin', label: 'Berlin' }];

const defaultValues: OrganizationFormData = {
  name: '',
  state: 'berlin',
  active: true,
};

export default function OrganizationsPage() {
  const t = useTranslations();
  const [page, setPage] = useState(1);

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    watch,
    formState: { errors },
  } = useForm<OrganizationFormData>({
    resolver: zodResolver(organizationSchema),
    defaultValues,
  });

  const { data: paginatedData, isLoading } = useQuery({
    queryKey: ['organizations', page],
    queryFn: () => apiClient.getOrganizations({ page, limit: DEFAULT_PAGE_SIZE }),
  });

  const organizations = paginatedData?.data;

  const dialogs = useCrudDialogs<Organization, OrganizationFormData>({
    reset,
    itemToFormData: (org) => ({ name: org.name, state: org.state, active: org.active }),
    defaultValues,
  });

  const mutations = useCrudMutations<
    Organization,
    OrganizationCreateRequest,
    OrganizationUpdateRequest
  >({
    resourceName: 'organizations',
    queryKey: ['organizations'],
    createFn: (data) => apiClient.createOrganization(data),
    updateFn: (id, data) => apiClient.updateOrganization(id, data),
    deleteFn: (id) => apiClient.deleteOrganization(id),
    onSuccess: dialogs.closeDialog,
    onDeleteSuccess: dialogs.closeDeleteDialog,
  });

  const onSubmit = (data: OrganizationFormData) => {
    if (dialogs.editingItem) {
      mutations.updateMutation.mutate({ id: dialogs.editingItem.id, data });
    } else {
      mutations.createMutation.mutate(data);
    }
  };

  const columns = useMemo<Column<Organization>[]>(
    () => [
      { key: 'id', header: 'common.id', render: (org) => org.id },
      { key: 'name', header: 'common.name', render: (org) => org.name, className: 'font-medium' },
      { key: 'state', header: 'states.state', render: (org) => t(`states.${org.state}`) },
      {
        key: 'status',
        header: 'common.status',
        render: (org) => (
          <Badge variant={org.active ? 'success' : 'secondary'}>
            {org.active ? t('common.active') : t('common.inactive')}
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
        title="organizations.title"
        onNew={dialogs.handleCreate}
        newButtonText="organizations.newOrganization"
      />

      <Card>
        <CardHeader>
          <CardTitle>{t('organizations.title')}</CardTitle>
        </CardHeader>
        <CardContent>
          <ResourceTable
            items={organizations}
            columns={columns}
            getItemKey={(org) => org.id}
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
            <DialogTitle>
              {dialogs.isEditing ? t('organizations.edit') : t('organizations.create')}
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

            <div className="space-y-2">
              <Label htmlFor="state">{t('states.state')}</Label>
              <Select value={watch('state')} onValueChange={(value) => setValue('state', value)}>
                <SelectTrigger>
                  <SelectValue placeholder={t('states.selectState')} />
                </SelectTrigger>
                <SelectContent>
                  {states.map((state) => (
                    <SelectItem key={state.value} value={state.value}>
                      {t(`states.${state.value}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {errors.state && (
                <p className="text-sm text-destructive">{t('validation.stateRequired')}</p>
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
        resourceName="organizations"
      />
    </div>
  );
}
