<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { apiClient } from '@/api/client'
import type {
  GovernmentFunding,
  GovernmentFundingPeriod,
  GovernmentFundingEntry,
  GovernmentFundingProperty,
  GovernmentFundingPeriodCreateRequest,
  GovernmentFundingEntryCreateRequest,
  GovernmentFundingPropertyCreateRequest,
  GovernmentFundingPeriodUpdateRequest,
  GovernmentFundingEntryUpdateRequest,
  GovernmentFundingPropertyUpdateRequest
} from '@/api/types'
import { flattenGovernmentFundingToRows } from '@/utils/government-funding'
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

const governmentFundingId = computed(() => Number(route.params.id))
const governmentFunding = ref<GovernmentFunding | null>(null)
const loading = ref(false)

// View mode toggle
const viewMode = ref<'panels' | 'table'>('panels')
const viewModeOptions = [
  { label: 'Panels', value: 'panels', icon: 'pi pi-list' },
  { label: 'Table', value: 'table', icon: 'pi pi-table' }
]

// Compute flattened rows for table view using utility function
const flattenedRows = computed(() => flattenGovernmentFundingToRows(governmentFunding.value))

// Dialog states
const periodDialog = ref(false)
const entryDialog = ref(false)
const propertyDialog = ref(false)

// Current editing contexts
const editingPeriod = ref<GovernmentFundingPeriod | null>(null)
const editingEntry = ref<GovernmentFundingEntry | null>(null)
const editingProperty = ref<GovernmentFundingProperty | null>(null)
const currentPeriodId = ref<number | null>(null)
const currentEntryId = ref<number | null>(null)

// Form data
const periodForm = ref({
  from: null as Date | null,
  to: null as Date | null | undefined,
  comment: ''
})

const entryForm = ref({
  min_age: 0,
  max_age: 1
})

const propertyForm = ref({
  name: '',
  payment: 0,
  requirement: 0,
  comment: ''
})

async function fetchGovernmentFunding() {
  loading.value = true
  try {
    governmentFunding.value = await apiClient.getGovernmentFunding(governmentFundingId.value)
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to load government funding',
      life: 3000
    })
    router.push({ name: 'government-fundings' })
  } finally {
    loading.value = false
  }
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
    toast.add({ severity: 'error', summary: 'Error', detail: 'From date is required', life: 3000 })
    return
  }

  const formatDate = (d: Date) => d.toISOString().split('T')[0]

  try {
    if (editingPeriod.value) {
      const data: GovernmentFundingPeriodUpdateRequest = {
        from: formatDate(periodForm.value.from),
        to: periodForm.value.to ? formatDate(periodForm.value.to) : null,
        comment: periodForm.value.comment || undefined
      }
      await apiClient.updateGovernmentFundingPeriod(
        governmentFundingId.value,
        editingPeriod.value.id,
        data
      )
      toast.add({ severity: 'success', summary: 'Success', detail: 'Period updated', life: 3000 })
    } else {
      const data: GovernmentFundingPeriodCreateRequest = {
        from: formatDate(periodForm.value.from),
        to: periodForm.value.to ? formatDate(periodForm.value.to) : undefined,
        comment: periodForm.value.comment || undefined
      }
      await apiClient.createGovernmentFundingPeriod(governmentFundingId.value, data)
      toast.add({ severity: 'success', summary: 'Success', detail: 'Period created', life: 3000 })
    }
    periodDialog.value = false
    await fetchGovernmentFunding()
  } catch {
    toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to save period', life: 3000 })
  }
}

