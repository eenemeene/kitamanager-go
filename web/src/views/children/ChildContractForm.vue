<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { Child, ChildContract, ChildContractCreateRequest } from '@/api/types'
import Dialog from 'primevue/dialog'
import DatePicker from 'primevue/datepicker'
import Button from 'primevue/button'
import Chips from 'primevue/chips'
import Checkbox from 'primevue/checkbox'
import Message from 'primevue/message'

const props = defineProps<{
  visible: boolean
  child: Child | null
  currentContract: ChildContract | null
}>()

const emit = defineEmits<{
  close: []
  save: [data: ChildContractCreateRequest, endCurrentContract: boolean]
}>()

const form = ref({
  from: null as Date | null,
  to: null as Date | null,
  attributes: [] as string[]
})

const endCurrentContract = ref(true)

const errors = ref<{
  from?: string
}>({})

const dialogTitle = computed(() =>
  props.child
    ? `New Contract for ${props.child.first_name} ${props.child.last_name}`
    : 'New Contract'
)

const hasActiveContract = computed(() => props.currentContract !== null)

const currentContractInfo = computed(() => {
  if (!props.currentContract) return ''
  const from = new Date(props.currentContract.from).toLocaleDateString()
  const attrs = props.currentContract.attributes?.join(', ') || 'no attributes'
  return `Active since ${from} (${attrs})`
})

const suggestedEndDate = computed(() => {
  if (!form.value.from) return null
  const endDate = new Date(form.value.from)
  endDate.setDate(endDate.getDate() - 1)
  return endDate.toLocaleDateString()
})

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      form.value = {
        from: new Date(),
        to: null,
        attributes: []
      }
      endCurrentContract.value = true
      errors.value = {}
    }
  }
)

function validate(): boolean {
  errors.value = {}

  if (!form.value.from) {
    errors.value.from = 'Start date is required'
  }

  return Object.keys(errors.value).length === 0
}

function handleSave() {
  if (validate()) {
    emit(
      'save',
      {
        from: form.value.from!.toISOString(),
        to: form.value.to ? form.value.to.toISOString() : null,
        attributes: form.value.attributes
      },
      hasActiveContract.value && endCurrentContract.value
    )
  }
}
</script>

<template>
  <Dialog
    :visible="visible"
    :header="dialogTitle"
    modal
    :closable="true"
    :style="{ width: '550px' }"
    @update:visible="$emit('close')"
  >
    <div class="form-grid">
      <!-- Active contract warning -->
      <Message v-if="hasActiveContract" severity="warn" :closable="false" class="mb-3">
        <div class="active-contract-info">
          <p class="mb-2">
            <strong>This child has an active contract:</strong><br />
            {{ currentContractInfo }}
          </p>
          <div class="flex items-center gap-2">
            <Checkbox v-model="endCurrentContract" input-id="endContract" :binary="true" />
            <label for="endContract">
              End current contract on {{ suggestedEndDate }} (day before new contract starts)
            </label>
          </div>
        </div>
      </Message>

      <div class="field">
        <label for="from">Start Date</label>
        <DatePicker
          id="from"
          v-model="form.from"
          date-format="dd.mm.yy"
          :class="{ 'p-invalid': errors.from }"
          placeholder="Contract start date"
          show-icon
        />
        <small v-if="errors.from" class="p-error">{{ errors.from }}</small>
      </div>

      <div class="field">
        <label for="to">End Date (optional)</label>
        <DatePicker
          id="to"
          v-model="form.to"
          date-format="dd.mm.yy"
          placeholder="Contract end date"
          show-icon
        />
      </div>

      <div class="field">
        <label for="attributes">Attributes (care type & extras)</label>
        <Chips
          id="attributes"
          v-model="form.attributes"
          placeholder="e.g. ganztags, ndh, integration_a"
        />
        <small class="text-secondary">
          Press Enter to add each attribute (e.g., ganztags, halbtags, teilzeit, ndh, integration_a)
        </small>
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
.text-secondary {
  color: var(--text-color-secondary);
}

.active-contract-info p {
  margin: 0;
}

.mb-2 {
  margin-bottom: 0.5rem;
}

.mb-3 {
  margin-bottom: 1rem;
}

.flex {
  display: flex;
}

.items-center {
  align-items: center;
}

.gap-2 {
  gap: 0.5rem;
}
</style>
