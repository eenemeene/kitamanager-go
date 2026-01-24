<script setup lang="ts">
import { ref, watch, computed, onMounted } from 'vue'
import { apiClient } from '@/api/client'
import { useUiStore } from '@/stores/ui'
import type { Group, GroupCreate, Organization } from '@/api/types'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Dropdown from 'primevue/dropdown'
import Checkbox from 'primevue/checkbox'
import Button from 'primevue/button'

const props = defineProps<{
  visible: boolean
  group: Group | null
}>()

const emit = defineEmits<{
  close: []
  save: [data: GroupCreate]
}>()

const uiStore = useUiStore()
const organizations = ref<Organization[]>([])

const form = ref({
  name: '',
  organization_id: 0,
  active: true
})

const errors = ref<{ name?: string; organization_id?: string }>({})

const isEditing = computed(() => !!props.group)
const dialogTitle = computed(() => (isEditing.value ? 'Edit Group' : 'New Group'))

// Load organizations for the dropdown
onMounted(async () => {
  try {
    organizations.value = await apiClient.getOrganizations()
  } catch {
    // Ignore errors, dropdown will be empty
  }
})

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      if (props.group) {
        form.value = {
          name: props.group.name,
          organization_id: props.group.organization_id,
          active: props.group.active
        }
      } else {
        form.value = {
          name: '',
          organization_id: uiStore.selectedOrganizationId || 0,
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
  if (!isEditing.value && !form.value.organization_id) {
    errors.value.organization_id = 'Organization is required'
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
          placeholder="Group name"
        />
        <small v-if="errors.name" class="p-error">{{ errors.name }}</small>
      </div>

      <div class="field" v-if="!isEditing">
        <label for="organization">Organization</label>
        <Dropdown
          id="organization"
          v-model="form.organization_id"
          :options="organizations"
          option-label="name"
          option-value="id"
          placeholder="Select Organization"
          :class="{ 'p-invalid': errors.organization_id }"
          class="w-full"
        />
        <small v-if="errors.organization_id" class="p-error">{{ errors.organization_id }}</small>
      </div>

      <div class="field" v-if="isEditing">
        <span class="field-label">Organization</span>
        <p id="organization-display" class="text-muted">
          {{ organizations.find((o) => o.id === form.organization_id)?.name || 'Unknown' }}
        </p>
        <small class="text-muted">Organization cannot be changed after creation.</small>
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

.w-full {
  width: 100%;
}

.text-muted {
  color: var(--text-color-secondary);
  margin: 0;
}

.field-label {
  display: block;
  font-weight: 500;
  margin-bottom: 0.5rem;
}
</style>
