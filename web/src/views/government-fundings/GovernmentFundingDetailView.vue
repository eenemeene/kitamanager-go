<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'
import type {
  GovernmentFunding,
  GovernmentFundingPeriod,
  GovernmentFundingProperty,
  GovernmentFundingPeriodCreateRequest,
  GovernmentFundingPropertyCreateRequest,
  GovernmentFundingPeriodUpdateRequest,
  GovernmentFundingPropertyUpdateRequest
} from '@/api/types'
import { flattenGovernmentFundingToRows } from '@/utils/government-funding'
import { formatDate, formatCurrency, formatDateToISO } from '@/utils/formatting'
import Button from 'primevue/button'
import Panel from 'primevue/panel'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import Textarea from 'primevue/textarea'
import Calendar from 'primevue/calendar'
import SelectButton from 'primevue/selectbutton'

const route = useRoute()
const router = useRouter()
const toast = useToast()
const confirm = useConfirm()
const { t } = useI18n()

const governmentFundingId = computed(() => Number(route.params.id))
const governmentFunding = ref<GovernmentFunding | null>(null)
const loading = ref(false)
const showAllPeriods = ref(false)
const loadingAllPeriods = ref(false)

// View mode toggle
const viewMode = ref<'panels' | 'table'>('panels')
const viewModeOptions = computed(() => [
  { label: t('governmentFundings.viewPanels'), value: 'panels', icon: 'pi pi-list' },
  { label: t('governmentFundings.viewTable'), value: 'table', icon: 'pi pi-table' }
])

// Compute flattened rows for table view using utility function
const flattenedRows = computed(() => flattenGovernmentFundingToRows(governmentFunding.value))

// Dialog states
const periodDialog = ref(false)
const propertyDialog = ref(false)

// Current editing contexts
const editingPeriod = ref<GovernmentFundingPeriod | null>(null)
const editingProperty = ref<GovernmentFundingProperty | null>(null)
const currentPeriodId = ref<number | null>(null)

// Form data
const periodForm = ref({
  from: null as Date | null,
  to: null as Date | null | undefined,
  comment: ''
})

const propertyForm = ref({
  name: '',
  payment: 0,
  requirement: 0,
  min_age: null as number | null,
  max_age: null as number | null,
  comment: ''
})

async function fetchGovernmentFunding(periodsLimit?: number) {
  loading.value = true
  try {
    // Default to 1 (latest period only) for performance, 0 = all periods
    const limit = periodsLimit !== undefined ? periodsLimit : showAllPeriods.value ? 0 : 1
    governmentFunding.value = await apiClient.getGovernmentFunding(governmentFundingId.value, limit)
  } catch {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: t('governmentFundings.failedToLoadFunding'),
      life: 3000
    })
    router.push({ name: 'government-fundings' })
  } finally {
    loading.value = false
  }
}

async function loadAllPeriods() {
  loadingAllPeriods.value = true
  showAllPeriods.value = true
  await fetchGovernmentFunding(0)
  loadingAllPeriods.value = false
}

// Period functions
function openAddPeriod() {
  editingPeriod.value = null
  periodForm.value = { from: null, to: undefined, comment: '' }
  periodDialog.value = true
}

function openEditPeriod(period: GovernmentFundingPeriod) {
  editingPeriod.value = period
  periodForm.value = {
    from: new Date(period.from),
    to: period.to ? new Date(period.to) : undefined,
    comment: period.comment || ''
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

  try {
    if (editingPeriod.value) {
      const data: GovernmentFundingPeriodUpdateRequest = {
        from: formatDateToISO(periodForm.value.from),
        to: periodForm.value.to ? formatDateToISO(periodForm.value.to) : null,
        comment: periodForm.value.comment || undefined
      }
      await apiClient.updateGovernmentFundingPeriod(
        governmentFundingId.value,
        editingPeriod.value.id,
        data
      )
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('governmentFundings.periodUpdated'),
        life: 3000
      })
    } else {
      const data: GovernmentFundingPeriodCreateRequest = {
        from: formatDateToISO(periodForm.value.from),
        to: periodForm.value.to ? formatDateToISO(periodForm.value.to) : undefined,
        comment: periodForm.value.comment || undefined
      }
      await apiClient.createGovernmentFundingPeriod(governmentFundingId.value, data)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('governmentFundings.periodCreated'),
        life: 3000
      })
    }
    periodDialog.value = false
    await fetchGovernmentFunding()
  } catch {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: t('governmentFundings.failedToSavePeriod'),
      life: 3000
    })
  }
}

