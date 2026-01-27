<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { Child, ChildContractCreateRequest } from '@/api/types'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import DatePicker from 'primevue/datepicker'
import Checkbox from 'primevue/checkbox'
import Button from 'primevue/button'
import Chips from 'primevue/chips'

const props = defineProps<{
  visible: boolean
  child: Child | null
}>()

const emit = defineEmits<{
  close: []
  save: [data: ChildContractCreateRequest]
}>()

const form = ref({
  from: null as Date | null,
  to: null as Date | null,
  care_hours_per_week: 35,
  group_id: null as number | null,
  meals_included: true,
  special_needs: '',
  attributes: [] as string[]
})

const errors = ref<{
  from?: string
  care_hours_per_week?: string
}>({})

const dialogTitle = computed(() =>
  props.child
    ? `New Contract for ${props.child.first_name} ${props.child.last_name}`
    : 'New Contract'
)

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      form.value = {
        from: new Date(),
        to: null,
        care_hours_per_week: 35,
        group_id: null,
        meals_included: true,
        special_needs: '',
        attributes: []
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

  if (!form.value.care_hours_per_week || form.value.care_hours_per_week <= 0) {
    errors.value.care_hours_per_week = 'Care hours must be greater than 0'
  }

  return Object.keys(errors.value).length === 0
}

function handleSave() {
  if (validate()) {
    emit('save', {
      from: form.value.from!.toISOString().split('T')[0],
      to: form.value.to ? form.value.to.toISOString().split('T')[0] : null,
      care_hours_per_week: form.value.care_hours_per_week,
      group_id: form.value.group_id,
      meals_included: form.value.meals_included,
      special_needs: form.value.special_needs,
      attributes: form.value.attributes
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
        <label for="care_hours">Care Hours per Week</label>
        <InputNumber
          id="care_hours"
          v-model="form.care_hours_per_week"
          :class="{ 'p-invalid': errors.care_hours_per_week }"
          :min="0"
          :max="60"
          suffix=" h"
        />
        <small v-if="errors.care_hours_per_week" class="p-error">{{
          errors.care_hours_per_week
        }}</small>
      </div>

      <div class="field">
        <div class="flex align-items-center gap-2">
          <Checkbox v-model="form.meals_included" input-id="meals" :binary="true" />
          <label for="meals">Meals Included</label>
        </div>
      </div>

      <div class="field">
        <label for="special_needs">Special Needs</label>
        <InputText
          id="special_needs"
          v-model="form.special_needs"
          placeholder="Any special requirements"
        />
      </div>

      <div class="field">
        <label for="attributes">Attributes (for funding calculation)</label>
        <Chips
          id="attributes"
          v-model="form.attributes"
          placeholder="e.g. ganztag, ndh, integration"
        />
        <small class="text-secondary">Press Enter to add each attribute</small>
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

.text-secondary {
  color: var(--text-color-secondary);
}
</style>
