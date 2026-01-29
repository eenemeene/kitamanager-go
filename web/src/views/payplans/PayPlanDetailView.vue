<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { useI18n } from 'vue-i18n'
import { apiClient, getErrorMessage } from '@/api/client'
import type {
  PayPlan,
  PayPlanPeriod,
  PayPlanEntry,
  PayPlanPeriodCreateRequest,
  PayPlanEntryCreateRequest,
  PayPlanPeriodUpdateRequest,
  PayPlanEntryUpdateRequest
} from '@/api/types'
import { formatDate, formatCurrency, formatDateToISO } from '@/utils/formatting'
import Button from 'primevue/button'
import Panel from 'primevue/panel'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import Calendar from 'primevue/calendar'
import SelectButton from 'primevue/selectbutton'

const props = defineProps<{
  orgId: string
  id: string
}>()

const route = useRoute()
const router = useRouter()
const toast = useToast()
const confirm = useConfirm()
const { t } = useI18n()

const orgId = computed(() => Number(props.orgId || route.params.orgId))
const payPlanId = computed(() => Number(props.id || route.params.id))
const payPlan = ref<PayPlan | null>(null)
const loading = ref(false)

// View mode toggle
const viewMode = ref<'panels' | 'table'>('panels')
const viewModeOptions = computed(() => [
  { label: t('payPlans.viewPanels'), value: 'panels', icon: 'pi pi-list' },
  { label: t('payPlans.viewTable'), value: 'table', icon: 'pi pi-table' }
])

// Dialog states
const periodDialog = ref(false)
const entryDialog = ref(false)

// Current editing contexts
const editingPeriod = ref<PayPlanPeriod | null>(null)
const editingEntry = ref<PayPlanEntry | null>(null)
const currentPeriodId = ref<number | null>(null)

// Form data
const periodForm = ref({
  from: null as Date | null,
  to: null as Date | null | undefined,
  weekly_hours: 39.0
})

const entryForm = ref({
  grade: '',
  step: 1,
  monthly_amount: 0
})

// Compute flattened rows for table view
const flattenedRows = computed(() => {
  if (!payPlan.value?.periods) return []

  const rows: Array<{
    periodId: number
    periodFrom: string
    periodTo: string | null | undefined
    weeklyHours: number
    entryId: number | null
    grade: string
    step: number | null
    monthlyAmount: number
    isFirstEntryInPeriod: boolean
  }> = []

  for (const period of payPlan.value.periods) {
    if (period.entries && period.entries.length > 0) {
      period.entries.forEach((entry, index) => {
        rows.push({
          periodId: period.id,
          periodFrom: period.from,
          periodTo: period.to,
          weeklyHours: period.weekly_hours,
          entryId: entry.id,
          grade: entry.grade,
          step: entry.step,
          monthlyAmount: entry.monthly_amount,
          isFirstEntryInPeriod: index === 0
        })
      })
    } else {
      rows.push({
        periodId: period.id,
        periodFrom: period.from,
        periodTo: period.to,
        weeklyHours: period.weekly_hours,
        entryId: null,
        grade: '-',
        step: null,
        monthlyAmount: 0,
        isFirstEntryInPeriod: true
      })
    }
  }

  return rows
})

async function fetchPayPlan() {
  loading.value = true
  try {
    payPlan.value = await apiClient.getPayPlan(orgId.value, payPlanId.value)
  } catch (error) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: getErrorMessage(error, t('payPlans.failedToLoadPayPlan')),
      life: 3000
    })
    router.push({ name: 'payplans', params: { orgId: orgId.value } })
  } finally {
    loading.value = false
  }
}

// Period functions
function openAddPeriod() {
  editingPeriod.value = null
  periodForm.value = { from: null, to: undefined, weekly_hours: 39.0 }
  periodDialog.value = true
}

function openEditPeriod(period: PayPlanPeriod) {
  editingPeriod.value = period
  periodForm.value = {
    from: new Date(period.from),
    to: period.to ? new Date(period.to) : undefined,
    weekly_hours: period.weekly_hours
  }
  periodDialog.value = true
}

