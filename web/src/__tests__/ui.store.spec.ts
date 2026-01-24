import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUiStore } from '../stores/ui'
import { apiClient } from '../api/client'

// Mock the apiClient
vi.mock('../api/client', () => ({
  apiClient: {
    getOrganizations: vi.fn()
  }
}))

describe('UI Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    document.documentElement.classList.remove('dark-mode')
    vi.clearAllMocks()
  })

  it('should initialize with default values', () => {
    const store = useUiStore()
    expect(store.sidebarCollapsed).toBe(false)
    expect(store.darkMode).toBe(false)
    expect(store.selectedOrganizationId).toBeNull()
  })

  it('should toggle sidebar', () => {
    const store = useUiStore()
    expect(store.sidebarCollapsed).toBe(false)

    store.toggleSidebar()
    expect(store.sidebarCollapsed).toBe(true)

    store.toggleSidebar()
    expect(store.sidebarCollapsed).toBe(false)
  })

  it('should toggle dark mode and persist to localStorage', () => {
    const store = useUiStore()
    expect(store.darkMode).toBe(false)

    store.toggleDarkMode()

    expect(store.darkMode).toBe(true)
    expect(localStorage.getItem('darkMode')).toBe('true')
    expect(document.documentElement.classList.contains('dark-mode')).toBe(true)

    store.toggleDarkMode()

    expect(store.darkMode).toBe(false)
    expect(localStorage.getItem('darkMode')).toBe('false')
    expect(document.documentElement.classList.contains('dark-mode')).toBe(false)
  })

  it('should set dark mode directly', () => {
    const store = useUiStore()

    store.setDarkMode(true)
    expect(store.darkMode).toBe(true)

    store.setDarkMode(false)
    expect(store.darkMode).toBe(false)
  })

  it('should set selected organization and persist to localStorage', () => {
    const store = useUiStore()

    store.setSelectedOrganization(42)
    expect(store.selectedOrganizationId).toBe(42)
    expect(localStorage.getItem('selectedOrgId')).toBe('42')

    store.setSelectedOrganization(null)
    expect(store.selectedOrganizationId).toBeNull()
    expect(localStorage.getItem('selectedOrgId')).toBeNull()
  })

  it('should restore dark mode from localStorage', () => {
    localStorage.setItem('darkMode', 'true')

    const store = useUiStore()
    expect(store.darkMode).toBe(true)
  })

  it('should restore selected organization from localStorage', () => {
    localStorage.setItem('selectedOrgId', '123')

    const store = useUiStore()
    expect(store.selectedOrganizationId).toBe(123)
  })

  describe('fetchOrganizations', () => {
    const createMockOrg = (id: number, name: string) => ({
      id,
      name,
      active: true,
      created_at: '2024-01-01T00:00:00Z',
      created_by: 'test',
      updated_at: '2024-01-01T00:00:00Z'
    })

    it('should fetch organizations and update the store', async () => {
      const mockOrgs = [createMockOrg(1, 'Org 1'), createMockOrg(2, 'Org 2')]
      vi.mocked(apiClient.getOrganizations).mockResolvedValue(mockOrgs)

      const store = useUiStore()
      await store.fetchOrganizations()

      expect(apiClient.getOrganizations).toHaveBeenCalled()
      expect(store.organizations).toEqual(mockOrgs)
      expect(store.organizationsLoading).toBe(false)
    })

    it('should auto-select first org if none selected', async () => {
      const mockOrgs = [createMockOrg(1, 'Org 1'), createMockOrg(2, 'Org 2')]
      vi.mocked(apiClient.getOrganizations).mockResolvedValue(mockOrgs)

      const store = useUiStore()
      expect(store.selectedOrganizationId).toBeNull()

      await store.fetchOrganizations()

      expect(store.selectedOrganizationId).toBe(1)
    })

    it('should not change selection if org already selected', async () => {
      const mockOrgs = [createMockOrg(1, 'Org 1'), createMockOrg(2, 'Org 2')]
      vi.mocked(apiClient.getOrganizations).mockResolvedValue(mockOrgs)

      const store = useUiStore()
      store.setSelectedOrganization(2)

      await store.fetchOrganizations()

      expect(store.selectedOrganizationId).toBe(2)
    })

    it('should set loading state during fetch', async () => {
      let resolvePromise: (value: unknown) => void
      const pendingPromise = new Promise((resolve) => {
        resolvePromise = resolve
      })
      vi.mocked(apiClient.getOrganizations).mockReturnValue(pendingPromise as Promise<never>)

      const store = useUiStore()
      const fetchPromise = store.fetchOrganizations()

      expect(store.organizationsLoading).toBe(true)

      resolvePromise!([])
      await fetchPromise

      expect(store.organizationsLoading).toBe(false)
    })

    it('should handle fetch errors gracefully', async () => {
      vi.mocked(apiClient.getOrganizations).mockRejectedValue(new Error('Network error'))
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const store = useUiStore()
      await store.fetchOrganizations()

      expect(store.organizationsLoading).toBe(false)
      expect(store.organizations).toEqual([])
      expect(consoleSpy).toHaveBeenCalledWith('Failed to load organizations:', expect.any(Error))

      consoleSpy.mockRestore()
    })
  })

  describe('selectedOrganization computed', () => {
    const createMockOrg = (id: number, name: string) => ({
      id,
      name,
      active: true,
      created_at: '2024-01-01T00:00:00Z',
      created_by: 'test',
      updated_at: '2024-01-01T00:00:00Z'
    })

    it('should return null when no org selected', async () => {
      const mockOrgs = [createMockOrg(1, 'Org 1'), createMockOrg(2, 'Org 2')]
      vi.mocked(apiClient.getOrganizations).mockResolvedValue(mockOrgs)

      const store = useUiStore()
      // Manually set organizations without triggering auto-select
      store.organizations.push(...mockOrgs)

      store.setSelectedOrganization(null)
      expect(store.selectedOrganization).toBeNull()
    })

    it('should return the selected organization object', async () => {
      const mockOrgs = [createMockOrg(1, 'Org 1'), createMockOrg(2, 'Org 2')]
      vi.mocked(apiClient.getOrganizations).mockResolvedValue(mockOrgs)

      const store = useUiStore()
      await store.fetchOrganizations()
      store.setSelectedOrganization(2)

      expect(store.selectedOrganization).toEqual(createMockOrg(2, 'Org 2'))
    })

    it('should return null if selected id not in organizations list', () => {
      const store = useUiStore()
      store.setSelectedOrganization(999)

      expect(store.selectedOrganization).toBeNull()
    })
  })
})
