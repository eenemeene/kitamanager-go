import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiClient } from '@/api/client'
import type { LoginRequest, User } from '@/api/types'
import router from '@/router'

interface JwtPayload {
  user_id: number
  email: string
  exp: number
}

function parseJwt(token: string): JwtPayload | null {
  try {
    const base64Url = token.split('.')[1]
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/')
    const jsonPayload = decodeURIComponent(
      atob(base64)
        .split('')
        .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join('')
    )
    return JSON.parse(jsonPayload)
  } catch {
    return null
  }
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('token'))
  const user = ref<Partial<User> | null>(null)
  const userLoading = ref(false)
  const userLoaded = ref(false)

  const isAuthenticated = computed(() => {
    if (!token.value) return false
    const payload = parseJwt(token.value)
    if (!payload) return false
    // Check if token is expired
    return payload.exp * 1000 > Date.now()
  })

  const userId = computed(() => {
    if (!token.value) return null
    const payload = parseJwt(token.value)
    return payload?.user_id ?? null
  })

  const userEmail = computed(() => {
    if (!token.value) return null
    const payload = parseJwt(token.value)
    return payload?.email ?? null
  })

  // Set up unauthorized callback
  apiClient.setOnUnauthorized(() => {
    logout()
  })

  async function login(credentials: LoginRequest) {
    const response = await apiClient.login(credentials)
    token.value = response.token
    localStorage.setItem('token', response.token)

    // Parse user info from token
    const payload = parseJwt(response.token)
    if (payload) {
      // Fetch full user data to get is_superadmin and other fields
      try {
        const userData = await apiClient.getUser(payload.user_id)
        user.value = userData
      } catch {
        // Fallback to basic info from token
        user.value = {
          id: payload.user_id,
          email: payload.email
        }
      }
    }
  }

  function logout() {
    token.value = null
    user.value = null
    // Clear all auth-related localStorage to prevent state leakage
    localStorage.removeItem('token')
    localStorage.removeItem('selectedOrgId')
    router.push('/login')
  }

  // Promise that resolves when user data is fully loaded
  let initPromise: Promise<void> | null = null

  // Initialize user from token on store creation
  async function init() {
    if (token.value) {
      const payload = parseJwt(token.value)
      if (payload && payload.exp * 1000 > Date.now()) {
        userLoading.value = true
        // Set basic info immediately
        user.value = {
          id: payload.user_id,
          email: payload.email
        }
        // Fetch full user data with timeout to prevent hanging
        try {
          const timeoutPromise = new Promise<never>((_, reject) =>
            setTimeout(() => reject(new Error('Timeout')), 5000)
          )
          const userData = await Promise.race([apiClient.getUser(payload.user_id), timeoutPromise])
          user.value = userData
        } catch {
          // Keep basic info on error or timeout
        } finally {
          userLoading.value = false
          userLoaded.value = true
        }
      } else {
        // Token expired, clear it
        token.value = null
        user.value = null
        localStorage.removeItem('token')
        userLoaded.value = true
      }
    } else {
      userLoaded.value = true
    }
  }

  // Wait for user data to be fully loaded
  async function ensureUserLoaded(): Promise<void> {
    if (userLoaded.value) return
    if (!initPromise) {
      initPromise = init()
    }
    await initPromise
  }

  initPromise = init()

  return {
    token,
    user,
    userLoading,
    userLoaded,
    isAuthenticated,
    userId,
    userEmail,
    login,
    logout,
    ensureUserLoaded
  }
})