async function savePeriod() {
  if (!periodForm.value.from) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: t('validation.fromDateRequired'),
      life: 3000
    })
    return
  }

  if (periodForm.value.weekly_hours <= 0) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: t('payPlans.weeklyHoursMin'),
      life: 3000
    })
    return
  }

  try {
    if (editingPeriod.value) {
      const data: PayPlanPeriodUpdateRequest = {
        from: formatDateToISO(periodForm.value.from),
        to: periodForm.value.to ? formatDateToISO(periodForm.value.to) : null,
        weekly_hours: periodForm.value.weekly_hours
      }
      await apiClient.updatePayPlanPeriod(
        orgId.value,
        payPlanId.value,
        editingPeriod.value.id,
        data
      )
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('payPlans.periodUpdated'),
        life: 3000
      })
    } else {
      const data: PayPlanPeriodCreateRequest = {
        from: formatDateToISO(periodForm.value.from),
        to: periodForm.value.to ? formatDateToISO(periodForm.value.to) : undefined,
        weekly_hours: periodForm.value.weekly_hours
      }
      await apiClient.createPayPlanPeriod(orgId.value, payPlanId.value, data)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('payPlans.periodCreated'),
        life: 3000
      })
    }
    periodDialog.value = false
    await fetchPayPlan()
  } catch (error) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: getErrorMessage(error, t('payPlans.failedToSavePeriod')),
      life: 3000
    })
  }
}

function confirmDeletePeriod(period: PayPlanPeriod) {
  confirm.require({
    message: t('payPlans.deletePeriodConfirm'),
    header: t('common.confirmDelete'),
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await apiClient.deletePayPlanPeriod(orgId.value, payPlanId.value, period.id)
        toast.add({
          severity: 'success',
          summary: t('common.success'),
          detail: t('payPlans.periodDeleted'),
          life: 3000
        })
        await fetchPayPlan()
      } catch (error) {
        toast.add({
          severity: 'error',
          summary: t('common.error'),
          detail: getErrorMessage(error, t('payPlans.failedToDeletePeriod')),
          life: 3000
        })
      }
    }
  })
}

// Entry functions
function openAddEntry(periodId: number) {
  currentPeriodId.value = periodId
  editingEntry.value = null
  entryForm.value = {
    grade: '',
    step: 1,
    monthly_amount: 0
  }
  entryDialog.value = true
}

function openEditEntry(periodId: number, entry: PayPlanEntry) {
  currentPeriodId.value = periodId
  editingEntry.value = entry
  entryForm.value = {
    grade: entry.grade,
    step: entry.step,
    monthly_amount: entry.monthly_amount
  }
  entryDialog.value = true
}

async function saveEntry() {
  if (!entryForm.value.grade.trim()) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: t('payPlans.gradeRequired'),
      life: 3000
    })
    return
  }

  if (entryForm.value.step < 1 || entryForm.value.step > 6) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: t('payPlans.stepMin'),
      life: 3000
    })
    return
  }

  try {
    if (editingEntry.value && currentPeriodId.value) {
      const data: PayPlanEntryUpdateRequest = {
        grade: entryForm.value.grade,
        step: entryForm.value.step,
        monthly_amount: entryForm.value.monthly_amount
      }
      await apiClient.updatePayPlanEntry(
        orgId.value,
        payPlanId.value,
        currentPeriodId.value,
        editingEntry.value.id,
        data
      )
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('payPlans.entryUpdated'),
        life: 3000
      })
    } else if (currentPeriodId.value) {
      const data: PayPlanEntryCreateRequest = {
        grade: entryForm.value.grade,
        step: entryForm.value.step,
        monthly_amount: entryForm.value.monthly_amount
      }
      await apiClient.createPayPlanEntry(orgId.value, payPlanId.value, currentPeriodId.value, data)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('payPlans.entryCreated'),
        life: 3000
      })
    }
    entryDialog.value = false
    await fetchPayPlan()
  } catch (error) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: getErrorMessage(error, t('payPlans.failedToSaveEntry')),
      life: 3000
    })
  }
}

function confirmDeleteEntry(periodId: number, entry: PayPlanEntry) {
  confirm.require({
    message: t('payPlans.deleteEntryConfirm'),
    header: t('common.confirmDelete'),
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await apiClient.deletePayPlanEntry(orgId.value, payPlanId.value, periodId, entry.id)
        toast.add({
          severity: 'success',
          summary: t('common.success'),
          detail: t('payPlans.entryDeleted'),
          life: 3000
        })
        await fetchPayPlan()
      } catch (error) {
        toast.add({
          severity: 'error',
          summary: t('common.error'),
          detail: getErrorMessage(error, t('payPlans.failedToDeleteEntry')),
          life: 3000
        })
      }
    }
  })
}

onMounted(() => {
  fetchPayPlan()
})
</script>

