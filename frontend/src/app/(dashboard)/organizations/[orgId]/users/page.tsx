'use client';

import { useCallback, useMemo, useState } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useQueries, useMutation, useQueryClient } from '@tanstack/react-query';
import { Pencil, Trash2, Users } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { useToast } from '@/lib/hooks/use-toast';
import { apiClient, getErrorMessage } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import type { User, UserCreateRequest, UserUpdateRequest, UserMembership } from '@/lib/api/types';
import { Pagination } from '@/components/ui/pagination';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { formatDate } from '@/lib/utils/formatting';
import { useAuthStore } from '@/stores/auth-store';
import { useCrudMutations } from '@/lib/hooks/use-crud-mutations';
import { FormErrorSummary } from '@/components/forms/form-error-summary';
import { useProblemFormErrors } from '@/lib/forms/use-problem-form-errors';
import { getInvalidParams } from '@/lib/api/problem';
import { useCrudDialogs } from '@/lib/hooks/use-crud-dialogs';
import {
  CrudPageHeader,
  ResourceTable,
  DeleteConfirmDialog,
  CrudFormDialog,
  QueryError,
  Column,
} from '@/components/crud';
import {
  userCreateSchema,
  userUpdateSchema,
  type UserCreateFormData,
  type UserUpdateFormData,
} from '@/lib/schemas';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { UserMembershipDialog } from '@/components/users/user-membership-dialog';
import { validationTiming } from '@/lib/forms/validation-timing';

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
  const [membershipUser, setMembershipUser] = useState<User | null>(null);

  const {
    register: registerCreate,
    handleSubmit: handleSubmitCreate,
    reset: resetCreate,
    setValue: setValueCreate,
    watch: watchCreate,
    setError: setErrorCreate,
    clearErrors: clearErrorsCreate,
    getValues: getValuesCreate,
    formState: { errors: errorsCreate },
  } = useForm<UserCreateFormData>({
    ...validationTiming,
    resolver: zodResolver(userCreateSchema) as any,
    defaultValues: createDefaultValues,
  });

  const {
    register: registerUpdate,
    handleSubmit: handleSubmitUpdate,
    reset: resetUpdate,
    setValue: setValueUpdate,
    watch: watchUpdate,
    setError: setErrorUpdate,
    clearErrors: clearErrorsUpdate,
    getValues: getValuesUpdate,
    formState: { errors: errorsUpdate },
  } = useForm<UserUpdateFormData>({
    ...validationTiming,
    resolver: zodResolver(userUpdateSchema) as any,
    defaultValues: updateDefaultValues,
  });

  const {
    data: paginatedData,
    isLoading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: queryKeys.users.list(page),
    queryFn: () => apiClient.getUsers({ page }),
  });

  const users = paginatedData?.data;

  // Fetch memberships for all visible users
  const membershipQueries = useQueries({
    queries: (users ?? []).map((user) => ({
      queryKey: queryKeys.users.memberships(user.id),
      queryFn: () => apiClient.getUserMemberships(user.id),
    })),
  });

  const membershipsByUserId = useMemo(() => {
    const map = new Map<number, UserMembership | undefined>();
    if (!users) return map;
    users.forEach((user, i) => {
      const data = membershipQueries[i]?.data;
      if (data?.memberships) {
        map.set(
          user.id,
          data.memberships.find((m) => m.organization_id === orgId)
        );
      }
    });
    return map;
  }, [users, membershipQueries, orgId]);

  // Use separate dialog hooks for create (no edit item) and general dialogs
  const dialogs = useCrudDialogs<User, UserUpdateFormData>({
    reset: resetUpdate,
    itemToFormData: (user) => ({ name: user.name, email: user.email, active: user.active }),
    defaultValues: updateDefaultValues,
  });

  const userFieldLabels = {
    name: t('common.name'),
    email: t('common.email'),
    password: t('users.password'),
    active: t('common.active'),
  };

  const mutations = useCrudMutations<User, UserCreateRequest, UserUpdateRequest>({
    resourceName: 'users',
    // Two forms share this hook -- create and edit differ enough to warrant it --
    // so the marking is wired per form below rather than here, and this only
    // decides whether the toast would be a duplicate.
    onMutationError: (error) => getInvalidParams(error).length > 0,
    queryKey: queryKeys.users.all(),
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
      queryClient.invalidateQueries({ queryKey: queryKeys.users.all() });
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

  // Each form reacts to the mutation that submits it, so an edit rejection
  // cannot mark fields on the create form or the other way round.
  const unmappedCreate = useProblemFormErrors(mutations.createMutation.error, {
    setError: setErrorCreate,
    clearErrors: clearErrorsCreate,
    getValues: getValuesCreate,
  });
  const unmappedUpdate = useProblemFormErrors(mutations.updateMutation.error, {
    setError: setErrorUpdate,
    clearErrors: clearErrorsUpdate,
    getValues: getValuesUpdate,
  });

  const onSubmitCreate = (data: UserCreateFormData) => {
    mutations.createMutation.mutate(data);
  };

  const onSubmitUpdate = (data: UserUpdateFormData) => {
    if (dialogs.editingItem) {
      mutations.updateMutation.mutate({ id: dialogs.editingItem.id, data });
    }
  };

  const handleSuperadminToggle = useCallback(
    (user: User, checked: boolean) => {
      superadminMutation.mutate({ userId: user.id, isSuperadmin: checked });
    },
    [superadminMutation]
  );

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
      {
        key: 'role',
        header: 'roles.role',
        render: (user) => {
          const membership = membershipsByUserId.get(user.id);
          if (!membership) {
            return <span className="text-muted-foreground">—</span>;
          }
          return <Badge variant="outline">{t(`roles.${membership.role}`)}</Badge>;
        },
        className: 'hidden md:table-cell',
        headerClassName: 'hidden md:table-cell',
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
  }, [t, isSuperadmin, currentUser?.id, handleSuperadminToggle, membershipsByUserId]);

  const renderActions = (user: User) => (
    <>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button variant="ghost" size="icon" onClick={() => dialogs.handleEdit(user)}>
            <Pencil className="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{t('common.edit')}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button variant="ghost" size="icon" onClick={() => setMembershipUser(user)}>
            <Users className="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{t('users.manageMemberships')}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => dialogs.handleDelete(user)}
            disabled={user.id === currentUser?.id}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{t('common.delete')}</TooltipContent>
      </Tooltip>
    </>
  );

  return (
    <div className="space-y-6">
      <CrudPageHeader
        title="users.title"
        description="users.description"
        onNew={handleCreate}
        newButtonText="users.newUser"
      />

      <Card>
        <CardHeader>
          <CardTitle>{t('users.title')}</CardTitle>
        </CardHeader>
        <CardContent>
          <QueryError error={queryError} onRetry={refetch} />
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

      <CrudFormDialog
        open={dialogs.isDialogOpen}
        onOpenChange={dialogs.setIsDialogOpen}
        isEditing={dialogs.isEditing}
        translationPrefix="users"
        onSubmit={
          dialogs.isEditing
            ? handleSubmitUpdate(onSubmitUpdate as never)
            : handleSubmitCreate(onSubmitCreate as never)
        }
        isSaving={mutations.isMutating}
      >
        {dialogs.isEditing ? (
          <>
            <FormErrorSummary
              errors={errorsUpdate}
              unmapped={unmappedUpdate}
              labels={userFieldLabels}
            />
            <div className="space-y-2">
              <Label htmlFor="name">{t('common.name')}</Label>
              <Input id="name" {...registerUpdate('name')} />
              {errorsUpdate.name && (
                <p className="text-destructive text-sm">{t('validation.nameRequired')}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="email">{t('common.email')}</Label>
              <Input id="email" type="email" {...registerUpdate('email')} />
              {errorsUpdate.email && (
                <p className="text-destructive text-sm">{t('validation.invalidEmail')}</p>
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
          </>
        ) : (
          <>
            <FormErrorSummary
              errors={errorsCreate}
              unmapped={unmappedCreate}
              labels={userFieldLabels}
            />
            <div className="space-y-2">
              <Label htmlFor="name">{t('common.name')}</Label>
              <Input id="name" {...registerCreate('name')} />
              {errorsCreate.name && (
                <p className="text-destructive text-sm">{t('validation.nameRequired')}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="email">{t('common.email')}</Label>
              <Input id="email" type="email" {...registerCreate('email')} />
              {errorsCreate.email && (
                <p className="text-destructive text-sm">{t('validation.invalidEmail')}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">{t('users.password')}</Label>
              <Input id="password" type="password" {...registerCreate('password')} />
              {errorsCreate.password && (
                <p className="text-destructive text-sm">{t('validation.passwordTooShort')}</p>
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
          </>
        )}
      </CrudFormDialog>

      <DeleteConfirmDialog
        open={dialogs.isDeleteDialogOpen}
        onOpenChange={dialogs.setIsDeleteDialogOpen}
        onConfirm={() =>
          dialogs.deletingItem && mutations.deleteMutation.mutate(dialogs.deletingItem.id)
        }
        isLoading={mutations.deleteMutation.isPending}
        resourceName="users"
      />

      <UserMembershipDialog
        user={membershipUser}
        orgId={orgId}
        onClose={() => setMembershipUser(null)}
      />
    </div>
  );
}
