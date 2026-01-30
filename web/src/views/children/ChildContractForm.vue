<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Child, ChildContract, ChildContractCreateRequest } from '@/api/types'
import Dialog from 'primevue/dialog'
import DatePicker from 'primevue/datepicker'
import Button from 'primevue/button'
import InputChips from 'primevue/inputchips'
import Checkbox from 'primevue/checkbox'
import Message from 'primevue/message'

const { t } = useI18n()

const props = defineProps<{
  visible: boolean
  child: Child | null
  currentContract: ChildContract | null
}>()

const emit = defineEmits<{
  close: []
  save: [data: ChildContractCreateRequest, endCurrentContract: boolean]
}>()

// Form key - changes to force form reset when dialog opens
const formKey = ref(0)

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
    ? t('contracts.newContractFor', { name: `${props.child.first_name} ${props.child.last_name}` })
    : t('contracts.newContract')
)

const hasActiveContract = computed(() => props.currentContract !== null)

const currentContractInfo = computed(() => {
  if (!props.currentContract) return ''
  const from = new Date(props.currentContract.from).toLocaleDateString()
  const attrs = props.currentContract.attributes?.join(', ') || t('contracts.noAttributes')
  return t('contracts.activeSince', { date: from, attrs })
})

const suggestedEndDate = computed(() => {
  if (!form.value.from) return null
  const endDate = new Date(form.value.from)
  endDate.setDate(endDate.getDate() - 1)
  return endDate.toLocaleDateString()
})

// Track if we've initialized the form for this dialog session
const isInitialized = ref(false)

// Reset initialization flag when dialog closes
watch(
  () => props.visible,
  (visible) => {
    if (!visible) {
      isInitialized.value = false
    }
  }
)

// Initialize form when dialog opens AND currentContract is available
// This handles the case where props might update in different ticks
watch(
  [() => props.visible, () => props.currentContract],
  async ([visible, currentContract]) => {
    // Initialize when dialog is visible and we haven't initialized yet
    if (visible && !isInitialized.value) {
      // Mark as initialized to prevent re-initialization
      isInitialized.value = true

      // Prefill attributes from current contract
      const prefillAttrs = currentContract?.attributes || []

      form.value = {
        from: new Date(),
        to: null,
        attributes: [...prefillAttrs]
      }
      endCurrentContract.value = true
      errors.value = {}

      // Wait for Vue to process the form update, then force InputChips re-render
      await nextTick()
      formKey.value++
    }
  },
  { immediate: true }
)

function validate(): boolean {
  errors.value = {}

  if (!form.value.from) {
    errors.value.from = t('contracts.startDateRequired')
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
            <strong>{{ t('contracts.hasActiveContract') }}</strong
            ><br />
            {{ currentContractInfo }}
          </p>
          <div class="flex items-center gap-2">
            <Checkbox v-model="endCurrentContract" input-id="endContract" :binary="true" />
            <label for="endContract">
              {{ t('contracts.endCurrentContract', { date: suggestedEndDate }) }}
            </label>
          </div>
        </div>
      </Message>

      <div class="field">
        <label for="from">{{ t('contracts.startDate') }}</label>
        <DatePicker
          id="from"
          v-model="form.from"
          date-format="dd.mm.yy"
          :class="{ 'p-invalid': errors.from }"
          :placeholder="t('contracts.contractStartPlaceholder')"
          show-icon
        />
        <small v-if="errors.from" class="p-error">{{ errors.from }}</small>
      </div>

      <div class="field">
        <label for="to">{{ t('contracts.endDateOptional') }}</label>
        <DatePicker
          id="to"
          v-model="form.to"
          date-format="dd.mm.yy"
          :placeholder="t('contracts.contractEndPlaceholder')"
          show-icon
        />
      </div>

      <div class="field">
        <label for="attributes">{{ t('contracts.attributesLabel') }}</label>
        <InputChips
          :key="formKey"
          id="attributes"
          v-model="form.attributes"
          placeholder="e.g. ganztags, ndh, integration_a"
        />
        <small class="text-secondary">
          {{ t('contracts.attributesHelp') }}
        </small>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <Button :label="t('common.cancel')" text @click="$emit('close')" />
        <Button :label="t('common.save')" @click="handleSave" />
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