<template>
  <div v-if="payPlan">
    <div class="page-header">
      <div class="flex align-items-center gap-2">
        <Button
          icon="pi pi-arrow-left"
          text
          @click="router.push({ name: 'payplans', params: { orgId: orgId } })"
          data-testid="back-btn"
        />
        <h1>{{ payPlan.name }}</h1>
      </div>
      <div class="flex align-items-center gap-2">
        <SelectButton
          v-model="viewMode"
          :options="viewModeOptions"
          option-label="label"
          option-value="value"
          data-testid="view-mode-toggle"
        />
        <Button
          :label="t('payPlans.addPeriod')"
          icon="pi pi-plus"
          @click="openAddPeriod"
          data-testid="add-period-btn"
        />
      </div>
    </div>

    <!-- Table View -->
    <div v-if="viewMode === 'table'" class="card">
      <DataTable
        v-if="flattenedRows.length > 0"
        :value="flattenedRows"
        striped-rows
        show-gridlines
        size="small"
        class="payplan-table"
        data-testid="payplan-table"
      >
        <Column :header="t('payPlans.period')" style="min-width: 180px">
          <template #body="{ data }">
            <template v-if="data.isFirstEntryInPeriod">
              <div class="period-cell">
                <strong>{{ formatDate(data.periodFrom) }}</strong>
                <span> - </span>
                <strong>{{
                  data.periodTo ? formatDate(data.periodTo) : t('common.ongoing')
                }}</strong>
                <div class="text-secondary text-sm">
                  {{ data.weeklyHours }}h/{{ t('payPlans.weeklyHours') }}
                </div>
              </div>
            </template>
          </template>
        </Column>
        <Column field="grade" :header="t('payPlans.grade')" style="width: 100px">
          <template #body="{ data }">
            <span :class="{ 'text-secondary': !data.entryId }">{{ data.grade }}</span>
          </template>
        </Column>
        <Column :header="t('payPlans.step')" style="width: 80px">
          <template #body="{ data }">
            <span v-if="data.entryId">{{ data.step }}</span>
            <span v-else class="text-secondary">-</span>
          </template>
        </Column>
        <Column :header="t('payPlans.monthlyAmount')" style="width: 140px; text-align: right">
          <template #body="{ data }">
            <span v-if="data.entryId">{{ formatCurrency(data.monthlyAmount) }}</span>
            <span v-else class="text-secondary">-</span>
          </template>
        </Column>
      </DataTable>
      <p v-else class="text-secondary">
        {{ t('payPlans.noDataDefined') }}
      </p>
    </div>

    <!-- Panels View -->
    <div v-else-if="payPlan.periods && payPlan.periods.length > 0">
      <Panel
        v-for="period in payPlan.periods"
        :key="period.id"
        :header="`${formatDate(period.from)} - ${period.to ? formatDate(period.to) : t('common.ongoing')} (${period.weekly_hours}h)`"
        toggleable
        class="mb-3"
        :data-testid="`period-panel-${period.id}`"
      >
        <template #icons>
          <Button
            icon="pi pi-plus"
            text
            :title="t('payPlans.addEntry')"
            @click.stop="openAddEntry(period.id)"
            data-testid="add-entry-btn"
          />
          <Button
            icon="pi pi-pencil"
            text
            :title="t('payPlans.editPeriod')"
            @click.stop="openEditPeriod(period)"
            data-testid="edit-period-btn"
          />
          <Button
            icon="pi pi-trash"
            text
            severity="danger"
            :title="t('payPlans.deletePeriod')"
            @click.stop="confirmDeletePeriod(period)"
            data-testid="delete-period-btn"
          />
        </template>

        <DataTable
          v-if="period.entries && period.entries.length > 0"
          :value="period.entries"
          size="small"
          striped-rows
          data-testid="entries-table"
        >
          <Column field="grade" :header="t('payPlans.grade')" />
          <Column field="step" :header="t('payPlans.step')" />
          <Column field="monthly_amount" :header="t('payPlans.monthlyAmount')">
            <template #body="{ data }">
              {{ formatCurrency(data.monthly_amount) }}
            </template>
          </Column>
          <Column :header="t('common.actions')" style="width: 100px">
            <template #body="{ data: entry }">
              <Button
                icon="pi pi-pencil"
                text
                rounded
                size="small"
                @click="openEditEntry(period.id, entry)"
                data-testid="edit-entry-btn"
              />
              <Button
                icon="pi pi-trash"
                text
                rounded
                size="small"
                severity="danger"
                @click="confirmDeleteEntry(period.id, entry)"
                data-testid="delete-entry-btn"
              />
            </template>
          </Column>
        </DataTable>
        <p v-else class="text-secondary">{{ t('payPlans.noEntriesDefined') }}</p>
      </Panel>
    </div>
    <div v-else-if="viewMode === 'panels'" class="card">
      <p class="text-secondary">{{ t('payPlans.noPeriodsDefined') }}</p>
    </div>

    <!-- Period Dialog -->
    <Dialog
      v-model:visible="periodDialog"
      :header="editingPeriod ? t('payPlans.editPeriod') : t('payPlans.addPeriod')"
      modal
      :style="{ width: '450px' }"
      data-testid="period-dialog"
    >
      <div class="form-grid">
        <div class="field">
          <label for="period-from">{{ t('payPlans.fromDate') }}</label>
          <Calendar
            id="period-from"
            v-model="periodForm.from"
            date-format="yy-mm-dd"
            show-icon
            data-testid="period-from-input"
          />
        </div>
        <div class="field">
          <label for="period-to">{{ t('payPlans.toDateOptional') }}</label>
          <Calendar
            id="period-to"
            v-model="periodForm.to"
            date-format="yy-mm-dd"
            show-icon
            :show-clear="true"
            data-testid="period-to-input"
          />
        </div>
        <div class="field">
          <label for="period-weekly-hours">{{ t('payPlans.weeklyHoursLabel') }}</label>
          <InputNumber
            id="period-weekly-hours"
            v-model="periodForm.weekly_hours"
            :min="0.1"
            :max="168"
            :min-fraction-digits="1"
            :max-fraction-digits="2"
            data-testid="period-weekly-hours-input"
          />
        </div>
      </div>
      <template #footer>
        <Button
          :label="t('common.cancel')"
          text
          @click="periodDialog = false"
          data-testid="period-cancel-btn"
        />
        <Button :label="t('common.save')" @click="savePeriod" data-testid="period-save-btn" />
      </template>
    </Dialog>

    <!-- Entry Dialog -->
    <Dialog
      v-model:visible="entryDialog"
      :header="editingEntry ? t('payPlans.editEntry') : t('payPlans.addEntry')"
      modal
      :style="{ width: '450px' }"
      data-testid="entry-dialog"
    >
      <div class="form-grid">
        <div class="field">
          <label for="entry-grade">{{ t('payPlans.gradeLabel') }}</label>
          <InputText
            id="entry-grade"
            v-model="entryForm.grade"
            placeholder="e.g., S8a, S11b"
            data-testid="entry-grade-input"
          />
        </div>
        <div class="field">
          <label for="entry-step">{{ t('payPlans.stepLabel') }}</label>
          <InputNumber
            id="entry-step"
            v-model="entryForm.step"
            :min="1"
            :max="6"
            :show-buttons="true"
            data-testid="entry-step-input"
          />
        </div>
        <div class="field">
          <label for="entry-monthly-amount">{{ t('payPlans.monthlyAmountInCents') }}</label>
          <InputNumber
            id="entry-monthly-amount"
            v-model="entryForm.monthly_amount"
            :min="0"
            data-testid="entry-monthly-amount-input"
          />
          <small class="text-secondary">{{ formatCurrency(entryForm.monthly_amount) }}</small>
        </div>
      </div>
      <template #footer>
        <Button
          :label="t('common.cancel')"
          text
          @click="entryDialog = false"
          data-testid="entry-cancel-btn"
        />
        <Button :label="t('common.save')" @click="saveEntry" data-testid="entry-save-btn" />
      </template>
    </Dialog>
  </div>
  <div v-else-if="loading" class="card">
    <p>{{ t('common.loading') }}</p>
  </div>
</template>

<style scoped>
.flex {
  display: flex;
}

.align-items-center {
  align-items: center;
}

.gap-2 {
  gap: 0.5rem;
}

.mb-3 {
  margin-bottom: 1rem;
}

.text-secondary {
  color: var(--text-color-secondary);
}

.text-sm {
  font-size: 0.875rem;
}

.period-cell {
  line-height: 1.4;
}

.payplan-table :deep(td) {
  vertical-align: top;
}

.payplan-table :deep(.p-datatable-tbody > tr > td:nth-child(4)) {
  text-align: right;
}

.card {
  background: var(--surface-card);
  padding: 1.5rem;
  border-radius: var(--border-radius);
  box-shadow: var(--card-shadow);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.form-grid {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.field label {
  font-weight: 600;
}
</style>
