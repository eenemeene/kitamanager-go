<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useUiStore } from '@/stores/ui'
import type { Organization } from '@/api/types'
import Dropdown from 'primevue/dropdown'

const emit = defineEmits<{
  navigate: []
}>()

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const uiStore = useUiStore()

// Check if current route is org-scoped
const isOnOrgScopedRoute = computed(() => {
  return route.matched.some((record) => record.meta.orgScoped)
})

const selectedOrg = computed({
  get: () => uiStore.selectedOrganization,
  set: (org: Organization | null) => {
    const newOrgId = org?.id || null
    uiStore.setSelectedOrganization(newOrgId)

    // If on an org-scoped route, navigate to the same route with new org
    if (isOnOrgScopedRoute.value && newOrgId && route.name) {
      router.push({
        name: route.name as string,
        params: { ...route.params, orgId: newOrgId }
      })
    }
  }
})

const navItems = computed(() => [
  { to: '/', icon: 'pi-home', label: t('nav.dashboard'), exact: true },
  { to: '/organizations', icon: 'pi-building', label: t('nav.organizations') },
  { to: '/government-fundings', icon: 'pi-money-bill', label: t('nav.governmentFundings') },
  ...(selectedOrg.value
    ? [
        {
          to: `/organizations/${selectedOrg.value.id}/users`,
          icon: 'pi-users',
          label: t('nav.users')
        },
        {
          to: `/organizations/${selectedOrg.value.id}/groups`,
          icon: 'pi-sitemap',
          label: t('nav.groups')
        },
        {
          to: `/organizations/${selectedOrg.value.id}/employees`,
          icon: 'pi-id-card',
          label: t('nav.employees')
        },
        {
          to: `/organizations/${selectedOrg.value.id}/children`,
          icon: 'pi-face-smile',
          label: t('nav.children')
        },
        {
          to: `/organizations/${selectedOrg.value.id}/payplans`,
          icon: 'pi-dollar',
          label: t('nav.payPlans')
        },
        {
          to: `/organizations/${selectedOrg.value.id}/statistics`,
          icon: 'pi-chart-bar',
          label: t('nav.statistics')
        }
      ]
    : [])
])

function isActive(item: { to: string; exact?: boolean }) {
  if (item.exact) {
    return route.path === item.to
  }
  return route.path.startsWith(item.to)
}

onMounted(() => {
  uiStore.fetchOrganizations()
})
</script>

<template>
  <aside class="app-sidebar">
    <div class="logo">
      <img src="/logo.svg" alt="KitaManager" class="logo-image" />
    </div>

    <div class="org-selector">
      <Dropdown
        v-model="selectedOrg"
        :options="uiStore.organizations"
        option-label="name"
        :placeholder="t('organizations.selectOrg')"
        class="w-full"
        :loading="uiStore.organizationsLoading"
        filter
        filter-placeholder="Search..."
      />
    </div>

    <nav class="nav-menu">
      <RouterLink
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        class="nav-item"
        :class="{ 'router-link-active': isActive(item) }"
        @click="emit('navigate')"
      >
        <i class="pi" :class="item.icon"></i>
        <span>{{ item.label }}</span>
      </RouterLink>
    </nav>
  </aside>
</template>

<style scoped>
.org-selector {
  padding: 1rem;
  border-bottom: 1px solid var(--surface-border);
}

.org-selector .w-full {
  width: 100%;
}
</style>