function confirmDeletePeriod(period: GovernmentFundingPeriod) {
  confirm.require({
    message:
      'Are you sure you want to delete this period? This will also delete all entries and properties.',
    header: 'Confirm Delete',
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await apiClient.deleteGovernmentFundingPeriod(governmentFundingId.value, period.id)
        toast.add({ severity: 'success', summary: 'Success', detail: 'Period deleted', life: 3000 })
        await fetchGovernmentFunding()
      } catch {
        toast.add({
          severity: 'error',
          summary: 'Error',
          detail: 'Failed to delete period',
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
  entryForm.value = { min_age: 0, max_age: 1 }
  entryDialog.value = true
}

function openEditEntry(periodId: number, entry: GovernmentFundingEntry) {
  currentPeriodId.value = periodId
  editingEntry.value = entry
  entryForm.value = { min_age: entry.min_age, max_age: entry.max_age }
  entryDialog.value = true
}

async function saveEntry() {
  if (entryForm.value.min_age >= entryForm.value.max_age) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Max age must be greater than min age',
      life: 3000
    })
    return
  }

  try {
    if (editingEntry.value && currentPeriodId.value) {
      const data: GovernmentFundingEntryUpdateRequest = {
        min_age: entryForm.value.min_age,
        max_age: entryForm.value.max_age
      }
      await apiClient.updateGovernmentFundingEntry(
        governmentFundingId.value,
        currentPeriodId.value,
        editingEntry.value.id,
        data
      )
      toast.add({ severity: 'success', summary: 'Success', detail: 'Entry updated', life: 3000 })
    } else if (currentPeriodId.value) {
      const data: GovernmentFundingEntryCreateRequest = {
        min_age: entryForm.value.min_age,
        max_age: entryForm.value.max_age
      }
      await apiClient.createGovernmentFundingEntry(
        governmentFundingId.value,
        currentPeriodId.value,
        data
      )
      toast.add({ severity: 'success', summary: 'Success', detail: 'Entry created', life: 3000 })
    }
    entryDialog.value = false
    await fetchGovernmentFunding()
  } catch {
    toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to save entry', life: 3000 })
  }
}

function confirmDeleteEntry(periodId: number, entry: GovernmentFundingEntry) {
  confirm.require({
    message: 'Are you sure you want to delete this entry? This will also delete all properties.',
    header: 'Confirm Delete',
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await apiClient.deleteGovernmentFundingEntry(governmentFundingId.value, periodId, entry.id)
        toast.add({ severity: 'success', summary: 'Success', detail: 'Entry deleted', life: 3000 })
        await fetchGovernmentFunding()
      } catch {
        toast.add({
          severity: 'error',
          summary: 'Error',
          detail: 'Failed to delete entry',
          life: 3000
        })
      }
    }
  })
}

// Property functions
function openAddProperty(periodId: number, entryId: number) {
  currentPeriodId.value = periodId
  currentEntryId.value = entryId
  editingProperty.value = null
  propertyForm.value = { name: '', payment: 0, requirement: 0, comment: '' }
  propertyDialog.value = true
}

function openEditProperty(periodId: number, entryId: number, property: GovernmentFundingProperty) {
  currentPeriodId.value = periodId
  currentEntryId.value = entryId
  editingProperty.value = property
  propertyForm.value = {
    name: property.name,
    payment: property.payment,
    requirement: property.requirement,
    comment: property.comment || ''
  }
  propertyDialog.value = true
}

async function saveProperty() {
  if (!propertyForm.value.name.trim()) {
    toast.add({ severity: 'error', summary: 'Error', detail: 'Name is required', life: 3000 })
    return
  }

  try {
    if (editingProperty.value && currentPeriodId.value && currentEntryId.value) {
      const data: GovernmentFundingPropertyUpdateRequest = {
        name: propertyForm.value.name,
        payment: propertyForm.value.payment,
        requirement: propertyForm.value.requirement,
        comment: propertyForm.value.comment || undefined
      }
      await apiClient.updateGovernmentFundingProperty(
        governmentFundingId.value,
        currentPeriodId.value,
        currentEntryId.value,
        editingProperty.value.id,
        data
      )
      toast.add({ severity: 'success', summary: 'Success', detail: 'Property updated', life: 3000 })
    } else if (currentPeriodId.value && currentEntryId.value) {
      const data: GovernmentFundingPropertyCreateRequest = {
        name: propertyForm.value.name,
        payment: propertyForm.value.payment,
        requirement: propertyForm.value.requirement,
        comment: propertyForm.value.comment || undefined
      }
      await apiClient.createGovernmentFundingProperty(
        governmentFundingId.value,
        currentPeriodId.value,
        currentEntryId.value,
        data
      )
      toast.add({ severity: 'success', summary: 'Success', detail: 'Property created', life: 3000 })
    }
    propertyDialog.value = false
    await fetchGovernmentFunding()
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to save property',
      life: 3000
    })
  }
}

