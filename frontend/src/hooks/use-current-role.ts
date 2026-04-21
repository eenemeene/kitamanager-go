import type { Role } from '@/lib/api/types';
import { useAuthStore } from '@/stores/auth-store';
import { useUiStore } from '@/stores/ui-store';

export type EffectiveRole = Role | 'superadmin';

const ROLE_HIERARCHY: Record<EffectiveRole, number> = {
  superadmin: 4,
  admin: 3,
  manager: 2,
  member: 1,
  staff: 0,
};

export function useCurrentRole(): EffectiveRole | null {
  const user = useAuthStore((s) => s.user);
  const orgRoleMap = useAuthStore((s) => s.orgRoleMap);
  const selectedOrganizationId = useUiStore((s) => s.selectedOrganizationId);

  if (!user) return null;
  if (user.is_superadmin) return 'superadmin';
  if (!selectedOrganizationId) return null;
  return orgRoleMap.get(selectedOrganizationId) ?? null;
}

export function hasMinimumRole(current: EffectiveRole | null, required: EffectiveRole): boolean {
  if (!current) return false;
  return ROLE_HIERARCHY[current] >= ROLE_HIERARCHY[required];
}
