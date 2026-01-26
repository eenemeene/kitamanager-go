<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useCrud } from '@/composables/useCrud'
import { apiClient } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import type { User, UserCreateRequest, UserUpdateRequest, Group } from '@/api/types'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import UserForm from './UserForm.vue'
import UserMembershipsDialog from './UserMembershipsDialog.vue'

const props = defineProps<{
  orgId: string
}>()

const orgIdNum = computed(() => parseInt(props.orgId, 10))

const authStore = useAuthStore()
const toast = useToast()
const confirm = useConfirm()

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
} = useCrud<User, UserCreateRequest, UserUpdateRequest>({
  entityName: 'User',
  fetchAll: () => apiClient.getOrganizationUsers(orgIdNum.value),
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

// Get groups to display - filter to current org
function getGroupsToDisplay(userGroups: Group[] | undefined): Group[] {
  if (!userGroups) return []
  return userGroups.filter((group) => group.organization_id === orgIdNum.value)
}

// Watch for org changes and refetch
watch(orgIdNum, () => {
  fetchItems()
})

function formatLastLogin(lastLogin: string | null | undefined): string {
  if (!lastLogin) return 'Never'
  return new Date(lastLogin).toLocaleString()
}

// Toggle superadmin status
function confirmToggleSuperadmin(user: User) {
  const action = user.is_superadmin ? 'revoke superadmin from' : 'grant superadmin to'
  confirm.require({
    message: `Are you sure you want to ${action} ${user.name}?`,
    header: 'Confirm Superadmin Change',
    icon: 'pi pi-exclamation-triangle',
    acceptClass: user.is_superadmin ? 'p-button-danger' : 'p-button-success',
    accept: async () => {
      try {
        await apiClient.setSuperAdmin(user.id, !user.is_superadmin)
        toast.add({
          severity: 'success',
          summary: 'Success',
          detail: `Superadmin status updated for ${user.name}`,
          life: 3000
        })
        await fetchItems()
      } catch {
        toast.add({
          severity: 'error',
          summary: 'Error',
          detail: 'Failed to update superadmin status',
          life: 3000
        })
      }
    }
  })
}

// Check if current user can modify superadmin status
function canModifySuperadmin(user: User): boolean {
  // Only superadmins can modify superadmin status
  // And you can't modify your own superadmin status
  return authStore.user?.is_superadmin === true && authStore.user?.id !== user.id
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
        <Column field="name" header="Name" sortable>
          <template #body="{ data }">
            <span>{{ data.name }}</span>
            <Tag v-if="data.is_superadmin" value="Superadmin" severity="warn" class="ml-2" />
          </template>
        </Column>
        <Column field="email" header="Email" sortable></Column>
        <Column field="active" header="Status" sortable style="width: 120px">
          <template #body="{ data }">
            <Tag
              :value="data.active ? 'Active' : 'Inactive'"
              :severity="data.active ? 'success' : 'danger'"
            />
          </template>
        </Column>
        <Column field="groups" header="Groups" style="width: 200px">
          <template #body="{ data }">
            <div class="group-tags">
              <Tag
                v-for="group in getGroupsToDisplay(data.groups)"
                :key="group.id"
                :value="group.name"
                severity="info"
                class="mr-1"
              />
              <span v-if="getGroupsToDisplay(data.groups).length === 0" class="text-muted">
                -
              </span>
            </div>
          </template>
        </Column>
        <Column field="last_login" header="Last Login" sortable style="width: 180px">
          <template #body="{ data }">
            {{ formatLastLogin(data.last_login) }}
          </template>
        </Column>
        <Column field="created_at" header="Created" sortable style="width: 150px">
          <template #body="{ data }">
            {{ new Date(data.created_at).toLocaleDateString() }}
          </template>
        </Column>
        <Column header="Actions" style="width: 220px">
          <template #body="{ data }">
            <Button
              v-if="canModifySuperadmin(data)"
              :icon="data.is_superadmin ? 'pi pi-star-fill' : 'pi pi-star'"
              text
              rounded
              :severity="data.is_superadmin ? 'warn' : 'secondary'"
              :title="data.is_superadmin ? 'Revoke Superadmin' : 'Grant Superadmin'"
              @click="confirmToggleSuperadmin(data)"
            />
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

<style scoped>
.group-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
}

.text-muted {
  color: var(--text-color-secondary);
}

.mr-1 {
  margin-right: 0.25rem;
}

.ml-2 {
  margin-left: 0.5rem;
}
</style>
