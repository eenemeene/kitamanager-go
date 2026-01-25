<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useUiStore } from '@/stores/ui'
import Dropdown from 'primevue/dropdown'

const { t } = useI18n()
const route = useRoute()
const uiStore = useUiStore()

const selectedOrg = computed({
  get: () => uiStore.selectedOrganization,
  set: (org) => {
    uiStore.setSelectedOrganization(org?.id || null)
  }
})

const navItems = computed(() => [
  { to: '/', icon: 'pi-home', label: t('nav.dashboard'), exact: true },
  { to: '/organizations', icon: 'pi-building', label: t('nav.organizations') },
  { to: '/users', icon: 'pi-users', label: t('nav.users') },
  { to: '/groups', icon: 'pi-sitemap', label: t('nav.groups') },
  ...(selectedOrg.value
    ? [
        {
          to: `/organizations/${selectedOrg.value.id}/employees`,
          icon: 'pi-id-card',
          label: t('nav.employees')
        },
        {
          to: `/organizations/${selectedOrg.value.id}/children`,
          icon: 'pi-face-smile',
          label: t('nav.children')
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
      />
    </div>

    <nav class="nav-menu">
      <RouterLink
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        class="nav-item"
        :class="{ 'router-link-active': isActive(item) }"
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
