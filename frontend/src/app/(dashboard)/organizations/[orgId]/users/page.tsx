'use client';

import { useMemo, useState } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Pencil, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
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
import { useToast } from '@/lib/hooks/use-toast';
import { apiClient, getErrorMessage } from '@/lib/api/client';
import type { User, UserCreateRequest, UserUpdateRequest } from '@/lib/api/types';
import { Pagination } from '@/components/ui/pagination';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { formatDate } from '@/lib/utils/formatting';
import { useAuthStore } from '@/stores/auth-store';
import { useCrudMutations } from '@/lib/hooks/use-crud-mutations';
import { useCrudDialogs } from '@/lib/hooks/use-crud-dialogs';
import { CrudPageHeader, ResourceTable, DeleteConfirmDialog, Column } from '@/components/crud';

const userCreateSchema = z.object({
  name: z.string().min(1).max(255),
  email: z.string().email(),
  password: z.string().min(6),
  active: z.boolean().default(true),
});

const userUpdateSchema = z.object({
  name: z.string().min(1).max(255),
  email: z.string().email(),
  active: z.boolean().default(true),
});

type UserCreateFormData = z.infer<typeof userCreateSchema>;
type UserUpdateFormData = z.infer<typeof userUpdateSchema>;

const createDefaultValues: UserCreateFormData = {
  name: '',
  email: '',
  password: '',
  active: true,
};

const updateDefaultValues: UserUpdateFormData = {
  name: '',
  email: '',
  active: true,
};

