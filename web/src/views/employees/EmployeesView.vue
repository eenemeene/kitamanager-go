<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { apiClient } from '@/api/client'
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

const route = useRoute()
const toast = useToast()
const confirm = useConfirm()

const orgId = ref(Number(route.params.orgId))
const employees = ref<Employee[]>([])
const loading = ref(false)

const dialogVisible = ref(false)
const editingEmployee = ref<Employee | null>(null)

const contractDialogVisible = ref(false)
const selectedEmployee = ref<Employee | null>(null)

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
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to load employees',
      life: 3000
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
        summary: 'Success',
        detail: 'Employee updated successfully',
        life: 3000
      })
    } else {
      await apiClient.createEmployee(
        orgId.value,
        data as Omit<EmployeeCreateRequest, 'organization_id'>
      )
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'Employee created successfully',
        life: 3000
      })
    }
    closeDialog()
    await fetchEmployees()
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to save employee',
      life: 3000
    })
  }
}

function confirmDelete(employee: Employee) {
  confirm.require({
    message: `Are you sure you want to delete ${employee.first_name} ${employee.last_name}?`,
    header: 'Confirm Delete',
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await apiClient.deleteEmployee(orgId.value, employee.id)
        toast.add({
          severity: 'success',
          summary: 'Success',
          detail: 'Employee deleted successfully',
          life: 3000
        })
        await fetchEmployees()
      } catch {
        toast.add({
          severity: 'error',
          summary: 'Error',
          detail: 'Failed to delete employee',
          life: 3000
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
      summary: 'Success',
      detail: 'Contract created successfully',
      life: 3000
    })
    closeContractDialog()
    await fetchEmployees()
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

function formatCurrency(cents: number): string {
  return new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR' }).format(cents / 100)
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

onMounted(() => {
  fetchEmployees()
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1>Employees</h1>
      <Button label="New Employee" icon="pi pi-plus" @click="openCreateDialog" />
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
        <Column field="id" header="ID" sortable style="width: 80px"></Column>
        <Column header="Name" sortable>
          <template #body="{ data }"> {{ data.first_name }} {{ data.last_name }} </template>
        </Column>
        <Column field="birthdate" header="Birthdate" sortable style="width: 130px">
          <template #body="{ data }">
            {{ formatDate(data.birthdate) }}
          </template>
        </Column>
        <Column header="Current Position" style="width: 180px">
          <template #body="{ data }">
            {{ getCurrentContract(data)?.position || '-' }}
          </template>
        </Column>
        <Column header="Weekly Hours" style="width: 120px">
          <template #body="{ data }">
            {{ getCurrentContract(data)?.weekly_hours || '-' }}
          </template>
        </Column>
        <Column header="Salary" style="width: 120px">
          <template #body="{ data }">
            {{ getCurrentContract(data) ? formatCurrency(getCurrentContract(data)!.salary) : '-' }}
          </template>
        </Column>
        <Column header="Actions" style="width: 200px">
          <template #body="{ data }">
            <Button
              icon="pi pi-file"
              text
              rounded
              @click="openContractDialog(data)"
              title="Add Contract"
            />
            <Button icon="pi pi-pencil" text rounded @click="openEditDialog(data)" />
            <Button
              icon="pi pi-trash"
              text
              rounded
              severity="danger"
              @click="confirmDelete(data)"
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
  </div>
</template>
