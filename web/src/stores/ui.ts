import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiClient } from '@/api/client'
import type { Organization } from '@/api/types'

export const useUiStore = defineStore('ui', () => {
  const sidebarCollapsed = ref(false)
  const darkMode = ref(localStorage.getItem('darkMode') === 'true')
  const selectedOrganizationId = ref<number | null>(
    localStorage.getItem('selectedOrgId') ? Number(localStorage.getItem('selectedOrgId')) : null
  )
  const organizations = ref<Organization[]>([])
  const organizationsLoading = ref(false)

  const selectedOrganization = computed(
    () => organizations.value.find((o) => o.id === selectedOrganizationId.value) || null
  )

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  function toggleDarkMode() {
    darkMode.value = !darkMode.value
    localStorage.setItem('darkMode', String(darkMode.value))
    updateDarkModeClass()
  }

  function setDarkMode(value: boolean) {
    darkMode.value = value
    localStorage.setItem('darkMode', String(value))
    updateDarkModeClass()
  }

  function updateDarkModeClass() {
    if (darkMode.value) {
      document.documentElement.classList.add('dark-mode')
    } else {
      document.documentElement.classList.remove('dark-mode')
    }
  }

  function setSelectedOrganization(orgId: number | null) {
    selectedOrganizationId.value = orgId
    if (orgId) {
      localStorage.setItem('selectedOrgId', String(orgId))
    } else {
      localStorage.removeItem('selectedOrgId')
    }
  }

  async function fetchOrganizations() {
    organizationsLoading.value = true
    try {
      organizations.value = await apiClient.getOrganizations()
      // Auto-select first org if none selected and orgs exist
      if (!selectedOrganizationId.value && organizations.value.length > 0) {
        setSelectedOrganization(organizations.value[0].id)
      }
    } catch (error) {
      console.error('Failed to load organizations:', error)
    } finally {
      organizationsLoading.value = false
    }
  }

  // Initialize dark mode on load
  updateDarkModeClass()

  return {
    sidebarCollapsed,
    darkMode,
    selectedOrganizationId,
    organizations,
    organizationsLoading,
    selectedOrganization,
    toggleSidebar,
    toggleDarkMode,
    setDarkMode,
    setSelectedOrganization,
    fetchOrganizations
  }
})
