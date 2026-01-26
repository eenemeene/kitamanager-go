<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { apiClient } from '@/api/client'
import type { Group, GroupCreateRequest, GroupUpdateRequest } from '@/api/types'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import GroupForm from './GroupForm.vue'

const route = useRoute()
const toast = useToast()
const confirm = useConfirm()

const orgId = ref(Number(route.params.orgId))
const groups = ref<Group[]>([])
const loading = ref(false)

const dialogVisible = ref(false)
const editingGroup = ref<Group | null>(null)

watch(
  () => route.params.orgId,
  (newOrgId) => {
    orgId.value = Number(newOrgId)
    fetchGroups()
  }
)

async function fetchGroups() {
  loading.value = true
  try {
    groups.value = await apiClient.getGroups(orgId.value)
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to load groups',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editingGroup.value = null
  dialogVisible.value = true
}

function openEditDialog(group: Group) {
  editingGroup.value = group
  dialogVisible.value = true
}

function closeDialog() {
  dialogVisible.value = false
  editingGroup.value = null
}

async function saveGroup(data: GroupCreateRequest | GroupUpdateRequest) {
  try {
    if (editingGroup.value) {
      await apiClient.updateGroup(orgId.value, editingGroup.value.id, data as GroupUpdateRequest)
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'Group updated successfully',
        life: 3000
      })
    } else {
      await apiClient.createGroup(orgId.value, data as GroupCreateRequest)
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'Group created successfully',
        life: 3000
      })
    }
    closeDialog()
    await fetchGroups()
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to save group',
      life: 3000
    })
  }
}

function confirmDelete(group: Group) {
  confirm.require({
    message: `Are you sure you want to delete "${group.name}"?`,
    header: 'Confirm Delete',
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await apiClient.deleteGroup(orgId.value, group.id)
        toast.add({
          severity: 'success',
          summary: 'Success',
          detail: 'Group deleted successfully',
          life: 3000
        })
        await fetchGroups()
      } catch {
        toast.add({
          severity: 'error',
          summary: 'Error',
          detail: 'Failed to delete group',
          life: 3000
        })
      }
    }
  })
}

onMounted(() => {
  fetchGroups()
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
      :group="editingGroup"
      @close="closeDialog"
      @save="saveGroup"
    />
  </div>
</template>
