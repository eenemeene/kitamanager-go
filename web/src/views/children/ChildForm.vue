<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Child, ChildCreateRequest, Gender } from '@/api/types'
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

const form = ref({
  first_name: '',
  last_name: '',
  gender: null as Gender | null,
  birthdate: null as Date | null
})

const genderOptions = computed(() => [
  { value: 'male', label: t('gender.male') },
  { value: 'female', label: t('gender.female') },
  { value: 'diverse', label: t('gender.diverse') }
])

const errors = ref<{
  first_name?: string
  last_name?: string
  gender?: string
  birthdate?: string
}>({})

const isEditing = computed(() => !!props.child)
const dialogTitle = computed(() => (isEditing.value ? t('children.edit') : t('children.newChild')))

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      if (props.child) {
        form.value = {
          first_name: props.child.first_name,
          last_name: props.child.last_name,
          gender: props.child.gender as Gender,
          birthdate: new Date(props.child.birthdate)
        }
      } else {
        form.value = {
          first_name: '',
          last_name: '',
          gender: null,
          birthdate: null
        }
      }
      errors.value = {}
    }
  }
)

function validate(): boolean {
  errors.value = {}

  if (!form.value.first_name.trim()) {
    errors.value.first_name = t('validation.firstNameRequired')
  }

  if (!form.value.last_name.trim()) {
    errors.value.last_name = t('validation.lastNameRequired')
  }

  if (!form.value.gender) {
    errors.value.gender = t('validation.genderRequired')
  }

  if (!form.value.birthdate) {
    errors.value.birthdate = t('validation.birthdateRequired')
  }

  return Object.keys(errors.value).length === 0
}

function onSubmit() {
  if (!validate()) return

  emit('save', {
    first_name: form.value.first_name,
    last_name: form.value.last_name,
    gender: form.value.gender!,
    birthdate: form.value.birthdate!.toISOString()
  })
}
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
            v-model="form.first_name"
            :class="{ 'p-invalid': errors.first_name }"
            :placeholder="t('children.firstName')"
          />
          <small v-if="errors.first_name" class="p-error">{{ errors.first_name }}</small>
        </div>

        <div class="field">
          <label for="last_name">{{ t('children.lastName') }}</label>
          <InputText
            id="last_name"
            v-model="form.last_name"
            :class="{ 'p-invalid': errors.last_name }"
            :placeholder="t('children.lastName')"
          />
          <small v-if="errors.last_name" class="p-error">{{ errors.last_name }}</small>
        </div>

        <div class="field">
          <label for="gender">{{ t('gender.label') }}</label>
          <Select
            id="gender"
            v-model="form.gender"
            :options="genderOptions"
            option-label="label"
            option-value="value"
            :class="{ 'p-invalid': errors.gender }"
            :placeholder="t('gender.selectGender')"
          />
          <small v-if="errors.gender" class="p-error">{{ errors.gender }}</small>
        </div>

        <div class="field">
          <label for="birthdate">{{ t('children.birthdate') }}</label>
          <DatePicker
            id="birthdate"
            v-model="form.birthdate"
            date-format="dd.mm.yy"
            :class="{ 'p-invalid': errors.birthdate }"
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
