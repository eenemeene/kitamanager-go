import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '../stores/auth'

// Mock the router
vi.mock('../router', () => ({
  default: {
    push: vi.fn()
  }
}))

// Mock the API client
vi.mock('../api/client', () => ({
  apiClient: {
    login: vi.fn(),
    setOnUnauthorized: vi.fn()
  }
}))

describe('Auth Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  it('should initialize with no authentication', () => {
    const store = useAuthStore()
    expect(store.isAuthenticated).toBe(false)
    expect(store.token).toBeNull()
    expect(store.user).toBeNull()
  })

  it('should detect expired token as not authenticated', () => {
    // Create an expired JWT (exp in the past)
    const expiredPayload = {
      user_id: 1,
      email: 'test@example.com',
      exp: Math.floor(Date.now() / 1000) - 3600 // 1 hour ago
    }
    const expiredToken = `header.${btoa(JSON.stringify(expiredPayload))}.signature`
    localStorage.setItem('token', expiredToken)

    const store = useAuthStore()
    expect(store.isAuthenticated).toBe(false)
  })

  it('should detect valid token as authenticated', () => {
    // Create a valid JWT (exp in the future)
    const validPayload = {
      user_id: 1,
      email: 'test@example.com',
      exp: Math.floor(Date.now() / 1000) + 3600 // 1 hour from now
    }
    const validToken = `header.${btoa(JSON.stringify(validPayload))}.signature`
    localStorage.setItem('token', validToken)

    const store = useAuthStore()
    expect(store.isAuthenticated).toBe(true)
    expect(store.userId).toBe(1)
    expect(store.userEmail).toBe('test@example.com')
  })

  it('should clear token on logout', () => {
    const validPayload = {
      user_id: 1,
      email: 'test@example.com',
      exp: Math.floor(Date.now() / 1000) + 3600
    }
    const validToken = `header.${btoa(JSON.stringify(validPayload))}.signature`
    localStorage.setItem('token', validToken)

    const store = useAuthStore()
    expect(store.isAuthenticated).toBe(true)

    store.logout()

    expect(store.isAuthenticated).toBe(false)
    expect(store.token).toBeNull()
    expect(localStorage.getItem('token')).toBeNull()
  })
})
