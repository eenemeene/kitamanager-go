<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'
import type { Employee, EmployeeContract } from '@/api/types'
import ContractHistory from '@/components/ContractHistory.vue'
import Column from 'primevue/column'

const { t } = useI18n()

const props = defineProps<{
  visible: boolean
  employee: Employee | null
  orgId: number
}>()

defineEmits<{
  close: []
}>()

async function fetchContracts(employeeId: number): Promise<EmployeeContract[]> {
  return apiClient.getEmployeeContracts(props.orgId, employeeId)
}
</script>

<template>
  <ContractHistory
    :visible="visible"
    :person="employee"
    title-key="employees.contractHistory"
    no-contracts-key="employees.noContractsFound"
    :fetch-contracts="fetchContracts"
    dialog-width="800px"
    @close="$emit('close')"
  >
    <template #columns>
      <Column :header="t('employees.position')" style="width: 140px">
        <template #body="{ data }">
          {{ data.position || '-' }}
        </template>
      </Column>
      <Column :header="t('employees.grade')" style="width: 80px">
        <template #body="{ data }">
          {{ data.grade || '-' }}
        </template>
      </Column>
      <Column :header="t('employees.step')" style="width: 60px">
        <template #body="{ data }">
          {{ data.step || '-' }}
        </template>
      </Column>
      <Column :header="t('employees.weeklyHours')" style="width: 100px">
        <template #body="{ data }">
          {{ data.weekly_hours || '-' }}
        </template>
      </Column>
    </template>
  </ContractHistory>
</template>