function confirmDeletePeriod(period: GovernmentFundingPeriod) {
  confirm.require({
    message: t('governmentFundings.deletePeriodConfirm'),
    header: t('common.confirmDelete'),
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await apiClient.deleteGovernmentFundingPeriod(governmentFundingId.value, period.id)
        toast.add({
          severity: 'success',
          summary: t('common.success'),
          detail: t('governmentFundings.periodDeleted'),
          life: 3000
        })
        await fetchGovernmentFunding()
      } catch {
        toast.add({
          severity: 'error',
          summary: t('common.error'),
          detail: t('governmentFundings.failedToDeletePeriod'),
          life: 3000
        })
      }
    }
  })
}

// Property functions
function openAddProperty(periodId: number) {
  currentPeriodId.value = periodId
  editingProperty.value = null
  propertyForm.value = {
    name: '',
    payment: 0,
    requirement: 0,
    min_age: null,
    max_age: null,
    comment: ''
  }
  propertyDialog.value = true
}

function openEditProperty(periodId: number, property: GovernmentFundingProperty) {
  currentPeriodId.value = periodId
  editingProperty.value = property
  propertyForm.value = {
    name: property.name,
    payment: property.payment,
    requirement: property.requirement,
    min_age: property.min_age ?? null,
    max_age: property.max_age ?? null,
    comment: property.comment || ''
  }
  propertyDialog.value = true
}

async function saveProperty() {
  if (!propertyForm.value.name.trim()) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: t('validation.nameRequired'),
      life: 3000
    })
    return
  }

  // Validate age range if both are provided
  if (propertyForm.value.min_age !== null && propertyForm.value.max_age !== null) {
    if (propertyForm.value.min_age >= propertyForm.value.max_age) {
      toast.add({
        severity: 'error',
        summary: t('common.error'),
        detail: t('validation.maxAgeMustBeGreater'),
        life: 3000
      })
      return
    }
  }

  try {
    if (editingProperty.value && currentPeriodId.value) {
      const data: GovernmentFundingPropertyUpdateRequest = {
        name: propertyForm.value.name,
        payment: propertyForm.value.payment,
        requirement: propertyForm.value.requirement,
        min_age: propertyForm.value.min_age,
        max_age: propertyForm.value.max_age,
        comment: propertyForm.value.comment || undefined
      }
      await apiClient.updateGovernmentFundingProperty(
        governmentFundingId.value,
        currentPeriodId.value,
        editingProperty.value.id,
        data
      )
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('governmentFundings.propertyUpdated'),
        life: 3000
      })
    } else if (currentPeriodId.value) {
      const data: GovernmentFundingPropertyCreateRequest = {
        name: propertyForm.value.name,
        payment: propertyForm.value.payment,
        requirement: propertyForm.value.requirement,
        min_age: propertyForm.value.min_age,
        max_age: propertyForm.value.max_age,
        comment: propertyForm.value.comment || undefined
      }
      await apiClient.createGovernmentFundingProperty(
        governmentFundingId.value,
        currentPeriodId.value,
        data
      )
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('governmentFundings.propertyCreated'),
        life: 3000
      })
    }
    propertyDialog.value = false
    await fetchGovernmentFunding()
  } catch {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: t('governmentFundings.failedToSaveProperty'),
      life: 3000
    })
  }
}

function confirmDeleteProperty(periodId: number, property: GovernmentFundingProperty) {
  confirm.require({
    message: t('governmentFundings.deletePropertyConfirm'),
    header: t('common.confirmDelete'),
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await apiClient.deleteGovernmentFundingProperty(
          governmentFundingId.value,
          periodId,
          property.id
        )
        toast.add({
          severity: 'success',
          summary: t('common.success'),
          detail: t('governmentFundings.propertyDeleted'),
          life: 3000
        })
        await fetchGovernmentFunding()
      } catch {
        toast.add({
          severity: 'error',
          summary: t('common.error'),
          detail: t('governmentFundings.failedToDeleteProperty'),
          life: 3000
        })
      }
    }
  })
}

function formatAgeRange(
  minAge: number | null | undefined,
  maxAge: number | null | undefined
): string {
  if (minAge != null && maxAge != null) {
    return `${minAge} - ${maxAge}`
  } else if (minAge != null) {
    return `${minAge}+`
  } else if (maxAge != null) {
    return `< ${maxAge}`
  }
  return '-'
}

onMounted(() => {
  fetchGovernmentFunding()
})
</script>

