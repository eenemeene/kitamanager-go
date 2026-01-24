<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useCrud } from '@/composables/useCrud'
import { apiClient } from '@/api/client'
import type { User, UserCreate, UserUpdate } from '@/api/types'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import UserForm from './UserForm.vue'
import UserMembershipsDialog from './UserMembershipsDialog.vue'

const {
  items: users,
  loading,
  dialogVisible,
  editingItem,
  fetchItems,
  openCreateDialog,
  openEditDialog,
  closeDialog,
  saveItem,
  confirmDelete
} = useCrud<User, UserCreate, UserUpdate>({
  entityName: 'User',
  fetchAll: () => apiClient.getUsers(),
  create: (data) => apiClient.createUser(data),
  update: (id, data) => apiClient.updateUser(id, data),
  remove: (id) => apiClient.deleteUser(id)
})

const membershipsDialogVisible = ref(false)
const selectedUserForMemberships = ref<User | null>(null)

function openMembershipsDialog(user: User) {
  selectedUserForMemberships.value = user
  membershipsDialogVisible.value = true
}

function closeMembershipsDialog() {
  membershipsDialogVisible.value = false
  selectedUserForMemberships.value = null
}

onMounted(() => {
  fetchItems()
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1>Users</h1>
      <Button label="New User" icon="pi pi-plus" @click="openCreateDialog" />
    </div>

    <div class="card">
      <DataTable
        :value="users"
        :loading="loading"
        striped-rows
        paginator
        :rows="10"
        :rows-per-page-options="[10, 25, 50]"
      >
        <Column field="id" header="ID" sortable style="width: 80px"></Column>
        <Column field="name" header="Name" sortable></Column>
        <Column field="email" header="Email" sortable></Column>
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
        <Column header="Actions" style="width: 200px">
          <template #body="{ data }">
            <Button
              icon="pi pi-users"
              text
              rounded
              title="Manage Memberships"
              @click="openMembershipsDialog(data)"
            />
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

    <UserForm :visible="dialogVisible" :user="editingItem" @close="closeDialog" @save="saveItem" />

    <UserMembershipsDialog
      :visible="membershipsDialogVisible"
      :user="selectedUserForMemberships"
      @close="closeMembershipsDialog"
      @updated="fetchItems"
    />
  </div>
</template>
