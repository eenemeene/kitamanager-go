<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { useI18n } from 'vue-i18n'
import { apiClient, getErrorMessage } from '@/api/client'
import { formatDate } from '@/utils/formatting'
import type {
  Employee,
  EmployeeCreateRequest,
  EmployeeUpdateRequest,
  EmployeeContractCreateRequest
} from '@/api/types'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import EmployeeForm from './EmployeeForm.vue'
import EmployeeContractForm from './EmployeeContractForm.vue'
import EmployeeContractHistory from './EmployeeContractHistory.vue'

const route = useRoute()
const toast = useToast()
const confirm = useConfirm()
const { t } = useI18n()

const orgId = ref(Number(route.params.orgId))
const employees = ref<Employee[]>([])
const loading = ref(false)

const dialogVisible = ref(false)
const editingEmployee = ref<Employee | null>(null)

const contractDialogVisible = ref(false)
const selectedEmployee = ref<Employee | null>(null)

const historyDialogVisible = ref(false)
const historyEmployee = ref<Employee | null>(null)

watch(
  () => route.params.orgId,
  (newOrgId) => {
    orgId.value = Number(newOrgId)
    fetchEmployees()
  }
)

async function fetchEmployees() {
  loading.value = true
  try {
    employees.value = await apiClient.getEmployees(orgId.value)
  } catch (error) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: getErrorMessage(error, t('common.failedToLoad', { resource: t('employees.title') })),
      life: 5000
    })
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editingEmployee.value = null
  dialogVisible.value = true
}

function openEditDialog(employee: Employee) {
  editingEmployee.value = employee
  dialogVisible.value = true
}

function closeDialog() {
  dialogVisible.value = false
  editingEmployee.value = null
}

async function saveEmployee(data: EmployeeCreateRequest | EmployeeUpdateRequest) {
  try {
    if (editingEmployee.value) {
      await apiClient.updateEmployee(
        orgId.value,
        editingEmployee.value.id,
        data as EmployeeUpdateRequest
      )
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('employees.updateSuccess'),
        life: 3000
      })
    } else {
      await apiClient.createEmployee(
        orgId.value,
        data as Omit<EmployeeCreateRequest, 'organization_id'>
      )
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('employees.createSuccess'),
        life: 3000
      })
    }
    closeDialog()
    await fetchEmployees()
  } catch (error) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: getErrorMessage(error, t('common.failedToSave', { resource: t('employees.title') })),
      life: 5000
    })
  }
}

function confirmDelete(employee: Employee) {
  confirm.require({
    message: t('employees.confirmDeleteMessage', {
      name: `${employee.first_name} ${employee.last_name}`
    }),
    header: t('common.confirmDelete'),
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await apiClient.deleteEmployee(orgId.value, employee.id)
        toast.add({
          severity: 'success',
          summary: t('common.success'),
          detail: t('employees.deleteSuccess'),
          life: 3000
        })
        await fetchEmployees()
      } catch (error) {
        toast.add({
          severity: 'error',
          summary: t('common.error'),
          detail: getErrorMessage(
            error,
            t('common.failedToDelete', { resource: t('employees.title') })
          ),
          life: 5000
        })
      }
    }
  })
}

function openContractDialog(employee: Employee) {
  selectedEmployee.value = employee
  contractDialogVisible.value = true
}

function closeContractDialog() {
  contractDialogVisible.value = false
  selectedEmployee.value = null
}

async function saveContract(data: EmployeeContractCreateRequest) {
  if (!selectedEmployee.value) return

  try {
    await apiClient.createEmployeeContract(orgId.value, selectedEmployee.value.id, data)
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('contracts.createSuccess'),
      life: 3000
    })
    closeContractDialog()
    await fetchEmployees()
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

function getCurrentContract(employee: Employee) {
  if (!employee.contracts || employee.contracts.length === 0) return null
  const now = new Date()
  return employee.contracts.find((c) => {
    const from = new Date(c.from)
    const to = c.to ? new Date(c.to) : null
    return from <= now && (!to || to >= now)
  })
}

function openHistoryDialog(employee: Employee) {
  historyEmployee.value = employee
  historyDialogVisible.value = true
}

function closeHistoryDialog() {
  historyDialogVisible.value = false
  historyEmployee.value = null
}

onMounted(() => {
  fetchEmployees()
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1>{{ t('employees.title') }}</h1>
      <Button :label="t('employees.newEmployee')" icon="pi pi-plus" @click="openCreateDialog" />
    </div>

    <div class="card">
      <DataTable
        :value="employees"
        :loading="loading"
        striped-rows
        paginator
        :rows="10"
        :rows-per-page-options="[10, 25, 50]"
        row-group-mode="subheader"
      >
        <Column field="id" :header="t('common.id')" sortable style="width: 80px"></Column>
        <Column :header="t('common.name')" sortable>
          <template #body="{ data }"> {{ data.first_name }} {{ data.last_name }} </template>
        </Column>
        <Column field="birthdate" :header="t('employees.birthdate')" sortable style="width: 130px">
          <template #body="{ data }">
            {{ formatDate(data.birthdate) }}
          </template>
        </Column>
        <Column :header="t('employees.currentPosition')" style="width: 180px">
          <template #body="{ data }">
            {{ getCurrentContract(data)?.position || '-' }}
          </template>
        </Column>
        <Column :header="t('employees.weeklyHours')" style="width: 100px">
          <template #body="{ data }">
            {{ getCurrentContract(data)?.weekly_hours || '-' }}
          </template>
        </Column>
        <Column :header="t('employees.grade')" style="width: 80px">
          <template #body="{ data }">
            {{ getCurrentContract(data)?.grade || '-' }}
          </template>
        </Column>
        <Column :header="t('employees.step')" style="width: 80px">
          <template #body="{ data }">
            {{ getCurrentContract(data)?.step || '-' }}
          </template>
        </Column>
        <Column :header="t('common.actions')" style="width: 240px">
          <template #body="{ data }">
            <Button
              icon="pi pi-history"
              text
              rounded
              @click="openHistoryDialog(data)"
              :title="t('employees.contractHistory')"
            />
            <Button
              icon="pi pi-file"
              text
              rounded
              @click="openContractDialog(data)"
              :title="t('employees.addContract')"
            />
            <Button
              icon="pi pi-pencil"
              text
              rounded
              @click="openEditDialog(data)"
              :title="t('employees.editEmployee')"
            />
            <Button
              icon="pi pi-trash"
              text
              rounded
              severity="danger"
              @click="confirmDelete(data)"
              :title="t('employees.deleteEmployee')"
            />
          </template>
        </Column>
      </DataTable>
    </div>

    <EmployeeForm
      :visible="dialogVisible"
      :employee="editingEmployee"
      @close="closeDialog"
      @save="saveEmployee"
    />

    <EmployeeContractForm
      :visible="contractDialogVisible"
      :employee="selectedEmployee"
      @close="closeContractDialog"
      @save="saveContract"
    />

    <EmployeeContractHistory
      :visible="historyDialogVisible"
      :employee="historyEmployee"
      :org-id="orgId"
      @close="closeHistoryDialog"
    />
  </div>
</template>
