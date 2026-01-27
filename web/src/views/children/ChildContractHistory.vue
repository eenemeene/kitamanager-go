<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { apiClient } from '@/api/client'
import type { Child, ChildContract } from '@/api/types'
import Dialog from 'primevue/dialog'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import ProgressSpinner from 'primevue/progressspinner'

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
    ? `Contract History: ${props.child.first_name} ${props.child.last_name}`
    : 'Contract History'
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
    error.value = 'Failed to load contracts'
    contracts.value = []
  } finally {
    loading.value = false
  }
}

function formatDate(dateStr: string | null | undefined): string {
  if (!dateStr) return 'ongoing'
  return new Date(dateStr).toLocaleDateString()
}

function isCurrentContract(contract: ChildContract): boolean {
  const now = new Date()
  const from = new Date(contract.from)
  const to = contract.to ? new Date(contract.to) : null
  return from <= now && (!to || to >= now)
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
      No contracts found for this child.
    </div>

    <DataTable v-else :value="sortedContracts" striped-rows>
      <Column header="Status" style="width: 100px">
        <template #body="{ data }">
          <Tag
            :value="isCurrentContract(data) ? 'Active' : 'Inactive'"
            :severity="isCurrentContract(data) ? 'success' : 'secondary'"
          />
        </template>
      </Column>
      <Column header="From" style="width: 120px">
        <template #body="{ data }">
          {{ formatDate(data.from) }}
        </template>
      </Column>
      <Column header="To" style="width: 120px">
        <template #body="{ data }">
          {{ formatDate(data.to) }}
        </template>
      </Column>
      <Column header="Attributes">
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
        <Button label="Close" @click="$emit('close')" />
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
</style>
