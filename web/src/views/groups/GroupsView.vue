<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useCrud } from '@/composables/useCrud'
import { apiClient } from '@/api/client'
import type { Group, GroupCreate, GroupUpdate, Organization } from '@/api/types'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import GroupForm from './GroupForm.vue'

const organizations = ref<Map<number, Organization>>(new Map())

const {
  items: groups,
  loading,
  dialogVisible,
  editingItem,
  fetchItems,
  openCreateDialog,
  openEditDialog,
  closeDialog,
  saveItem,
  confirmDelete
} = useCrud<Group, GroupCreate, GroupUpdate>({
  entityName: 'Group',
  fetchAll: () => apiClient.getGroups(),
  create: (data) => apiClient.createGroup(data),
  update: (id, data) => apiClient.updateGroup(id, data),
  remove: (id) => apiClient.deleteGroup(id)
})

function getOrganizationName(orgId: number): string {
  return organizations.value.get(orgId)?.name || 'Unknown'
}

async function loadOrganizations() {
  try {
    const orgs = await apiClient.getOrganizations()
    organizations.value = new Map(orgs.map((o) => [o.id, o]))
  } catch {
    // Ignore errors
  }
}

onMounted(async () => {
  await loadOrganizations()
  fetchItems()
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1>Groups</h1>
      <Button label="New Group" icon="pi pi-plus" @click="openCreateDialog" />
    </div>

    <div class="card">
      <DataTable
        :value="groups"
        :loading="loading"
        striped-rows
        paginator
        :rows="10"
        :rows-per-page-options="[10, 25, 50]"
      >
        <Column field="id" header="ID" sortable style="width: 80px"></Column>
        <Column field="name" header="Name" sortable></Column>
        <Column field="organization_id" header="Organization" sortable style="width: 200px">
          <template #body="{ data }">
            {{ getOrganizationName(data.organization_id) }}
          </template>
        </Column>
        <Column field="active" header="Status" sortable style="width: 120px">
          <template #body="{ data }">
            <Tag
              :value="data.active ? 'Active' : 'Inactive'"
              :severity="data.active ? 'success' : 'danger'"
            />
          </template>
        </Column>
        <Column field="created_at" header="Created" sortable style="width: 180px">
          <template #body="{ data }">
            {{ new Date(data.created_at).toLocaleDateString() }}
          </template>
        </Column>
        <Column header="Actions" style="width: 150px">
          <template #body="{ data }">
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

    <GroupForm
      :visible="dialogVisible"
      :group="editingItem"
      @close="closeDialog"
      @save="saveItem"
    />
  </div>
</template>
