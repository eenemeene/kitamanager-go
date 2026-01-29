import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import type { User } from '@/api/types'

export function usePermissions() {
  const authStore = useAuthStore()

  const isSuperAdmin = computed(() => {
    return (authStore.user as User | null)?.is_superadmin ?? false
  })

  const userOrganizations = computed(() => {
    const user = authStore.user as User | null
    return user?.organizations ?? []
  })

  function canAccessOrganization(orgId: number): boolean {
    // Superadmins can access all organizations
    if (isSuperAdmin.value) {
      return true
    }

    // Check if user is a member of the organization
    return userOrganizations.value.some((org) => org.id === orgId)
  }

  function hasAnyOrganization(): boolean {
    return isSuperAdmin.value || userOrganizations.value.length > 0
  }

  return {
    isSuperAdmin,
    userOrganizations,
    canAccessOrganization,
    hasAnyOrganization
  }
}
