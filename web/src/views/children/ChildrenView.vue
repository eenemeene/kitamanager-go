<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { useI18n } from 'vue-i18n'
import { apiClient, getErrorMessage } from '@/api/client'
import { formatDate, formatCurrency, calculateAge } from '@/utils/formatting'
import type {
  Child,
  ChildCreateRequest,
  ChildUpdateRequest,
  ChildContractCreateRequest,
  ChildFundingResponse
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
const { t } = useI18n()

const orgId = ref(Number(route.params.orgId))
const children = ref<Child[]>([])
const fundingData = ref<ChildFundingResponse[]>([])
const loading = ref(false)

const dialogVisible = ref(false)
const editingChild = ref<Child | null>(null)

const contractDialogVisible = ref(false)
const selectedChild = ref<Child | null>(null)

const historyDialogVisible = ref(false)
const historyChild = ref<Child | null>(null)

// Computed property for the current contract of the selected child
const selectedChildCurrentContract = computed(() => {
  if (!selectedChild.value) return null
  return getCurrentContract(selectedChild.value) ?? null
})

// Map of child ID to funding data for quick lookup
const fundingByChildId = computed(() => {
  const map = new Map<number, ChildFundingResponse>()
  for (const f of fundingData.value) {
    map.set(f.child_id, f)
  }
  return map
})

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
    const [childrenResult, fundingResult] = await Promise.all([
      apiClient.getChildren(orgId.value),
      apiClient.getChildrenFunding(orgId.value).catch((error) => {
        console.warn('Failed to load funding data:', error)
        toast.add({
          severity: 'warn',
          summary: t('common.warning'),
          detail: t('children.fundingDataUnavailable'),
          life: 5000
        })
        return { date: '', children: [] }
      })
    ])
    children.value = childrenResult
    fundingData.value = fundingResult.children
  } catch (error) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: getErrorMessage(error, t('common.failedToLoad', { resource: t('children.title') })),
      life: 5000
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
        summary: t('common.success'),
        detail: t('children.updateSuccess'),
        life: 3000
      })
    } else {
      await apiClient.createChild(orgId.value, data as Omit<ChildCreateRequest, 'organization_id'>)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('children.createSuccess'),
        life: 3000
      })
    }
    closeDialog()
    await fetchChildren()
  } catch (error) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: getErrorMessage(error, t('common.failedToSave', { resource: t('children.title') })),
      life: 5000
    })
  }
}

function confirmDelete(child: Child) {
  confirm.require({
    message: t('children.confirmDeleteMessage', { name: `${child.first_name} ${child.last_name}` }),
    header: t('common.confirmDelete'),
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await apiClient.deleteChild(orgId.value, child.id)
        toast.add({
          severity: 'success',
          summary: t('common.success'),
          detail: t('children.deleteSuccess'),
          life: 3000
        })
        await fetchChildren()
      } catch (error) {
        toast.add({
          severity: 'error',
          summary: t('common.error'),
          detail: getErrorMessage(
            error,
            t('common.failedToDelete', { resource: t('children.title') })
          ),
          life: 5000
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

async function saveContract(data: ChildContractCreateRequest, endCurrentContract: boolean) {
  if (!selectedChild.value) return

  try {
    // If we need to end the current contract first
    if (endCurrentContract && selectedChildCurrentContract.value) {
      const newStartDate = new Date(data.from)
      const endDate = new Date(newStartDate)
      endDate.setDate(endDate.getDate() - 1)

      await apiClient.updateChildContract(
        orgId.value,
        selectedChild.value.id,
        selectedChildCurrentContract.value.id,
        { to: endDate.toISOString() }
      )
    }

    // Create the new contract
    await apiClient.createChildContract(orgId.value, selectedChild.value.id, data)
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: endCurrentContract
        ? t('contracts.previousContractEnded')
        : t('contracts.createSuccess'),
      life: 3000
    })
    closeContractDialog()
    await fetchChildren()
  } catch (error) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: getErrorMessage(
        error,
        t('common.failedToCreate', { resource: t('contracts.title') })
      ),
      life: 5000
    })
  }
}

function getChildFunding(childId: number): number | null {
  return fundingByChildId.value.get(childId)?.funding ?? null
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

onMounted(() => {
  fetchChildren()
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1>{{ t('children.title') }}</h1>
      <Button :label="t('children.newChild')" icon="pi pi-plus" @click="openCreateDialog" />
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
        <Column field="id" :header="t('common.id')" sortable style="width: 80px"></Column>
        <Column :header="t('common.name')" sortable>
          <template #body="{ data }"> {{ data.first_name }} {{ data.last_name }} </template>
        </Column>
        <Column field="birthdate" :header="t('children.birthdate')" sortable style="width: 130px">
          <template #body="{ data }">
            {{ formatDate(data.birthdate) }}
          </template>
        </Column>
        <Column :header="t('children.age')" style="width: 80px">
          <template #body="{ data }">
            {{ calculateAge(data.birthdate) }}
          </template>
        </Column>
        <Column :header="t('children.attributes')" style="width: 200px">
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
        <Column :header="t('children.funding')" style="width: 120px">
          <template #body="{ data }">
            <span v-if="getChildFunding(data.id) !== null">
              {{ formatCurrency(getChildFunding(data.id)!) }}
            </span>
            <span v-else>-</span>
          </template>
        </Column>
        <Column :header="t('common.actions')" style="width: 250px">
          <template #body="{ data }">
            <Button
              icon="pi pi-history"
              text
              rounded
              @click="openHistoryDialog(data)"
              :title="t('children.contractHistory')"
            />
            <Button
              icon="pi pi-file-plus"
              text
              rounded
              @click="openContractDialog(data)"
              :title="t('children.addContract')"
            />
            <Button
              icon="pi pi-pencil"
              text
              rounded
              @click="openEditDialog(data)"
              :title="t('children.editChild')"
            />
            <Button
              icon="pi pi-trash"
              text
              rounded
              severity="danger"
              @click="confirmDelete(data)"
              :title="t('children.deleteChild')"
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
      :current-contract="selectedChildCurrentContract"
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
