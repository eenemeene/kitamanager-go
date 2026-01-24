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
      user.value = {
        id: payload.user_id,
        email: payload.email
      }
    }
  }

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem('token')
    router.push('/login')
  }

  // Initialize user from token on store creation
  function init() {
    if (token.value) {
      const payload = parseJwt(token.value)
      if (payload && payload.exp * 1000 > Date.now()) {
        user.value = {
          id: payload.user_id,
          email: payload.email
        }
      } else {
        // Token expired, clear it
        token.value = null
        user.value = null
        localStorage.removeItem('token')
      }
    }
  }

  init()

  return {
    token,
    user,
    isAuthenticated,
    userId,
    userEmail,
    login,
    logout
  }
})
