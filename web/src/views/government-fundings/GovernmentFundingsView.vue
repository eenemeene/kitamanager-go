<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useCrud } from '@/composables/useCrud'
import { apiClient } from '@/api/client'
import type {
  GovernmentFunding,
  GovernmentFundingCreateRequest,
  GovernmentFundingUpdateRequest
} from '@/api/types'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import GovernmentFundingForm from './GovernmentFundingForm.vue'

const router = useRouter()

const {
  items: governmentFundings,
  loading,
  dialogVisible,
  editingItem,
  fetchItems,
  openCreateDialog,
  openEditDialog,
  closeDialog,
  saveItem,
  confirmDelete
} = useCrud<GovernmentFunding, GovernmentFundingCreateRequest, GovernmentFundingUpdateRequest>({
  entityName: 'Government Funding',
  fetchAll: () => apiClient.getGovernmentFundings(),
  create: (data) => apiClient.createGovernmentFunding(data),
  update: (id, data) => apiClient.updateGovernmentFunding(id, data),
  remove: (id) => apiClient.deleteGovernmentFunding(id)
})

function openDetails(governmentFunding: GovernmentFunding) {
  router.push({ name: 'government-funding-detail', params: { id: governmentFunding.id } })
}

onMounted(() => {
  fetchItems()
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1>Government Fundings</h1>
      <Button label="New Government Funding" icon="pi pi-plus" @click="openCreateDialog" />
    </div>

    <div class="card">
      <DataTable
        :value="governmentFundings"
        :loading="loading"
        striped-rows
        paginator
        :rows="10"
        :rows-per-page-options="[10, 25, 50]"
      >
        <Column field="id" header="ID" sortable style="width: 80px"></Column>
        <Column field="name" header="Name" sortable></Column>
        <Column field="created_at" header="Created" sortable style="width: 180px">
          <template #body="{ data }">
            {{ new Date(data.created_at).toLocaleDateString() }}
          </template>
        </Column>
        <Column header="Actions" style="width: 200px">
          <template #body="{ data }">
            <Button icon="pi pi-eye" text rounded title="View Details" @click="openDetails(data)" />
            <Button icon="pi pi-pencil" text rounded title="Edit" @click="openEditDialog(data)" />
            <Button
              icon="pi pi-trash"
              text
              rounded
              severity="danger"
              title="Delete"
              @click="confirmDelete(data)"
            />
          </template>
        </Column>
      </DataTable>
    </div>

    <GovernmentFundingForm
      :visible="dialogVisible"
      :government-funding="editingItem"
      @close="closeDialog"
      @save="saveItem"
    />
  </div>
</template>