function confirmDeleteProperty(
  periodId: number,
  entryId: number,
  property: GovernmentFundingProperty
) {
  confirm.require({
    message: 'Are you sure you want to delete this property?',
    header: 'Confirm Delete',
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      try {
        await apiClient.deleteGovernmentFundingProperty(
          governmentFundingId.value,
          periodId,
          entryId,
          property.id
        )
        toast.add({
          severity: 'success',
          summary: 'Success',
          detail: 'Property deleted',
          life: 3000
        })
        await fetchGovernmentFunding()
      } catch {
        toast.add({
          severity: 'error',
          summary: 'Error',
          detail: 'Failed to delete property',
          life: 3000
        })
      }
    }
  })
}

function formatCurrency(cents: number): string {
  return (cents / 100).toLocaleString('de-DE', { style: 'currency', currency: 'EUR' })
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('en-GB', {
    day: 'numeric',
    month: 'short',
    year: 'numeric'
  })
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
        <Button label="Add Period" icon="pi pi-plus" @click="openAddPeriod" />
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
        <Column header="Period" style="min-width: 180px">
          <template #body="{ data }">
            <template v-if="data.isFirstEntryInPeriod">
              <div class="period-cell">
                <strong>{{ formatDate(data.periodFrom) }}</strong>
                <span> - </span>
                <strong>{{ data.periodTo ? formatDate(data.periodTo) : 'ongoing' }}</strong>
                <div v-if="data.periodComment" class="text-secondary text-sm">
                  {{ data.periodComment }}
                </div>
              </div>
            </template>
          </template>
        </Column>
        <Column header="Age Range" style="width: 120px">
          <template #body="{ data }">
            <template v-if="data.isFirstPropertyInEntry && data.entryId">
              {{ data.ageRange }} years
            </template>
            <template v-else-if="!data.entryId">
              <span class="text-secondary">-</span>
            </template>
          </template>
        </Column>
        <Column field="propertyName" header="Property" style="width: 120px">
          <template #body="{ data }">
            <span :class="{ 'text-secondary': !data.propertyId }">{{ data.propertyName }}</span>
          </template>
        </Column>
        <Column header="Payment" style="width: 120px; text-align: right">
          <template #body="{ data }">
            <span v-if="data.propertyId">{{ formatCurrency(data.payment) }}</span>
            <span v-else class="text-secondary">-</span>
          </template>
        </Column>
        <Column header="Requirement (FTE)" style="width: 120px; text-align: right">
          <template #body="{ data }">
            <span v-if="data.propertyId">{{ data.requirement.toFixed(3) }}</span>
            <span v-else class="text-secondary">-</span>
          </template>
        </Column>
        <Column field="propertyComment" header="Comment">
          <template #body="{ data }">
            <span class="text-secondary">{{ data.propertyComment }}</span>
          </template>
        </Column>
      </DataTable>
      <p v-else class="text-secondary">
        No data defined. Switch to Panels view to add periods, entries, and properties.
      </p>
    </div>

    <!-- Panels View -->
    <div v-else-if="governmentFunding.periods && governmentFunding.periods.length > 0">
      <Panel
        v-for="period in governmentFunding.periods"
        :key="period.id"
        :header="`${formatDate(period.from)} - ${period.to ? formatDate(period.to) : 'ongoing'}`"
        toggleable
        class="mb-3"
      >
        <template #icons>
          <Button icon="pi pi-plus" text title="Add Entry" @click.stop="openAddEntry(period.id)" />
          <Button
            icon="pi pi-pencil"
            text
            title="Edit Period"
            @click.stop="openEditPeriod(period)"
          />
          <Button
            icon="pi pi-trash"
            text
            severity="danger"
            title="Delete Period"
            @click.stop="confirmDeletePeriod(period)"
          />
        </template>

        <p v-if="period.comment" class="text-secondary mb-3">{{ period.comment }}</p>

        <div v-if="period.entries && period.entries.length > 0">
          <Panel
            v-for="entry in period.entries"
            :key="entry.id"
            :header="`Age ${entry.min_age} - ${entry.max_age} years`"
            toggleable
            class="mb-2"
          >
            <template #icons>
              <Button
                icon="pi pi-plus"
                text
                title="Add Property"
                @click.stop="openAddProperty(period.id, entry.id)"
              />
              <Button
                icon="pi pi-pencil"
                text
                title="Edit Entry"
                @click.stop="openEditEntry(period.id, entry)"
              />
              <Button
                icon="pi pi-trash"
                text
                severity="danger"
                title="Delete Entry"
                @click.stop="confirmDeleteEntry(period.id, entry)"
              />
            </template>

            <DataTable
              v-if="entry.properties && entry.properties.length > 0"
              :value="entry.properties"
              size="small"
              striped-rows
            >
              <Column field="name" header="Name"></Column>
              <Column field="payment" header="Payment">
                <template #body="{ data }">
                  {{ formatCurrency(data.payment) }}
                </template>
              </Column>
              <Column field="requirement" header="Requirement (FTE)">
                <template #body="{ data }">
                  {{ data.requirement.toFixed(3) }}
                </template>
              </Column>
              <Column field="comment" header="Comment"></Column>
              <Column header="Actions" style="width: 100px">
                <template #body="{ data: prop }">
                  <Button
                    icon="pi pi-pencil"
                    text
                    rounded
                    size="small"
                    @click="openEditProperty(period.id, entry.id, prop)"
                  />
                  <Button
                    icon="pi pi-trash"
                    text
                    rounded
                    size="small"
                    severity="danger"
                    @click="confirmDeleteProperty(period.id, entry.id, prop)"
                  />
                </template>
              </Column>
            </DataTable>
            <p v-else class="text-secondary">No properties defined. Click + to add one.</p>
          </Panel>
        </div>
        <p v-else class="text-secondary">No entries defined. Click + to add age ranges.</p>
      </Panel>
    </div>
    <div v-else-if="viewMode === 'panels'" class="card">
      <p class="text-secondary">No periods defined. Click "Add Period" to get started.</p>
    </div>

    <!-- Period Dialog -->
    <Dialog
      v-model:visible="periodDialog"
      :header="editingPeriod ? 'Edit Period' : 'Add Period'"
      modal
      :style="{ width: '450px' }"
    >
      <div class="form-grid">
        <div class="field">
          <label for="period-from">From Date</label>
          <Calendar id="period-from" v-model="periodForm.from" date-format="yy-mm-dd" show-icon />
        </div>
        <div class="field">
          <label for="period-to">To Date (leave empty for ongoing)</label>
          <Calendar
            id="period-to"
            v-model="periodForm.to"
            date-format="yy-mm-dd"
            show-icon
            :show-clear="true"
          />
        </div>
        <div class="field">
          <label for="period-comment">Comment</label>
          <Textarea id="period-comment" v-model="periodForm.comment" rows="2" />
        </div>
      </div>
      <template #footer>
        <Button label="Cancel" text @click="periodDialog = false" />
        <Button label="Save" @click="savePeriod" />
      </template>
    </Dialog>

    <!-- Entry Dialog -->
    <Dialog
      v-model:visible="entryDialog"
      :header="editingEntry ? 'Edit Entry' : 'Add Entry'"
      modal
      :style="{ width: '400px' }"
    >
      <div class="form-grid">
        <div class="field">
          <label for="entry-min-age">Min Age (inclusive)</label>
          <InputNumber id="entry-min-age" v-model="entryForm.min_age" :min="0" :max="99" />
        </div>
        <div class="field">
          <label for="entry-max-age">Max Age (exclusive)</label>
          <InputNumber id="entry-max-age" v-model="entryForm.max_age" :min="1" :max="100" />
        </div>
      </div>
      <template #footer>
        <Button label="Cancel" text @click="entryDialog = false" />
        <Button label="Save" @click="saveEntry" />
      </template>
    </Dialog>

    <!-- Property Dialog -->
    <Dialog
      v-model:visible="propertyDialog"
      :header="editingProperty ? 'Edit Property' : 'Add Property'"
      modal
      :style="{ width: '450px' }"
    >
      <div class="form-grid">
        <div class="field">
          <label for="property-name">Name</label>
          <InputText
            id="property-name"
            v-model="propertyForm.name"
            placeholder="e.g. ganztag, halbtag"
          />
        </div>
        <div class="field">
          <label for="property-payment">Payment (in cents)</label>
          <InputNumber id="property-payment" v-model="propertyForm.payment" :min="0" />
          <small class="text-secondary">{{ formatCurrency(propertyForm.payment) }}</small>
        </div>
        <div class="field">
          <label for="property-requirement">Requirement (staffing ratio)</label>
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
          <label for="property-comment">Comment</label>
          <InputText id="property-comment" v-model="propertyForm.comment" />
        </div>
      </div>
      <template #footer>
        <Button label="Cancel" text @click="propertyDialog = false" />
        <Button label="Save" @click="saveProperty" />
      </template>
    </Dialog>
  </div>
  <div v-else-if="loading" class="card">
    <p>Loading...</p>
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
</style>
