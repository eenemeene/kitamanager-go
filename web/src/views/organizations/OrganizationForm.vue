<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { Organization, OrganizationCreateRequest } from '@/api/types'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Checkbox from 'primevue/checkbox'
import Button from 'primevue/button'

const props = defineProps<{
  visible: boolean
  organization: Organization | null
}>()

const emit = defineEmits<{
  close: []
  save: [data: OrganizationCreateRequest]
}>()

const form = ref({
  name: '',
  active: true
})

const errors = ref<{ name?: string }>({})

const isEditing = computed(() => !!props.organization)
const dialogTitle = computed(() => (isEditing.value ? 'Edit Organization' : 'New Organization'))

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      if (props.organization) {
        form.value = {
          name: props.organization.name,
          active: props.organization.active
        }
      } else {
        form.value = {
          name: '',
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
  return Object.keys(errors.value).length === 0
}

function handleSave() {
  if (validate()) {
    emit('save', form.value)
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
          placeholder="Organization name"
        />
        <small v-if="errors.name" class="p-error">{{ errors.name }}</small>
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
