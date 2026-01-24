<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { setLocale, getLocale, type SupportedLocale } from '@/i18n'
import Button from 'primevue/button'
import Menu from 'primevue/menu'
import Select from 'primevue/select'
import type { MenuItem } from 'primevue/menuitem'

const { t } = useI18n()
const authStore = useAuthStore()
const uiStore = useUiStore()
const menu = ref()

const currentLocale = ref<SupportedLocale>(getLocale())

const languageOptions = [
  { label: 'English', value: 'en' as SupportedLocale },
  { label: 'Deutsch', value: 'de' as SupportedLocale }
]

function changeLanguage(locale: SupportedLocale) {
  currentLocale.value = locale
  setLocale(locale)
}

function onLanguageChange(event: { value: SupportedLocale }) {
  changeLanguage(event.value)
}

const menuItems = computed<MenuItem[]>(() => [
  {
    label: t('common.name'),
    icon: 'pi pi-user',
    command: () => {
      // TODO: Navigate to profile
    }
  },
  {
    separator: true
  },
  {
    label: t('auth.logout'),
    icon: 'pi pi-sign-out',
    command: () => {
      authStore.logout()
    }
  }
])

function toggleMenu(event: Event) {
  menu.value.toggle(event)
}
</script>

<template>
  <header class="app-header">
    <Select
      v-model="currentLocale"
      :options="languageOptions"
      option-label="label"
      option-value="value"
      class="language-select"
      :aria-label="t('settings.language')"
      @change="onLanguageChange"
    />

    <Button
      :icon="uiStore.darkMode ? 'pi pi-sun' : 'pi pi-moon'"
      text
      rounded
      @click="uiStore.toggleDarkMode"
      :aria-label="uiStore.darkMode ? t('settings.lightMode') : t('settings.darkMode')"
    />

    <Button
      type="button"
      :label="authStore.userEmail || 'User'"
      icon="pi pi-user"
      @click="toggleMenu"
      text
    />
    <Menu ref="menu" :model="menuItems" popup />
  </header>
</template>
