<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { Employee, EmployeeContractCreate } from '@/api/types'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import DatePicker from 'primevue/datepicker'
import Button from 'primevue/button'

const props = defineProps<{
  visible: boolean
  employee: Employee | null
}>()

const emit = defineEmits<{
  close: []
  save: [data: EmployeeContractCreate]
}>()

const form = ref({
  from: null as Date | null,
  to: null as Date | null,
  position: '',
  weekly_hours: 40,
  salary: 0
})

const errors = ref<{
  from?: string
  position?: string
  weekly_hours?: string
  salary?: string
}>({})

const dialogTitle = computed(() =>
  props.employee
    ? `New Contract for ${props.employee.first_name} ${props.employee.last_name}`
    : 'New Contract'
)

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      form.value = {
        from: new Date(),
        to: null,
        position: '',
        weekly_hours: 40,
        salary: 0
      }
      errors.value = {}
    }
  }
)

function validate(): boolean {
  errors.value = {}

  if (!form.value.from) {
    errors.value.from = 'Start date is required'
  }

  if (!form.value.position.trim()) {
    errors.value.position = 'Position is required'
  }

  if (!form.value.weekly_hours || form.value.weekly_hours <= 0) {
    errors.value.weekly_hours = 'Weekly hours must be greater than 0'
  }

  if (!form.value.salary || form.value.salary <= 0) {
    errors.value.salary = 'Salary must be greater than 0'
  }

  return Object.keys(errors.value).length === 0
}

function handleSave() {
  if (validate()) {
    emit('save', {
      from: form.value.from!.toISOString().split('T')[0],
      to: form.value.to ? form.value.to.toISOString().split('T')[0] : null,
      position: form.value.position,
      weekly_hours: form.value.weekly_hours,
      salary: Math.round(form.value.salary * 100) // Convert to cents
    })
  }
}
</script>

<template>
  <Dialog
    :visible="visible"
    :header="dialogTitle"
    modal
    :closable="true"
    :style="{ width: '500px' }"
    @update:visible="$emit('close')"
  >
    <div class="form-grid">
      <div class="field">
        <label for="from">Start Date</label>
        <DatePicker
          id="from"
          v-model="form.from"
          date-format="dd.mm.yy"
          :class="{ 'p-invalid': errors.from }"
          placeholder="Contract start date"
          show-icon
        />
        <small v-if="errors.from" class="p-error">{{ errors.from }}</small>
      </div>

      <div class="field">
        <label for="to">End Date (optional)</label>
        <DatePicker
          id="to"
          v-model="form.to"
          date-format="dd.mm.yy"
          placeholder="Contract end date"
          show-icon
        />
      </div>

      <div class="field">
        <label for="position">Position</label>
        <InputText
          id="position"
          v-model="form.position"
          :class="{ 'p-invalid': errors.position }"
          placeholder="e.g., Erzieher"
        />
        <small v-if="errors.position" class="p-error">{{ errors.position }}</small>
      </div>

      <div class="field">
        <label for="weekly_hours">Weekly Hours</label>
        <InputNumber
          id="weekly_hours"
          v-model="form.weekly_hours"
          :class="{ 'p-invalid': errors.weekly_hours }"
          :min="0"
          :max="60"
          suffix=" h"
        />
        <small v-if="errors.weekly_hours" class="p-error">{{ errors.weekly_hours }}</small>
      </div>

      <div class="field">
        <label for="salary">Monthly Salary</label>
        <InputNumber
          id="salary"
          v-model="form.salary"
          :class="{ 'p-invalid': errors.salary }"
          mode="currency"
          currency="EUR"
          locale="de-DE"
        />
        <small v-if="errors.salary" class="p-error">{{ errors.salary }}</small>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <Button label="Cancel" text @click="$emit('close')" />
        <Button label="Save" @click="handleSave" />
      </div>
    </template>
  </Dialog>
</template>