<template>
  <div v-if="governmentFunding">
    <div class="page-header">
      <div class="flex align-items-center gap-2">
        <Button
          icon="pi pi-arrow-left"
          text
          @click="router.push({ name: 'government-fundings' })"
        />
        <h1>{{ governmentFunding.name }}</h1>
      </div>
      <div class="flex align-items-center gap-2">
        <SelectButton
          v-model="viewMode"
          :options="viewModeOptions"
          option-label="label"
          option-value="value"
        />
        <Button
          :label="t('governmentFundings.addPeriod')"
          icon="pi pi-plus"
          @click="openAddPeriod"
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
        class="government-funding-table"
      >
        <Column :header="t('governmentFundings.period')" style="min-width: 180px">
          <template #body="{ data }">
            <template v-if="data.isFirstPropertyInPeriod">
              <div class="period-cell">
                <strong>{{ formatDate(data.periodFrom) }}</strong>
                <span> - </span>
                <strong>{{
                  data.periodTo ? formatDate(data.periodTo) : t('common.ongoing')
                }}</strong>
                <div v-if="data.periodComment" class="text-secondary text-sm">
                  {{ data.periodComment }}
                </div>
              </div>
            </template>
          </template>
        </Column>
        <Column
          field="propertyName"
          :header="t('governmentFundings.property')"
          style="width: 120px"
        >
          <template #body="{ data }">
            <span :class="{ 'text-secondary': !data.propertyId }">{{ data.propertyName }}</span>
          </template>
        </Column>
        <Column :header="t('governmentFundings.ageRange')" style="width: 100px">
          <template #body="{ data }">
            <span v-if="data.ageRange !== '-'">
              {{ data.ageRange }} {{ t('governmentFundings.years') }}
            </span>
            <span v-else class="text-secondary">-</span>
          </template>
        </Column>
        <Column :header="t('governmentFundings.payment')" style="width: 120px; text-align: right">
          <template #body="{ data }">
            <span v-if="data.propertyId">{{ formatCurrency(data.payment) }}</span>
            <span v-else class="text-secondary">-</span>
          </template>
        </Column>
        <Column
          :header="t('governmentFundings.requirementFte')"
          style="width: 120px; text-align: right"
        >
          <template #body="{ data }">
            <span v-if="data.propertyId">{{ data.requirement.toFixed(3) }}</span>
            <span v-else class="text-secondary">-</span>
          </template>
        </Column>
        <Column field="propertyComment" :header="t('common.comment')">
          <template #body="{ data }">
            <span class="text-secondary">{{ data.propertyComment }}</span>
          </template>
        </Column>
      </DataTable>
      <p v-else class="text-secondary">
        {{ t('governmentFundings.noDataDefined') }}
      </p>
    </div>

    <!-- Panels View -->
    <div v-else-if="governmentFunding.periods && governmentFunding.periods.length > 0">
      <Panel
        v-for="period in governmentFunding.periods"
        :key="period.id"
        :header="`${formatDate(period.from)} - ${period.to ? formatDate(period.to) : t('common.ongoing')}`"
        toggleable
        class="mb-3"
      >
        <template #icons>
          <Button
            icon="pi pi-plus"
            text
            :title="t('governmentFundings.addProperty')"
            @click.stop="openAddProperty(period.id)"
          />
          <Button
            icon="pi pi-pencil"
            text
            :title="t('governmentFundings.editPeriod')"
            @click.stop="openEditPeriod(period)"
          />
          <Button
            icon="pi pi-trash"
            text
            severity="danger"
            :title="t('governmentFundings.deletePeriod')"
            @click.stop="confirmDeletePeriod(period)"
          />
        </template>

        <p v-if="period.comment" class="text-secondary mb-3">{{ period.comment }}</p>

        <DataTable
          v-if="period.properties && period.properties.length > 0"
          :value="period.properties"
          size="small"
          striped-rows
        >
          <Column field="name" :header="t('common.name')"></Column>
          <Column :header="t('governmentFundings.ageRange')">
            <template #body="{ data }">
              <span v-if="data.min_age != null || data.max_age != null">
                {{ formatAgeRange(data.min_age, data.max_age) }} {{ t('governmentFundings.years') }}
              </span>
              <span v-else class="text-secondary">-</span>
            </template>
          </Column>
          <Column field="payment" :header="t('governmentFundings.payment')">
            <template #body="{ data }">
              {{ formatCurrency(data.payment) }}
            </template>
          </Column>
          <Column field="requirement" :header="t('governmentFundings.requirementFte')">
            <template #body="{ data }">
              {{ data.requirement.toFixed(3) }}
            </template>
          </Column>
          <Column field="comment" :header="t('common.comment')"></Column>
          <Column :header="t('common.actions')" style="width: 100px">
            <template #body="{ data: prop }">
              <Button
                icon="pi pi-pencil"
                text
                rounded
                size="small"
                @click="openEditProperty(period.id, prop)"
              />
              <Button
                icon="pi pi-trash"
                text
                rounded
                size="small"
                severity="danger"
                @click="confirmDeleteProperty(period.id, prop)"
              />
            </template>
          </Column>
        </DataTable>
        <p v-else class="text-secondary">{{ t('governmentFundings.noPropertiesDefined') }}</p>
      </Panel>
      <!-- Show all periods button -->
      <div
        v-if="
          !showAllPeriods &&
          governmentFunding.total_periods &&
          governmentFunding.total_periods > (governmentFunding.periods?.length || 0)
        "
        class="mt-3 text-center"
      >
        <Button
          :label="
            t('governmentFundings.showAllPeriods', { count: governmentFunding.total_periods })
          "
          :loading="loadingAllPeriods"
          text
          @click="loadAllPeriods"
        />
      </div>
    </div>
    <div v-else-if="viewMode === 'panels'" class="card">
      <p class="text-secondary">{{ t('governmentFundings.noPeriodsDefined') }}</p>
    </div>

    <!-- Period Dialog -->
    <Dialog
      v-model:visible="periodDialog"
      :header="
        editingPeriod ? t('governmentFundings.editPeriod') : t('governmentFundings.addPeriod')
      "
      modal
      :style="{ width: '450px' }"
    >
      <div class="form-grid">
        <div class="field">
          <label for="period-from">{{ t('governmentFundings.fromDate') }}</label>
          <Calendar id="period-from" v-model="periodForm.from" date-format="yy-mm-dd" show-icon />
        </div>
        <div class="field">
          <label for="period-to">{{ t('governmentFundings.toDateOptional') }}</label>
          <Calendar
            id="period-to"
            v-model="periodForm.to"
            date-format="yy-mm-dd"
            show-icon
            :show-clear="true"
          />
        </div>
        <div class="field">
          <label for="period-comment">{{ t('common.comment') }}</label>
          <Textarea id="period-comment" v-model="periodForm.comment" rows="2" />
        </div>
      </div>
      <template #footer>
        <Button :label="t('common.cancel')" text @click="periodDialog = false" />
        <Button :label="t('common.save')" @click="savePeriod" />
      </template>
    </Dialog>

    <!-- Property Dialog -->
    <Dialog
      v-model:visible="propertyDialog"
      :header="
        editingProperty ? t('governmentFundings.editProperty') : t('governmentFundings.addProperty')
      "
      modal
      :style="{ width: '450px' }"
    >
      <div class="form-grid">
        <div class="field">
          <label for="property-name">{{ t('common.name') }}</label>
          <InputText
            id="property-name"
            v-model="propertyForm.name"
            placeholder="e.g. ganztag, halbtag"
          />
        </div>
        <div class="field-row">
          <div class="field">
            <label for="property-min-age">{{ t('governmentFundings.minAge') }}</label>
            <InputNumber
              id="property-min-age"
              v-model="propertyForm.min_age"
              :min="0"
              :max="99"
              :show-buttons="true"
              placeholder="Optional"
            />
          </div>
          <div class="field">
            <label for="property-max-age">{{ t('governmentFundings.maxAge') }}</label>
            <InputNumber
              id="property-max-age"
              v-model="propertyForm.max_age"
              :min="1"
              :max="100"
              :show-buttons="true"
              placeholder="Optional"
            />
          </div>
        </div>
        <small class="text-secondary mb-2">{{ t('governmentFundings.ageRangeHelp') }}</small>
        <div class="field">
          <label for="property-payment">{{ t('governmentFundings.paymentInCents') }}</label>
          <InputNumber id="property-payment" v-model="propertyForm.payment" :min="0" />
          <small class="text-secondary">{{ formatCurrency(propertyForm.payment) }}</small>
        </div>
        <div class="field">
          <label for="property-requirement">{{ t('governmentFundings.requirement') }}</label>
          <InputNumber
            id="property-requirement"
            v-model="propertyForm.requirement"
            :min="0"
            :max="10"
            :min-fraction-digits="3"
            :max-fraction-digits="3"
          />
        </div>
        <div class="field">
          <label for="property-comment">{{ t('common.comment') }}</label>
          <InputText id="property-comment" v-model="propertyForm.comment" />
        </div>
      </div>
      <template #footer>
        <Button :label="t('common.cancel')" text @click="propertyDialog = false" />
        <Button :label="t('common.save')" @click="saveProperty" />
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

.mb-2 {
  margin-bottom: 0.5rem;
}

.mb-3 {
  margin-bottom: 1rem;
}

.mt-3 {
  margin-top: 1rem;
}

.text-center {
  text-align: center;
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

.government-funding-table :deep(td) {
  vertical-align: top;
}

.government-funding-table :deep(.p-datatable-tbody > tr > td:nth-child(4)),
.government-funding-table :deep(.p-datatable-tbody > tr > td:nth-child(5)) {
  text-align: right;
}

.card {
  background: var(--surface-card);
  padding: 1.5rem;
  border-radius: var(--border-radius);
  box-shadow: var(--card-shadow);
}

.field-row {
  display: flex;
  gap: 1rem;
}

.field-row .field {
  flex: 1;
}
</style>
