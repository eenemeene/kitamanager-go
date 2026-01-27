<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { apiClient } from '@/api/client'
import type {
  Child,
  ChildCreateRequest,
  ChildUpdateRequest,
  ChildContractCreateRequest
} from '@/api/types'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import ChildForm from './ChildForm.vue'
import ChildContractForm from './ChildContractForm.vue'
import ChildContractHistory from './ChildContractHistory.vue'

const route = useRoute()
const toast = useToast()
const confirm = useConfirm()

const orgId = ref(Number(route.params.orgId))
const children = ref<Child[]>([])
const loading = ref(false)

const dialogVisible = ref(false)
const editingChild = ref<Child | null>(null)

const contractDialogVisible = ref(false)
const selectedChild = ref<Child | null>(null)

const historyDialogVisible = ref(false)
const historyChild = ref<Child | null>(null)

watch(
  () => route.params.orgId,
  (newOrgId) => {
    orgId.value = Number(newOrgId)
    fetchChildren()
  }
)

async function fetchChildren() {
  loading.value = true
  try {
    children.value = await apiClient.getChildren(orgId.value)
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to load children',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editingChild.value = null
  dialogVisible.value = true
}

function openEditDialog(child: Child) {
  editingChild.value = child
  dialogVisible.value = true
}

function closeDialog() {
  dialogVisible.value = false
  editingChild.value = null
}

async function saveChild(data: ChildCreateRequest | ChildUpdateRequest) {
  try {
    if (editingChild.value) {
      await apiClient.updateChild(orgId.value, editingChild.value.id, data as ChildUpdateRequest)
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'Child updated successfully',
        life: 3000
      })
    } else {
      await apiClient.createChild(orgId.value, data as Omit<ChildCreateRequest, 'organization_id'>)
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'Child created successfully',
        life: 3000
      })
    }
    closeDialog()
    await fetchChildren()
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to save child',
      life: 3000
    })
  }
}

function confirmDelete(child: Child) {
  confirm.require({
    message: `Are you sure you want to delete ${child.first_name} ${child.last_name}?`,
    header: 'Confirm Delete',
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await apiClient.deleteChild(orgId.value, child.id)
        toast.add({
          severity: 'success',
          summary: 'Success',
          detail: 'Child deleted successfully',
          life: 3000
        })
        await fetchChildren()
      } catch {
        toast.add({
          severity: 'error',
          summary: 'Error',
          detail: 'Failed to delete child',
          life: 3000
        })
      }
    }
  })
}

function openContractDialog(child: Child) {
  selectedChild.value = child
  contractDialogVisible.value = true
}

function closeContractDialog() {
  contractDialogVisible.value = false
  selectedChild.value = null
}

function openHistoryDialog(child: Child) {
  historyChild.value = child
  historyDialogVisible.value = true
}

function closeHistoryDialog() {
  historyDialogVisible.value = false
  historyChild.value = null
}

async function saveContract(data: ChildContractCreateRequest) {
  if (!selectedChild.value) return

  try {
    await apiClient.createChildContract(orgId.value, selectedChild.value.id, data)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Contract created successfully',
      life: 3000
    })
    closeContractDialog()
    await fetchChildren()
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to create contract',
      life: 3000
    })
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString()
}

function getCurrentContract(child: Child) {
  if (!child.contracts || child.contracts.length === 0) return null
  const now = new Date()
  return child.contracts.find((c) => {
    const from = new Date(c.from)
    const to = c.to ? new Date(c.to) : null
    return from <= now && (!to || to >= now)
  })
}

function calculateAge(birthdate: string): number {
  const birth = new Date(birthdate)
  const today = new Date()
  let age = today.getFullYear() - birth.getFullYear()
  const monthDiff = today.getMonth() - birth.getMonth()
  if (monthDiff < 0 || (monthDiff === 0 && today.getDate() < birth.getDate())) {
    age--
  }
  return age
}

onMounted(() => {
  fetchChildren()
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1>Children</h1>
      <Button label="New Child" icon="pi pi-plus" @click="openCreateDialog" />
    </div>

    <div class="card">
      <DataTable
        :value="children"
        :loading="loading"
        striped-rows
        paginator
        :rows="10"
        :rows-per-page-options="[10, 25, 50]"
      >
        <Column field="id" header="ID" sortable style="width: 80px"></Column>
        <Column header="Name" sortable>
          <template #body="{ data }"> {{ data.first_name }} {{ data.last_name }} </template>
        </Column>
        <Column field="birthdate" header="Birthdate" sortable style="width: 130px">
          <template #body="{ data }">
            {{ formatDate(data.birthdate) }}
          </template>
        </Column>
        <Column header="Age" style="width: 80px">
          <template #body="{ data }">
            {{ calculateAge(data.birthdate) }}
          </template>
        </Column>
        <Column header="Attributes" style="width: 200px">
          <template #body="{ data }">
            <template v-if="getCurrentContract(data)?.attributes?.length">
              <Tag
                v-for="attr in getCurrentContract(data)!.attributes"
                :key="attr"
                :value="attr"
                class="mr-1"
              />
            </template>
            <span v-else>-</span>
          </template>
        </Column>
        <Column header="Actions" style="width: 250px">
          <template #body="{ data }">
            <Button
              icon="pi pi-history"
              text
              rounded
              @click="openHistoryDialog(data)"
              title="Contract History"
            />
            <Button
              icon="pi pi-file-plus"
              text
              rounded
              @click="openContractDialog(data)"
              title="Add Contract"
            />
            <Button
              icon="pi pi-pencil"
              text
              rounded
              @click="openEditDialog(data)"
              title="Edit Child"
            />
            <Button
              icon="pi pi-trash"
              text
              rounded
              severity="danger"
              @click="confirmDelete(data)"
              title="Delete Child"
            />
          </template>
        </Column>
      </DataTable>
    </div>

    <ChildForm
      :visible="dialogVisible"
      :child="editingChild"
      @close="closeDialog"
      @save="saveChild"
    />

    <ChildContractForm
      :visible="contractDialogVisible"
      :child="selectedChild"
      @close="closeContractDialog"
      @save="saveContract"
    />

    <ChildContractHistory
      :visible="historyDialogVisible"
      :child="historyChild"
      :org-id="orgId"
      @close="closeHistoryDialog"
    />
  </div>
</template>
