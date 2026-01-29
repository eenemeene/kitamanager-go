<script setup lang="ts" generic="T extends ContractWithPeriod">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatDate } from '@/utils/formatting'
import { useContractStatus, type ContractWithPeriod } from '@/composables/useContractStatus'
import Dialog from 'primevue/dialog'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import ProgressSpinner from 'primevue/progressspinner'

const { t } = useI18n()
const { getContractStatus, getStatusLabel, getStatusSeverity, getRowClass, sortByDateDesc } =
  useContractStatus()

/**
 * Person interface - requires first_name and last_name for display.
 */
interface Person {
  id: number
  first_name: string
  last_name: string
}

const props = defineProps<{
  visible: boolean
  person: Person | null
  titleKey: string // i18n key for the title (e.g., 'employees.contractHistory')
  noContractsKey: string // i18n key for empty state (e.g., 'employees.noContractsFound')
  fetchContracts: (personId: number) => Promise<T[]>
  dialogWidth?: string
}>()

defineEmits<{
  close: []
}>()

const contracts = ref<T[]>([]) as { value: T[] }
const loading = ref(false)
const error = ref<string | null>(null)

const dialogTitle = computed(() =>
  props.person
    ? `${t(props.titleKey)}: ${props.person.first_name} ${props.person.last_name}`
    : t(props.titleKey)
)

const sortedContracts = computed(() => sortByDateDesc(contracts.value))

watch(
  () => props.visible,
  async (visible) => {
    if (visible && props.person) {
      await loadContracts()
    } else {
      contracts.value = []
      error.value = null
    }
  }
)

async function loadContracts() {
  if (!props.person) return

  loading.value = true
  error.value = null
  try {
    contracts.value = await props.fetchContracts(props.person.id)
  } catch {
    error.value = t('common.failedToLoad', { resource: t('contracts.title') })
    contracts.value = []
  } finally {
    loading.value = false
  }
}

function formatContractDate(dateStr: string | null | undefined): string {
  return formatDate(dateStr, 'de-DE', t('common.ongoing'))
}
</script>

<template>
  <Dialog
    :visible="visible"
    :header="dialogTitle"
    modal
    :closable="true"
    :style="{ width: dialogWidth || '800px' }"
    @update:visible="$emit('close')"
  >
    <div v-if="loading" class="loading-container">
      <ProgressSpinner />
    </div>

    <div v-else-if="error" class="error-message">
      {{ error }}
    </div>

    <div v-else-if="sortedContracts.length === 0" class="no-contracts">
      {{ t(noContractsKey) }}
    </div>

    <DataTable v-else :value="sortedContracts" striped-rows :row-class="getRowClass">
      <Column :header="t('common.status')" style="width: 100px">
        <template #body="{ data }">
          <Tag
            :value="getStatusLabel(getContractStatus(data))"
            :severity="getStatusSeverity(getContractStatus(data))"
          />
        </template>
      </Column>
      <Column :header="t('contracts.from')" style="width: 110px">
        <template #body="{ data }">
          {{ formatContractDate(data.from) }}
        </template>
      </Column>
      <Column :header="t('contracts.to')" style="width: 110px">
        <template #body="{ data }">
          {{ formatContractDate(data.to) }}
        </template>
      </Column>
      <!-- Custom columns slot -->
      <slot name="columns" :contracts="sortedContracts"></slot>
    </DataTable>

    <template #footer>
      <div class="dialog-footer">
        <Button :label="t('common.close')" @click="$emit('close')" />
      </div>
    </template>
  </Dialog>
</template>

<style scoped>
.loading-container {
  display: flex;
  justify-content: center;
  padding: 2rem;
}

.error-message {
  padding: 2rem;
  text-align: center;
  color: var(--red-500);
}

.no-contracts {
  padding: 2rem;
  text-align: center;
  color: var(--text-color-secondary);
}

:deep(.active-contract-row) {
  background-color: var(--green-50) !important;
  font-weight: 500;
}

:deep(.p-datatable-dark .active-contract-row) {
  background-color: var(--green-900) !important;
}
</style>