export default function UsersPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const { user: currentUser } = useAuthStore();
  const [page, setPage] = useState(1);

  const {
    register: registerCreate,
    handleSubmit: handleSubmitCreate,
    reset: resetCreate,
    setValue: setValueCreate,
    watch: watchCreate,
    formState: { errors: errorsCreate },
  } = useForm<UserCreateFormData>({
    resolver: zodResolver(userCreateSchema),
    defaultValues: createDefaultValues,
  });

  const {
    register: registerUpdate,
    handleSubmit: handleSubmitUpdate,
    reset: resetUpdate,
    setValue: setValueUpdate,
    watch: watchUpdate,
    formState: { errors: errorsUpdate },
  } = useForm<UserUpdateFormData>({
    resolver: zodResolver(userUpdateSchema),
    defaultValues: updateDefaultValues,
  });

  const { data: paginatedData, isLoading } = useQuery({
    queryKey: ['users', page],
    queryFn: () => apiClient.getUsers({ page }),
  });

  const users = paginatedData?.data;

  // Use separate dialog hooks for create (no edit item) and general dialogs
  const dialogs = useCrudDialogs<User, UserUpdateFormData>({
    reset: resetUpdate,
    itemToFormData: (user) => ({ name: user.name, email: user.email, active: user.active }),
    defaultValues: updateDefaultValues,
  });

  const mutations = useCrudMutations<User, UserCreateRequest, UserUpdateRequest>({
    resourceName: 'users',
    queryKey: ['users'],
    createFn: (data) => apiClient.createUser(data),
    updateFn: (id, data) => apiClient.updateUser(id, data),
    deleteFn: (id) => apiClient.deleteUser(id),
    onSuccess: () => {
      dialogs.closeDialog();
      resetCreate(createDefaultValues);
    },
    onDeleteSuccess: dialogs.closeDeleteDialog,
  });

  const superadminMutation = useMutation({
    mutationFn: ({ userId, isSuperadmin }: { userId: number; isSuperadmin: boolean }) =>
      apiClient.setSuperAdmin(userId, isSuperadmin),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
      toast({ title: t('common.success') });
    },
    onError: (error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.error')),
        variant: 'destructive',
      });
    },
  });

  const handleCreate = () => {
    dialogs.setIsDialogOpen(true);
    resetCreate(createDefaultValues);
  };

  const onSubmitCreate = (data: UserCreateFormData) => {
    mutations.createMutation.mutate(data);
  };

  const onSubmitUpdate = (data: UserUpdateFormData) => {
    if (dialogs.editingItem) {
      mutations.updateMutation.mutate({ id: dialogs.editingItem.id, data });
    }
  };

  const handleSuperadminToggle = (user: User, checked: boolean) => {
    superadminMutation.mutate({ userId: user.id, isSuperadmin: checked });
  };

  const isSuperadmin = currentUser?.is_superadmin;

  const columns = useMemo<Column<User>[]>(() => {
    const baseColumns: Column<User>[] = [
      { key: 'id', header: 'common.id', render: (user) => user.id },
      { key: 'name', header: 'common.name', render: (user) => user.name, className: 'font-medium' },
      { key: 'email', header: 'common.email', render: (user) => user.email },
      {
        key: 'status',
        header: 'common.status',
        render: (user) => (
          <Badge variant={user.active ? 'success' : 'secondary'}>
            {user.active ? t('common.active') : t('common.inactive')}
          </Badge>
        ),
      },
    ];

    if (isSuperadmin) {
      baseColumns.push({
        key: 'superadmin',
        header: 'users.superadmin',
        render: (user) => (
          <Switch
            checked={user.is_superadmin}
            onCheckedChange={(checked) => handleSuperadminToggle(user, checked)}
            disabled={user.id === currentUser?.id}
          />
        ),
      });
    }

    baseColumns.push({
      key: 'lastLogin',
      header: 'users.lastLogin',
      render: (user) => formatDate(user.last_login),
    });

    return baseColumns;
  }, [t, isSuperadmin, currentUser?.id]);

  const renderActions = (user: User) => (
    <>
      <Button variant="ghost" size="icon" onClick={() => dialogs.handleEdit(user)}>
        <Pencil className="h-4 w-4" />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        onClick={() => dialogs.handleDelete(user)}
        disabled={user.id === currentUser?.id}
      >
        <Trash2 className="h-4 w-4" />
      </Button>
    </>
  );

  return (
    <div className="space-y-6">
      <CrudPageHeader title="users.title" onNew={handleCreate} newButtonText="users.newUser" />

      <Card>
        <CardHeader>
          <CardTitle>{t('users.title')}</CardTitle>
        </CardHeader>
        <CardContent>
          <ResourceTable
            items={users}
            columns={columns}
            getItemKey={(user) => user.id}
            isLoading={isLoading}
            renderActions={renderActions}
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
            <DialogTitle>{dialogs.isEditing ? t('users.edit') : t('users.create')}</DialogTitle>
          </DialogHeader>
          {dialogs.isEditing ? (
            <form onSubmit={handleSubmitUpdate(onSubmitUpdate)} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="name">{t('common.name')}</Label>
                <Input id="name" {...registerUpdate('name')} />
                {errorsUpdate.name && (
                  <p className="text-sm text-destructive">{t('validation.nameRequired')}</p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="email">{t('common.email')}</Label>
                <Input id="email" type="email" {...registerUpdate('email')} />
                {errorsUpdate.email && (
                  <p className="text-sm text-destructive">{t('validation.invalidEmail')}</p>
                )}
              </div>

              <div className="flex items-center space-x-2">
                <Switch
                  id="active"
                  checked={watchUpdate('active')}
                  onCheckedChange={(checked) => setValueUpdate('active', checked)}
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
                <Button type="submit" disabled={mutations.updateMutation.isPending}>
                  {t('common.save')}
                </Button>
              </DialogFooter>
            </form>
          ) : (
            <form onSubmit={handleSubmitCreate(onSubmitCreate)} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="name">{t('common.name')}</Label>
                <Input id="name" {...registerCreate('name')} />
                {errorsCreate.name && (
                  <p className="text-sm text-destructive">{t('validation.nameRequired')}</p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="email">{t('common.email')}</Label>
                <Input id="email" type="email" {...registerCreate('email')} />
                {errorsCreate.email && (
                  <p className="text-sm text-destructive">{t('validation.invalidEmail')}</p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="password">{t('users.password')}</Label>
                <Input id="password" type="password" {...registerCreate('password')} />
                {errorsCreate.password && (
                  <p className="text-sm text-destructive">{t('validation.passwordTooShort')}</p>
                )}
              </div>

              <div className="flex items-center space-x-2">
                <Switch
                  id="active"
                  checked={watchCreate('active')}
                  onCheckedChange={(checked) => setValueCreate('active', checked)}
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
                <Button type="submit" disabled={mutations.createMutation.isPending}>
                  {t('common.save')}
                </Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>

      <DeleteConfirmDialog
        open={dialogs.isDeleteDialogOpen}
        onOpenChange={dialogs.setIsDeleteDialogOpen}
        onConfirm={() =>
          dialogs.deletingItem && mutations.deleteMutation.mutate(dialogs.deletingItem.id)
        }
        isLoading={mutations.deleteMutation.isPending}
        resourceName="users"
      />
    </div>
  );
}
