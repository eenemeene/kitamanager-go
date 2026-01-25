<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useToast } from 'primevue/usetoast'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Button from 'primevue/button'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const toast = useToast()

const email = ref('')
const password = ref('')
const loading = ref(false)
const errors = ref<{ email?: string; password?: string }>({})

async function handleSubmit() {
  errors.value = {}

  if (!email.value) {
    errors.value.email = t('validation.required')
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value)) {
    errors.value.email = t('validation.email')
  }

  if (!password.value) {
    errors.value.password = t('validation.required')
  }

  if (Object.keys(errors.value).length > 0) {
    return
  }

  loading.value = true
  try {
    await authStore.login({ email: email.value, password: password.value })
    router.push('/')
  } catch {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: t('auth.loginError'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-logo">
        <img src="/logo.svg" alt="KitaManager" />
      </div>
      <form @submit.prevent="handleSubmit">
        <div class="field">
          <label for="email">{{ t('auth.email') }}</label>
          <InputText
            id="email"
            v-model="email"
            type="email"
            :placeholder="t('auth.email')"
            :class="{ 'p-invalid': errors.email }"
          />
          <small v-if="errors.email" class="p-error">{{ errors.email }}</small>
        </div>

        <div class="field">
          <label for="password">{{ t('auth.password') }}</label>
          <Password
            id="password"
            v-model="password"
            :placeholder="t('auth.password')"
            :feedback="false"
            toggle-mask
            :class="{ 'p-invalid': errors.password }"
            :input-style="{ width: '100%' }"
          />
          <small v-if="errors.password" class="p-error">{{ errors.password }}</small>
        </div>

        <Button
          type="submit"
          :label="t('auth.loginButton')"
          :loading="loading"
          class="w-full"
          style="width: 100%"
        />
      </form>
    </div>
  </div>
</template>
