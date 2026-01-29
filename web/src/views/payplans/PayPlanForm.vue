<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PayPlan } from '@/api/types'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'

const props = defineProps<{
  visible: boolean
  payPlan: PayPlan | null
}>()

const emit = defineEmits<{
  save: [data: { name: string }]
  close: []
}>()

const { t } = useI18n()

const form = ref({
  name: ''
})

const errors = ref<{ name?: string }>({})

const isEditing = computed(() => !!props.payPlan)

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      if (props.payPlan) {
        form.value = {
          name: props.payPlan.name
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
    errors.value.name = t('validation.nameRequired')
  }
  return Object.keys(errors.value).length === 0
}

function onSubmit() {
  if (validate()) {
    emit('save', { name: form.value.name })
  }
}
</script>

<template>
  <Dialog
    :visible="visible"
    :header="isEditing ? t('payPlans.edit') : t('payPlans.create')"
    modal
    :style="{ width: '450px' }"
    @update:visible="$emit('close')"
  >
    <form @submit.prevent="onSubmit" data-testid="payplan-form">
      <div class="form-grid">
        <div class="field">
          <label for="name">{{ t('common.name') }}</label>
          <InputText
            id="name"
            v-model="form.name"
            :class="{ 'p-invalid': errors.name }"
            data-testid="name-input"
          />
          <small v-if="errors.name" class="p-error">{{ errors.name }}</small>
        </div>
      </div>
    </form>

    <template #footer>
      <Button :label="t('common.cancel')" text @click="$emit('close')" data-testid="cancel-btn" />
      <Button :label="t('common.save')" @click="onSubmit" data-testid="save-btn" />
    </template>
  </Dialog>
</template>

<style scoped>
.form-grid {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.field label {
  font-weight: 600;
}

.field :deep(input) {
  width: 100%;
}

.p-error {
  color: var(--red-500);
}
</style>
