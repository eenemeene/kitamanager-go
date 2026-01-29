<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { z } from 'zod'
import type { User, UserCreateRequest } from '@/api/types'
import { useFormValidation } from '@/composables/useFormValidation'
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

const isEditing = computed(() => !!props.user)
const dialogTitle = computed(() => (isEditing.value ? t('users.edit') : t('users.newUser')))

// Dynamic schema based on whether we're editing or creating
const userSchema = computed(() =>
  z.object({
    name: z
      .string({ required_error: 'validation.nameRequired' })
      .min(1, 'validation.nameRequired')
      .transform((v) => v.trim()),
    email: z
      .string({ required_error: 'validation.emailRequired' })
      .email('validation.invalidEmail'),
    password: isEditing.value
      ? z.string().min(6, 'validation.passwordTooShort').optional().or(z.literal(''))
      : z
          .string({ required_error: 'validation.passwordRequired' })
          .min(6, 'validation.passwordTooShort'),
    active: z.boolean().default(true)
  })
)

// Store active state separately since it's not in the schema validation
const active = ref(true)

const { values, errors, resetForm, setValues, handleSubmit, hasError } = useFormValidation(
  userSchema.value
)

// Re-initialize form validation when schema changes (edit mode toggle)
watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      if (props.user) {
        setValues({
          name: props.user.name,
          email: props.user.email,
          password: ''
        })
        active.value = props.user.active
      } else {
        resetForm({
          values: {
            name: '',
            email: '',
            password: ''
          }
        })
        active.value = true
      }
    }
  }
)

const onSubmit = handleSubmit((formValues) => {
  const data: UserCreateRequest = {
    name: formValues.name,
    email: formValues.email,
    password: formValues.password || '',
    active: active.value
  }
  // Don't send empty password for updates
  if (isEditing.value && !formValues.password) {
    delete (data as Partial<UserCreateRequest>).password
  }
  emit('save', data)
})
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
            v-model="values.name"
            :class="{ 'p-invalid': hasError('name') }"
            :placeholder="t('common.name')"
          />
          <small v-if="errors.name" class="p-error">{{ errors.name }}</small>
        </div>

        <div class="field">
          <label for="email">{{ t('common.email') }}</label>
          <InputText
            id="email"
            v-model="values.email"
            type="email"
            :class="{ 'p-invalid': hasError('email') }"
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
            v-model="values.password"
            :class="{ 'p-invalid': hasError('password') }"
            :feedback="false"
            toggle-mask
            :placeholder="t('users.password')"
            :input-style="{ width: '100%' }"
          />
          <small v-if="errors.password" class="p-error">{{ errors.password }}</small>
        </div>

        <div class="field">
          <div class="flex align-items-center gap-2">
            <Checkbox v-model="active" input-id="active" :binary="true" />
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
