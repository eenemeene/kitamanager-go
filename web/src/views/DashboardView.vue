<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'
import { useUiStore } from '@/stores/ui'

const { t } = useI18n()
const uiStore = useUiStore()

const stats = ref({
  organizations: 0,
  employees: 0,
  children: 0,
  users: 0
})

const loading = ref(true)

async function loadStats() {
  loading.value = true
  try {
    const [orgs, users] = await Promise.all([apiClient.getOrganizations(), apiClient.getUsers()])

    stats.value.organizations = orgs.length
    stats.value.users = users.length

    // Load employees and children for selected org
    if (uiStore.selectedOrganizationId) {
      const [employees, children] = await Promise.all([
        apiClient.getEmployees(uiStore.selectedOrganizationId),
        apiClient.getChildren(uiStore.selectedOrganizationId)
      ])
      stats.value.employees = employees.length
      stats.value.children = children.length
    }
  } catch (error) {
    console.error('Failed to load dashboard stats:', error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadStats()
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1>{{ t('dashboard.title') }}</h1>
    </div>

    <div class="dashboard-grid">
      <div class="stat-card">
        <div class="icon blue">
          <i class="pi pi-building"></i>
        </div>
        <div class="details">
          <h3>{{ stats.organizations }}</h3>
          <p>{{ t('dashboard.totalOrganizations') }}</p>
        </div>
      </div>

      <div class="stat-card">
        <div class="icon green">
          <i class="pi pi-id-card"></i>
        </div>
        <div class="details">
          <h3>{{ stats.employees }}</h3>
          <p>{{ t('dashboard.totalEmployees') }}</p>
        </div>
      </div>

      <div class="stat-card">
        <div class="icon orange">
          <i class="pi pi-face-smile"></i>
        </div>
        <div class="details">
          <h3>{{ stats.children }}</h3>
          <p>{{ t('dashboard.totalChildren') }}</p>
        </div>
      </div>

      <div class="stat-card">
        <div class="icon purple">
          <i class="pi pi-users"></i>
        </div>
        <div class="details">
          <h3>{{ stats.users }}</h3>
          <p>{{ t('dashboard.totalUsers') }}</p>
        </div>
      </div>
    </div>
  </div>
</template>
