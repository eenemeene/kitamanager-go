<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { User, UserCreateRequest } from '@/api/types'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Checkbox from 'primevue/checkbox'
import Button from 'primevue/button'

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

const errors = ref<{ name?: string; email?: string; password?: string }>({})

const isEditing = computed(() => !!props.user)
const dialogTitle = computed(() => (isEditing.value ? 'Edit User' : 'New User'))

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
    errors.value.name = 'Name is required'
  }

  if (!form.value.email.trim()) {
    errors.value.email = 'Email is required'
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.value.email)) {
    errors.value.email = 'Invalid email format'
  }

  if (!isEditing.value && !form.value.password) {
    errors.value.password = 'Password is required'
  } else if (form.value.password && form.value.password.length < 6) {
    errors.value.password = 'Password must be at least 6 characters'
  }

  return Object.keys(errors.value).length === 0
}

function handleSave() {
  if (validate()) {
    const data: UserCreateRequest = {
      name: form.value.name,
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
    <div class="form-grid">
      <div class="field">
        <label for="name">Name</label>
        <InputText
          id="name"
          v-model="form.name"
          :class="{ 'p-invalid': errors.name }"
          placeholder="Full name"
        />
        <small v-if="errors.name" class="p-error">{{ errors.name }}</small>
      </div>

      <div class="field">
        <label for="email">Email</label>
        <InputText
          id="email"
          v-model="form.email"
          type="email"
          :class="{ 'p-invalid': errors.email }"
          placeholder="Email address"
        />
        <small v-if="errors.email" class="p-error">{{ errors.email }}</small>
      </div>

      <div class="field">
        <label for="password">Password {{ isEditing ? '(leave blank to keep)' : '' }}</label>
        <Password
          id="password"
          v-model="form.password"
          :class="{ 'p-invalid': errors.password }"
          :feedback="false"
          toggle-mask
          placeholder="Password"
          :input-style="{ width: '100%' }"
        />
        <small v-if="errors.password" class="p-error">{{ errors.password }}</small>
      </div>

      <div class="field">
        <div class="flex align-items-center gap-2">
          <Checkbox v-model="form.active" input-id="active" :binary="true" />
          <label for="active">Active</label>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <Button label="Cancel" text @click="$emit('close')" />
        <Button label="Save" @click="handleSave" />
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
</style>
