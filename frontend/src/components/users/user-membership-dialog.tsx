'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useToast } from '@/lib/hooks/use-toast';
import { apiClient, getErrorMessage } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import type { User, Role, UserMembership } from '@/lib/api/types';
import { useUiStore } from '@/stores/ui-store';

const ROLES: Role[] = ['admin', 'manager', 'member'];

interface UserMembershipDialogProps {
  user: User | null;
  orgId: number;
  onClose: () => void;
}

export function UserMembershipDialog({ user, orgId, onClose }: UserMembershipDialogProps) {
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const organizations = useUiStore((s) => s.organizations);
  const orgName = organizations.find((o) => o.id === orgId)?.name ?? `#${orgId}`;
  const [removeTarget, setRemoveTarget] = useState<UserMembership | null>(null);
  const [addRole, setAddRole] = useState<Role>('member');

  const { data: membershipsData, isLoading: membershipsLoading } = useQuery({
    queryKey: queryKeys.users.memberships(user?.id ?? 0),
    queryFn: () => apiClient.getUserMemberships(user!.id),
    enabled: !!user,
  });

  const orgMembership =
    membershipsData?.memberships?.find((m) => m.organization_id === orgId) ?? null;
  const otherMemberships =
    membershipsData?.memberships?.filter((m) => m.organization_id !== orgId) ?? [];

  const invalidateMemberships = () => {
    if (user) {
      queryClient.invalidateQueries({ queryKey: queryKeys.users.memberships(user.id) });
    }
  };

  const updateRoleMutation = useMutation({
    mutationFn: ({ organizationId, role }: { organizationId: number; role: Role }) =>
      apiClient.updateUserOrganizationRole(user!.id, organizationId, role),
    onSuccess: () => {
      toast({ title: t('users.roleUpdated') });
      invalidateMemberships();
    },
    onError: (error) => {
      toast({
        title: t('users.failedToUpdateRole'),
        description: getErrorMessage(error, t('common.error')),
        variant: 'destructive',
      });
    },
  });

  const removeMutation = useMutation({
    mutationFn: (organizationId: number) =>
      apiClient.removeUserFromOrganization(user!.id, organizationId),
    onSuccess: () => {
      toast({ title: t('users.removedFromOrganization', { orgName }) });
      invalidateMemberships();
      setRemoveTarget(null);
    },
    onError: (error) => {
      toast({
        title: t('users.failedToRemoveFromOrganization', { orgName }),
        description: getErrorMessage(error, t('common.error')),
        variant: 'destructive',
      });
    },
  });

  const addMutation = useMutation({
    mutationFn: (role: Role) => apiClient.addUserToOrganization(user!.id, orgId, role),
    onSuccess: () => {
      toast({ title: t('users.addedToOrganization', { orgName }) });
      invalidateMemberships();
      setAddRole('member');
    },
    onError: (error) => {
      toast({
        title: t('users.failedToAddToOrganization', { orgName }),
        description: getErrorMessage(error, t('common.error')),
        variant: 'destructive',
      });
    },
  });

  return (
    <>
      <Dialog open={!!user} onOpenChange={(open) => !open && onClose()}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {user
                ? t('users.organizationMembershipFor', { name: user.name })
                : t('users.organizationMembership')}
            </DialogTitle>
            <DialogDescription className="sr-only">
              {t('users.organizationMembership')}
            </DialogDescription>
          </DialogHeader>

          {membershipsLoading ? (
            <div className="text-muted-foreground py-4 text-center text-sm">
              {t('common.loading')}
            </div>
          ) : orgMembership ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('roles.role')}</TableHead>
                  <TableHead className="w-[60px]" />
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow>
                  <TableCell>
                    <Select
                      value={orgMembership.role}
                      onValueChange={(role) =>
                        updateRoleMutation.mutate({
                          organizationId: orgMembership.organization_id,
                          role: role as Role,
                        })
                      }
                    >
                      <SelectTrigger className="w-[140px]">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {ROLES.map((role) => (
                          <SelectItem key={role} value={role}>
                            {t(`roles.${role}`)}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => setRemoveTarget(orgMembership)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          ) : (
            <div className="space-y-2">
              <p className="text-muted-foreground text-sm">
                {t('users.addToThisOrganization', { orgName })}
              </p>
              <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                <Select value={addRole} onValueChange={(v) => setAddRole(v as Role)}>
                  <SelectTrigger className="w-full sm:w-[180px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {ROLES.map((role) => (
                      <SelectItem key={role} value={role}>
                        {t(`roles.${role}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Button
                  onClick={() => addMutation.mutate(addRole)}
                  disabled={addMutation.isPending}
                >
                  {t('users.addToOrganization', { orgName })}
                </Button>
              </div>
            </div>
          )}

          {!membershipsLoading && otherMemberships.length > 0 && (
            <div className="mt-4 space-y-2">
              <h4 className="text-muted-foreground text-xs font-semibold tracking-wide uppercase">
                {t('users.otherMemberships')}
              </h4>
              <ul className="max-h-40 space-y-1 overflow-y-auto text-sm">
                {otherMemberships.map((m) => (
                  <li
                    key={m.organization_id}
                    className="flex items-center justify-between gap-2 rounded border px-3 py-2"
                  >
                    <span>{m.organization?.name ?? `#${m.organization_id}`}</span>
                    <Badge variant="outline">{t(`roles.${m.role}`)}</Badge>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Remove confirmation */}
      <AlertDialog open={!!removeTarget} onOpenChange={(open) => !open && setRemoveTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('users.confirmRemoval')}</AlertDialogTitle>
            <AlertDialogDescription>
              {removeTarget ? t('users.removeFromOrganizationConfirm', { orgName }) : ''}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => removeTarget && removeMutation.mutate(removeTarget.organization_id)}
              disabled={removeMutation.isPending}
            >
              {t('common.delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
