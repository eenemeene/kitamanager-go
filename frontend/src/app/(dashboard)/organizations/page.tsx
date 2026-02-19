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
import { queryKeys } from '@/lib/api/queryKeys';
import {
  type Organization,
  type OrganizationCreateRequest,
  type OrganizationUpdateRequest,
  DEFAULT_PAGE_SIZE,
  VALID_STATES,
} from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useCrudMutations } from '@/lib/hooks/use-crud-mutations';
import { useCrudDialogs } from '@/lib/hooks/use-crud-dialogs';
import { CrudPageHeader, ResourceTable, DeleteConfirmDialog, Column } from '@/components/crud';
import { Pagination } from '@/components/ui/pagination';
import { organizationSchema, type OrganizationFormData } from '@/lib/schemas';

const defaultValues: OrganizationFormData = {
  name: '',
  state: VALID_STATES[0],
  active: true,
  default_section_name: '',
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
    resolver: zodResolver(organizationSchema) as any,
    defaultValues,
  });

  const { data: paginatedData, isLoading } = useQuery({
    queryKey: queryKeys.organizations.list(page),
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
    queryKey: queryKeys.organizations.all(),
    createFn: (data) => apiClient.createOrganization(data),
    updateFn: (id, data) => apiClient.updateOrganization(id, data),
    deleteFn: (id) => apiClient.deleteOrganization(id),
    onSuccess: dialogs.closeDialog,
    onDeleteSuccess: dialogs.closeDeleteDialog,
  });

  const onSubmit = (data: OrganizationFormData) => {
    if (dialogs.editingItem) {
      const { default_section_name: _, ...updateData } = data;
      mutations.updateMutation.mutate({ id: dialogs.editingItem.id, data: updateData });
    } else {
      mutations.createMutation.mutate({
        ...data,
        default_section_name: data.default_section_name || '',
      });
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
          <form onSubmit={handleSubmit(onSubmit as never)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">{t('common.name')}</Label>
              <Input
                id="name"
                aria-invalid={!!errors.name}
                aria-describedby={errors.name ? 'name-error' : undefined}
                {...register('name')}
              />
              {errors.name && (
                <p id="name-error" className="text-destructive text-sm">
                  {t('validation.nameRequired')}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="state">{t('states.state')}</Label>
              <Select value={watch('state')} onValueChange={(value) => setValue('state', value)}>
                <SelectTrigger
                  aria-invalid={!!errors.state}
                  aria-describedby={errors.state ? 'state-error' : undefined}
                >
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
                <p id="state-error" className="text-destructive text-sm">
                  {t('validation.stateRequired')}
                </p>
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

            {!dialogs.isEditing && (
              <div className="space-y-2">
                <Label htmlFor="default_section_name">
                  {t('organizations.defaultSectionName')}
                </Label>
                <Input
                  id="default_section_name"
                  {...register('default_section_name')}
                  placeholder={t('organizations.defaultSectionNamePlaceholder')}
                />
                {errors.default_section_name && (
                  <p className="text-destructive text-sm">
                    {t('validation.defaultSectionNameRequired')}
                  </p>
                )}
              </div>
            )}

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
