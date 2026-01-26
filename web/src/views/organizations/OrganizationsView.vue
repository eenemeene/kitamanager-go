<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useCrud } from '@/composables/useCrud'
import { useToast } from 'primevue/usetoast'
import { useUiStore } from '@/stores/ui'
import { apiClient } from '@/api/client'
import type {
  Organization,
  OrganizationCreateRequest,
  OrganizationUpdateRequest,
  Payplan
} from '@/api/types'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import Dialog from 'primevue/dialog'
import Dropdown from 'primevue/dropdown'
import OrganizationForm from './OrganizationForm.vue'

const toast = useToast()
const uiStore = useUiStore()

const {
  items: organizations,
  loading,
  dialogVisible,
  editingItem,
  fetchItems,
  openCreateDialog,
  openEditDialog,
  closeDialog,
  saveItem: crudSaveItem,
  confirmDelete
} = useCrud<Organization, OrganizationCreateRequest, OrganizationUpdateRequest>({
  entityName: 'Organization',
  fetchAll: () => apiClient.getOrganizations(),
  create: (data) => apiClient.createOrganization(data),
  update: (id, data) => apiClient.updateOrganization(id, data),
  remove: (id) => apiClient.deleteOrganization(id)
})

// Wrap saveItem to also refresh the sidebar's org list
async function saveItem(data: OrganizationCreateRequest | OrganizationUpdateRequest) {
  await crudSaveItem(data)
  // Refresh sidebar organization dropdown
  await uiStore.fetchOrganizations()
}

// Payplan assignment
const payplans = ref<Payplan[]>([])
const payplanDialogVisible = ref(false)
const selectedOrg = ref<Organization | null>(null)
const selectedPayplanId = ref<number | null>(null)

async function fetchPayplans() {
  try {
    payplans.value = await apiClient.getPayplans()
  } catch {
    // Payplans might not be available to non-superadmins
  }
}

function openPayplanDialog(org: Organization) {
  selectedOrg.value = org
  selectedPayplanId.value = org.payplan_id || null
  payplanDialogVisible.value = true
}

async function savePayplanAssignment() {
  if (!selectedOrg.value) return

  try {
    if (selectedPayplanId.value) {
      await apiClient.assignPayplanToOrganization(selectedOrg.value.id, selectedPayplanId.value)
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'Payplan assigned successfully',
        life: 3000
      })
    } else {
      await apiClient.removePayplanFromOrganization(selectedOrg.value.id)
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'Payplan removed successfully',
        life: 3000
      })
    }
    payplanDialogVisible.value = false
    await fetchItems()
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to update payplan assignment',
      life: 3000
    })
  }
}

function getPayplanName(org: Organization): string {
  if (org.payplan) return org.payplan.name
  const plan = payplans.value.find((p) => p.id === org.payplan_id)
  return plan?.name || '-'
}

onMounted(() => {
  fetchItems()
  fetchPayplans()
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1>Organizations</h1>
      <Button label="New Organization" icon="pi pi-plus" @click="openCreateDialog" />
    </div>

    <div class="card">
      <DataTable
        :value="organizations"
        :loading="loading"
        striped-rows
        paginator
        :rows="10"
        :rows-per-page-options="[10, 25, 50]"
      >
        <Column field="id" header="ID" sortable style="width: 80px"></Column>
        <Column field="name" header="Name" sortable></Column>
        <Column header="Payplan" style="width: 150px">
          <template #body="{ data }">
            <span>{{ getPayplanName(data) }}</span>
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
        <Column header="Actions" style="width: 180px">
          <template #body="{ data }">
            <Button
              icon="pi pi-money-bill"
              text
              rounded
              title="Assign Payplan"
              @click="openPayplanDialog(data)"
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

    <OrganizationForm
      :visible="dialogVisible"
      :organization="editingItem"
      @close="closeDialog"
      @save="saveItem"
    />

    <!-- Payplan Assignment Dialog -->
    <Dialog
      v-model:visible="payplanDialogVisible"
      header="Assign Payplan"
      modal
      :style="{ width: '400px' }"
    >
      <div class="form-grid">
        <div class="field">
          <span class="field-label">Organization</span>
          <p>{{ selectedOrg?.name }}</p>
        </div>
        <div class="field">
          <label for="payplan">Payplan</label>
          <Dropdown
            id="payplan"
            v-model="selectedPayplanId"
            :options="payplans"
            option-label="name"
            option-value="id"
            placeholder="Select a payplan"
            :show-clear="true"
            class="w-full"
          />
        </div>
      </div>
      <template #footer>
        <Button label="Cancel" text @click="payplanDialogVisible = false" />
        <Button label="Save" @click="savePayplanAssignment" />
      </template>
    </Dialog>
  </div>
</template>

<style scoped>
.w-full {
  width: 100%;
}
</style>
