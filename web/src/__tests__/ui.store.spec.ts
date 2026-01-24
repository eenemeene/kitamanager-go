import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUiStore } from '../stores/ui'

describe('UI Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    document.documentElement.classList.remove('dark-mode')
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
})
