import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useUiStore = defineStore('ui', () => {
  const sidebarCollapsed = ref(false)
  const darkMode = ref(localStorage.getItem('darkMode') === 'true')
  const selectedOrganizationId = ref<number | null>(
    localStorage.getItem('selectedOrgId') ? Number(localStorage.getItem('selectedOrgId')) : null
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

  // Initialize dark mode on load
  updateDarkModeClass()

  return {
    sidebarCollapsed,
    darkMode,
    selectedOrganizationId,
    toggleSidebar,
    toggleDarkMode,
    setDarkMode,
    setSelectedOrganization
  }
})
