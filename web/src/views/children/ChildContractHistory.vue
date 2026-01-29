<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'
import type { Child, ChildContract } from '@/api/types'
import ContractHistory from '@/components/ContractHistory.vue'
import Column from 'primevue/column'
import Tag from 'primevue/tag'

const { t } = useI18n()

const props = defineProps<{
  visible: boolean
  child: Child | null
  orgId: number
}>()

defineEmits<{
  close: []
}>()

async function fetchContracts(childId: number): Promise<ChildContract[]> {
  return apiClient.getChildContracts(props.orgId, childId)
}
</script>

<template>
  <ContractHistory
    :visible="visible"
    :person="child"
    title-key="children.contractHistory"
    no-contracts-key="children.noContractsFound"
    :fetch-contracts="fetchContracts"
    dialog-width="700px"
    @close="$emit('close')"
  >
    <template #columns>
      <Column :header="t('children.attributes')">
        <template #body="{ data }">
          <template v-if="data.attributes?.length">
            <Tag v-for="attr in data.attributes" :key="attr" :value="attr" class="mr-1" />
          </template>
          <span v-else class="text-secondary">-</span>
        </template>
      </Column>
    </template>
  </ContractHistory>
</template>

<style scoped>
.text-secondary {
  color: var(--text-color-secondary);
}

.mr-1 {
  margin-right: 0.25rem;
}
</style>
