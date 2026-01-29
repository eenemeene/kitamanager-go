<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { User, UserCreateRequest } from '@/api/types'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Checkbox from 'primevue/checkbox'
import Button from 'primevue/button'

const { t } = useI18n()

const props = defineProps<{
  visible: boolean
  user: User | null
}>()

const emit = defineEmits<{
  close: []
  save: [data: UserCreateRequest]
}>()

const form = ref({
  name: '',
  email: '',
  password: '',
  active: true
})

const errors = ref<{
  name?: string
  email?: string
  password?: string
}>({})

const isEditing = computed(() => !!props.user)
const dialogTitle = computed(() => (isEditing.value ? t('users.edit') : t('users.newUser')))

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      if (props.user) {
        form.value = {
          name: props.user.name,
          email: props.user.email,
          password: '',
          active: props.user.active
        }
      } else {
        form.value = {
          name: '',
          email: '',
          password: '',
          active: true
        }
      }
      errors.value = {}
    }
  }
)

function validate(): boolean {
  errors.value = {}

  if (!form.value.name.trim()) {
    errors.value.name = t('validation.nameRequired')
  }

  if (!form.value.email.trim()) {
    errors.value.email = t('validation.emailRequired')
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.value.email)) {
    errors.value.email = t('validation.invalidEmail')
  }

  // Password required for new users, optional for editing
  if (!isEditing.value && !form.value.password) {
    errors.value.password = t('validation.passwordRequired')
  } else if (form.value.password && form.value.password.length < 6) {
    errors.value.password = t('validation.passwordTooShort')
  }

  return Object.keys(errors.value).length === 0
}

function onSubmit() {
  if (!validate()) return

  const data: UserCreateRequest = {
    name: form.value.name.trim(),
    email: form.value.email,
    password: form.value.password,
    active: form.value.active
  }
  // Don't send empty password for updates
  if (isEditing.value && !form.value.password) {
    delete (data as Partial<UserCreateRequest>).password
  }
  emit('save', data)
}
</script>

<template>
  <Dialog
    :visible="visible"
    :header="dialogTitle"
    modal
    :closable="true"
    :style="{ width: '450px' }"
    @update:visible="$emit('close')"
  >
    <form @submit.prevent="onSubmit">
      <div class="form-grid">
        <div class="field">
          <label for="name">{{ t('common.name') }}</label>
          <InputText
            id="name"
            v-model="form.name"
            :class="{ 'p-invalid': errors.name }"
            :placeholder="t('common.name')"
          />
          <small v-if="errors.name" class="p-error">{{ errors.name }}</small>
        </div>

        <div class="field">
          <label for="email">{{ t('common.email') }}</label>
          <InputText
            id="email"
            v-model="form.email"
            type="email"
            :class="{ 'p-invalid': errors.email }"
            :placeholder="t('common.email')"
          />
          <small v-if="errors.email" class="p-error">{{ errors.email }}</small>
        </div>

        <div class="field">
          <label for="password">
            {{ t('users.password') }}
            <span v-if="isEditing" class="password-hint">{{ t('users.passwordHint') }}</span>
          </label>
          <Password
            id="password"
            v-model="form.password"
            :class="{ 'p-invalid': errors.password }"
            :feedback="false"
            toggle-mask
            :placeholder="t('users.password')"
            :input-style="{ width: '100%' }"
          />
          <small v-if="errors.password" class="p-error">{{ errors.password }}</small>
        </div>

        <div class="field">
          <div class="flex align-items-center gap-2">
            <Checkbox v-model="form.active" input-id="active" :binary="true" />
            <label for="active">{{ t('common.active') }}</label>
          </div>
        </div>
      </div>
    </form>

    <template #footer>
      <div class="dialog-footer">
        <Button :label="t('common.cancel')" text @click="$emit('close')" />
        <Button :label="t('common.save')" @click="onSubmit" />
      </div>
    </template>
  </Dialog>
</template>

<style scoped>
.flex {
  display: flex;
}

.align-items-center {
  align-items: center;
}

.gap-2 {
  gap: 0.5rem;
}

.password-hint {
  font-size: 0.875rem;
  color: var(--text-color-secondary);
  margin-left: 0.5rem;
}
</style>
