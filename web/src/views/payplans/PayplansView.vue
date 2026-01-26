<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useCrud } from '@/composables/useCrud'
import { apiClient } from '@/api/client'
import type { Payplan, PayplanCreateRequest, PayplanUpdateRequest } from '@/api/types'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import PayplanForm from './PayplanForm.vue'

const router = useRouter()

const {
  items: payplans,
  loading,
  dialogVisible,
  editingItem,
  fetchItems,
  openCreateDialog,
  openEditDialog,
  closeDialog,
  saveItem,
  confirmDelete
} = useCrud<Payplan, PayplanCreateRequest, PayplanUpdateRequest>({
  entityName: 'Payplan',
  fetchAll: () => apiClient.getPayplans(),
  create: (data) => apiClient.createPayplan(data),
  update: (id, data) => apiClient.updatePayplan(id, data),
  remove: (id) => apiClient.deletePayplan(id)
})

function openDetails(payplan: Payplan) {
  router.push({ name: 'payplan-detail', params: { id: payplan.id } })
}

onMounted(() => {
  fetchItems()
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1>Payplans</h1>
      <Button label="New Payplan" icon="pi pi-plus" @click="openCreateDialog" />
    </div>

    <div class="card">
      <DataTable
        :value="payplans"
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

    <PayplanForm
      :visible="dialogVisible"
      :payplan="editingItem"
      @close="closeDialog"
      @save="saveItem"
    />
  </div>
</template>
