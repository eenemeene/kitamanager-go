<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { Child, ChildCreate } from '@/api/types'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import DatePicker from 'primevue/datepicker'
import Button from 'primevue/button'

const props = defineProps<{
  visible: boolean
  child: Child | null
}>()

const emit = defineEmits<{
  close: []
  save: [data: Omit<ChildCreate, 'organization_id'>]
}>()

const form = ref({
  first_name: '',
  last_name: '',
  birthdate: null as Date | null
})

const errors = ref<{ first_name?: string; last_name?: string; birthdate?: string }>({})

const isEditing = computed(() => !!props.child)
const dialogTitle = computed(() => (isEditing.value ? 'Edit Child' : 'New Child'))

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      if (props.child) {
        form.value = {
          first_name: props.child.first_name,
          last_name: props.child.last_name,
          birthdate: new Date(props.child.birthdate)
        }
      } else {
        form.value = {
          first_name: '',
          last_name: '',
          birthdate: null
        }
      }
      errors.value = {}
    }
  }
)

function validate(): boolean {
  errors.value = {}

  if (!form.value.first_name.trim()) {
    errors.value.first_name = 'First name is required'
  }

  if (!form.value.last_name.trim()) {
    errors.value.last_name = 'Last name is required'
  }

  if (!form.value.birthdate) {
    errors.value.birthdate = 'Birthdate is required'
  }

  return Object.keys(errors.value).length === 0
}

function handleSave() {
  if (validate()) {
    emit('save', {
      first_name: form.value.first_name,
      last_name: form.value.last_name,
      birthdate: form.value.birthdate!.toISOString().split('T')[0]
    })
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
        <label for="first_name">First Name</label>
        <InputText
          id="first_name"
          v-model="form.first_name"
          :class="{ 'p-invalid': errors.first_name }"
          placeholder="First name"
        />
        <small v-if="errors.first_name" class="p-error">{{ errors.first_name }}</small>
      </div>

      <div class="field">
        <label for="last_name">Last Name</label>
        <InputText
          id="last_name"
          v-model="form.last_name"
          :class="{ 'p-invalid': errors.last_name }"
          placeholder="Last name"
        />
        <small v-if="errors.last_name" class="p-error">{{ errors.last_name }}</small>
      </div>

      <div class="field">
        <label for="birthdate">Birthdate</label>
        <DatePicker
          id="birthdate"
          v-model="form.birthdate"
          date-format="dd.mm.yy"
          :class="{ 'p-invalid': errors.birthdate }"
          placeholder="Select birthdate"
          show-icon
        />
        <small v-if="errors.birthdate" class="p-error">{{ errors.birthdate }}</small>
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
