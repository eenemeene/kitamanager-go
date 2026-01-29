<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'
import { formatDate } from '@/utils/formatting'
import type { Child, ChildContract } from '@/api/types'
import Dialog from 'primevue/dialog'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import ProgressSpinner from 'primevue/progressspinner'

const { t } = useI18n()

const props = defineProps<{
  visible: boolean
  child: Child | null
  orgId: number
}>()

defineEmits<{
  close: []
}>()

const contracts = ref<ChildContract[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const dialogTitle = computed(() =>
  props.child
    ? `${t('children.contractHistory')}: ${props.child.first_name} ${props.child.last_name}`
    : t('children.contractHistory')
)

const sortedContracts = computed(() => {
  return [...contracts.value].sort((a, b) => {
    return new Date(b.from).getTime() - new Date(a.from).getTime()
  })
})

watch(
  () => props.visible,
  async (visible) => {
    if (visible && props.child) {
      await fetchContracts()
    } else {
      contracts.value = []
      error.value = null
    }
  }
)

async function fetchContracts() {
  if (!props.child) return

  loading.value = true
  error.value = null
  try {
    contracts.value = await apiClient.getChildContracts(props.orgId, props.child.id)
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

function getContractStatus(contract: ChildContract): 'active' | 'upcoming' | 'ended' {
  const now = new Date()
  now.setHours(0, 0, 0, 0) // Normalize to start of day for date comparison
  const from = new Date(contract.from)
  from.setHours(0, 0, 0, 0)
  const to = contract.to ? new Date(contract.to) : null
  if (to) {
    to.setHours(0, 0, 0, 0)
  }

  // Future contract (hasn't started yet)
  if (from > now) {
    return 'upcoming'
  }

  // Active contract (started and hasn't ended)
  if (from <= now && (!to || to >= now)) {
    return 'active'
  }

  // Ended contract
  return 'ended'
}

function getStatusLabel(status: 'active' | 'upcoming' | 'ended'): string {
  switch (status) {
    case 'active':
      return t('common.active')
    case 'upcoming':
      return t('common.upcoming')
    case 'ended':
      return t('common.ended')
  }
}

function getStatusSeverity(
  status: 'active' | 'upcoming' | 'ended'
): 'success' | 'info' | 'secondary' {
  switch (status) {
    case 'active':
      return 'success'
    case 'upcoming':
      return 'info'
    case 'ended':
      return 'secondary'
  }
}

function rowClass(data: ChildContract): string | undefined {
  return getContractStatus(data) === 'active' ? 'active-contract-row' : undefined
}
</script>

<template>
  <Dialog
    :visible="visible"
    :header="dialogTitle"
    modal
    :closable="true"
    :style="{ width: '700px' }"
    @update:visible="$emit('close')"
  >
    <div v-if="loading" class="loading-container">
      <ProgressSpinner />
    </div>

    <div v-else-if="error" class="error-message">
      {{ error }}
    </div>

    <div v-else-if="sortedContracts.length === 0" class="no-contracts">
      {{ t('children.noContractsFound') }}
    </div>

    <DataTable v-else :value="sortedContracts" striped-rows :row-class="rowClass">
      <Column :header="t('common.status')" style="width: 120px">
        <template #body="{ data }">
          <Tag
            :value="getStatusLabel(getContractStatus(data))"
            :severity="getStatusSeverity(getContractStatus(data))"
          />
        </template>
      </Column>
      <Column :header="t('contracts.from')" style="width: 120px">
        <template #body="{ data }">
          {{ formatContractDate(data.from) }}
        </template>
      </Column>
      <Column :header="t('contracts.to')" style="width: 120px">
        <template #body="{ data }">
          {{ formatContractDate(data.to) }}
        </template>
      </Column>
      <Column :header="t('children.attributes')">
        <template #body="{ data }">
          <template v-if="data.attributes?.length">
            <Tag v-for="attr in data.attributes" :key="attr" :value="attr" class="mr-1" />
          </template>
          <span v-else class="text-secondary">-</span>
        </template>
      </Column>
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

.text-secondary {
  color: var(--text-color-secondary);
}

.mr-1 {
  margin-right: 0.25rem;
}

:deep(.active-contract-row) {
  background-color: var(--green-50) !important;
  font-weight: 500;
}

:deep(.p-datatable-dark .active-contract-row) {
  background-color: var(--green-900) !important;
}
</style>
