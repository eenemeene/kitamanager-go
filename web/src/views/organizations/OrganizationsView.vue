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
  GovernmentFunding
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

// GovernmentFunding assignment
const governmentFundings = ref<GovernmentFunding[]>([])
const governmentFundingDialogVisible = ref(false)
const selectedOrg = ref<Organization | null>(null)
const selectedGovernmentFundingId = ref<number | null>(null)

async function fetchGovernmentFundings() {
  try {
    governmentFundings.value = await apiClient.getGovernmentFundings()
  } catch {
    // GovernmentFundings might not be available to non-superadmins
  }
}

function openGovernmentFundingDialog(org: Organization) {
  selectedOrg.value = org
  selectedGovernmentFundingId.value = org.government_funding_id || null
  governmentFundingDialogVisible.value = true
}

async function saveGovernmentFundingAssignment() {
  if (!selectedOrg.value) return

  try {
    if (selectedGovernmentFundingId.value) {
      await apiClient.assignGovernmentFundingToOrganization(
        selectedOrg.value.id,
        selectedGovernmentFundingId.value
      )
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'Government funding assigned successfully',
        life: 3000
      })
    } else {
      await apiClient.removeGovernmentFundingFromOrganization(selectedOrg.value.id)
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'Government funding removed successfully',
        life: 3000
      })
    }
    governmentFundingDialogVisible.value = false
    await fetchItems()
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to update government funding assignment',
      life: 3000
    })
  }
}

function getGovernmentFundingName(org: Organization): string {
  if (org.government_funding) return org.government_funding.name
  const plan = governmentFundings.value.find((p) => p.id === org.government_funding_id)
  return plan?.name || '-'
}

onMounted(() => {
  fetchItems()
  fetchGovernmentFundings()
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
        <Column header="Government Funding" style="width: 150px">
          <template #body="{ data }">
            <span>{{ getGovernmentFundingName(data) }}</span>
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
              title="Assign Government Funding"
              aria-label="Assign Government Funding"
              @click="openGovernmentFundingDialog(data)"
            />
            <Button
              icon="pi pi-pencil"
              text
              rounded
              title="Edit"
              aria-label="Edit"
              @click="openEditDialog(data)"
            />
            <Button
              icon="pi pi-trash"
              text
              rounded
              severity="danger"
              title="Delete"
              aria-label="Delete"
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

    <!-- GovernmentFunding Assignment Dialog -->
    <Dialog
      v-model:visible="governmentFundingDialogVisible"
      header="Assign Government Funding"
      modal
      :style="{ width: '400px' }"
    >
      <div class="form-grid">
        <div class="field">
          <span class="field-label">Organization</span>
          <p>{{ selectedOrg?.name }}</p>
        </div>
        <div class="field">
          <label for="government-funding">Government Funding</label>
          <Dropdown
            id="government-funding"
            v-model="selectedGovernmentFundingId"
            :options="governmentFundings"
            option-label="name"
            option-value="id"
            placeholder="Select a government funding"
            :show-clear="true"
            class="w-full"
          />
        </div>
      </div>
      <template #footer>
        <Button label="Cancel" text @click="governmentFundingDialogVisible = false" />
        <Button label="Save" @click="saveGovernmentFundingAssignment" />
      </template>
    </Dialog>
  </div>
</template>

<style scoped>
.w-full {
  width: 100%;
}
</style>
