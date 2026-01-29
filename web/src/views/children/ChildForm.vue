<script setup lang="ts">
import { watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Child, ChildCreateRequest, Gender } from '@/api/types'
import { childSchema } from '@/validation/schemas'
import { useFormValidation } from '@/composables/useFormValidation'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import DatePicker from 'primevue/datepicker'
import Select from 'primevue/select'
import Button from 'primevue/button'

const { t } = useI18n()

const props = defineProps<{
  visible: boolean
  child: Child | null
}>()

const emit = defineEmits<{
  close: []
  save: [data: Omit<ChildCreateRequest, 'organization_id'>]
}>()

const { values, errors, resetForm, setValues, handleSubmit, hasError } =
  useFormValidation(childSchema)

const genderOptions = computed(() => [
  { value: 'male', label: t('gender.male') },
  { value: 'female', label: t('gender.female') },
  { value: 'diverse', label: t('gender.diverse') }
])

const isEditing = computed(() => !!props.child)
const dialogTitle = computed(() => (isEditing.value ? t('children.edit') : t('children.newChild')))

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      if (props.child) {
        setValues({
          first_name: props.child.first_name,
          last_name: props.child.last_name,
          gender: props.child.gender as Gender,
          birthdate: new Date(props.child.birthdate)
        })
      } else {
        resetForm({
          values: {
            first_name: '',
            last_name: '',
            gender: undefined as unknown as Gender,
            birthdate: undefined as unknown as Date
          }
        })
      }
    }
  }
)

const onSubmit = handleSubmit((formValues) => {
  emit('save', {
    first_name: formValues.first_name,
    last_name: formValues.last_name,
    gender: formValues.gender,
    birthdate: formValues.birthdate.toISOString()
  })
})
</script>

<template>
  <Dialog
    :visible="visible"
    :header="dialogTitle"
    modal
    :closable="true"
    :style="{ width: '450px' }"
    @update:visible="$emit('close')"
  >
    <form @submit.prevent="onSubmit">
      <div class="form-grid">
        <div class="field">
          <label for="first_name">{{ t('children.firstName') }}</label>
          <InputText
            id="first_name"
            v-model="values.first_name"
            :class="{ 'p-invalid': hasError('first_name') }"
            :placeholder="t('children.firstName')"
          />
          <small v-if="errors.first_name" class="p-error">{{ errors.first_name }}</small>
        </div>

        <div class="field">
          <label for="last_name">{{ t('children.lastName') }}</label>
          <InputText
            id="last_name"
            v-model="values.last_name"
            :class="{ 'p-invalid': hasError('last_name') }"
            :placeholder="t('children.lastName')"
          />
          <small v-if="errors.last_name" class="p-error">{{ errors.last_name }}</small>
        </div>

        <div class="field">
          <label for="gender">{{ t('gender.label') }}</label>
          <Select
            id="gender"
            v-model="values.gender"
            :options="genderOptions"
            option-label="label"
            option-value="value"
            :class="{ 'p-invalid': hasError('gender') }"
            :placeholder="t('gender.selectGender')"
          />
          <small v-if="errors.gender" class="p-error">{{ errors.gender }}</small>
        </div>

        <div class="field">
          <label for="birthdate">{{ t('children.birthdate') }}</label>
          <DatePicker
            id="birthdate"
            v-model="values.birthdate"
            date-format="dd.mm.yy"
            :class="{ 'p-invalid': hasError('birthdate') }"
            :placeholder="t('validation.selectBirthdate')"
            show-icon
          />
          <small v-if="errors.birthdate" class="p-error">{{ errors.birthdate }}</small>
        </div>
      </div>
    </form>

    <template #footer>
      <div class="dialog-footer">
        <Button :label="t('common.cancel')" text @click="$emit('close')" />
        <Button :label="t('common.save')" @click="onSubmit" />
      </div>
    </template>
  </Dialog>
</template>
