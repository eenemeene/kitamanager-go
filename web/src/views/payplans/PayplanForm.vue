<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { Payplan, PayplanCreateRequest } from '@/api/types'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'

const props = defineProps<{
  visible: boolean
  payplan: Payplan | null
}>()

const emit = defineEmits<{
  close: []
  save: [data: PayplanCreateRequest]
}>()

const form = ref({
  name: ''
})

const errors = ref<{ name?: string }>({})

const isEditing = computed(() => !!props.payplan)
const dialogTitle = computed(() => (isEditing.value ? 'Edit Payplan' : 'New Payplan'))

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      if (props.payplan) {
        form.value = {
          name: props.payplan.name
        }
      } else {
        form.value = {
          name: ''
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
          placeholder="Payplan name (e.g. Berlin)"
        />
        <small v-if="errors.name" class="p-error">{{ errors.name }}</small>
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
