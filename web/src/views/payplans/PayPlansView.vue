<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { useI18n } from 'vue-i18n'
import { apiClient, getErrorMessage } from '@/api/client'
import type { PayPlan } from '@/api/types'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import PayPlanForm from './PayPlanForm.vue'

const props = defineProps<{
  orgId: string
}>()

const route = useRoute()
const router = useRouter()
const toast = useToast()
const confirm = useConfirm()
const { t } = useI18n()

const payPlans = ref<PayPlan[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const editingPayPlan = ref<PayPlan | null>(null)

const orgId = computed(() => Number(props.orgId || route.params.orgId))

async function fetchPayPlans() {
  if (!orgId.value) return
  loading.value = true
  try {
    payPlans.value = await apiClient.getPayPlans(orgId.value)
  } catch (error) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: getErrorMessage(error, t('common.failedToLoad', { resource: t('payPlans.title') })),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editingPayPlan.value = null
  dialogVisible.value = true
}

function openEditDialog(payPlan: PayPlan) {
  editingPayPlan.value = payPlan
  dialogVisible.value = true
}

function closeDialog() {
  dialogVisible.value = false
  editingPayPlan.value = null
}

async function savePayPlan(data: { name: string }) {
  try {
    if (editingPayPlan.value) {
      await apiClient.updatePayPlan(orgId.value, editingPayPlan.value.id, data)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('payPlans.updateSuccess'),
        life: 3000
      })
    } else {
      await apiClient.createPayPlan(orgId.value, data)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('payPlans.createSuccess'),
        life: 3000
      })
    }
    closeDialog()
    await fetchPayPlans()
  } catch (error) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: getErrorMessage(error, t('common.failedToSave', { resource: t('payPlans.title') })),
      life: 3000
    })
  }
}

function confirmDelete(payPlan: PayPlan) {
  confirm.require({
    message: t('payPlans.deleteConfirm'),
    header: t('common.confirmDelete'),
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await apiClient.deletePayPlan(orgId.value, payPlan.id)
        toast.add({
          severity: 'success',
          summary: t('common.success'),
          detail: t('payPlans.deleteSuccess'),
          life: 3000
        })
        await fetchPayPlans()
      } catch (error) {
        toast.add({
          severity: 'error',
          summary: t('common.error'),
          detail: getErrorMessage(
            error,
            t('common.failedToDelete', { resource: t('payPlans.title') })
          ),
          life: 3000
        })
      }
    }
  })
}

function viewDetails(payPlan: PayPlan) {
  router.push({
    name: 'payplan-detail',
    params: { orgId: orgId.value, id: payPlan.id }
  })
}

watch(
  () => props.orgId,
  () => {
    fetchPayPlans()
  }
)

onMounted(() => {
  fetchPayPlans()
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1>{{ t('payPlans.title') }}</h1>
      <Button :label="t('payPlans.newPayPlan')" icon="pi pi-plus" @click="openCreateDialog" />
    </div>

    <div class="card">
      <DataTable
        :value="payPlans"
        :loading="loading"
        striped-rows
        show-gridlines
        paginator
        :rows="20"
        :rows-per-page-options="[10, 20, 50]"
        data-testid="payplans-table"
      >
        <template #empty>
          <div class="text-center p-4">{{ t('common.noResults') }}</div>
        </template>

        <Column field="name" :header="t('common.name')" sortable />
        <Column :header="t('common.actions')" style="width: 150px">
          <template #body="{ data }">
            <Button
              icon="pi pi-eye"
              text
              rounded
              size="small"
              :title="t('common.viewDetails')"
              @click="viewDetails(data)"
              data-testid="view-btn"
            />
            <Button
              icon="pi pi-pencil"
              text
              rounded
              size="small"
              :title="t('common.edit')"
              @click="openEditDialog(data)"
              data-testid="edit-btn"
            />
            <Button
              icon="pi pi-trash"
              text
              rounded
              size="small"
              severity="danger"
              :title="t('common.delete')"
              @click="confirmDelete(data)"
              data-testid="delete-btn"
            />
          </template>
        </Column>
      </DataTable>
    </div>

    <PayPlanForm
      :visible="dialogVisible"
      :pay-plan="editingPayPlan"
      @save="savePayPlan"
      @close="closeDialog"
    />
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.card {
  background: var(--surface-card);
  padding: 1.5rem;
  border-radius: var(--border-radius);
  box-shadow: var(--card-shadow);
}

.text-center {
  text-align: center;
}

.p-4 {
  padding: 1rem;
}
</style>
