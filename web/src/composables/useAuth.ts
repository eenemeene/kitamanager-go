import { useAuthStore } from '@/stores/auth'
import { computed } from 'vue'

export function useAuth() {
  const authStore = useAuthStore()

  return {
    isAuthenticated: computed(() => authStore.isAuthenticated),
    user: computed(() => authStore.user),
    userId: computed(() => authStore.userId),
    userEmail: computed(() => authStore.userEmail),
    login: authStore.login,
    logout: authStore.logout
  }
}
