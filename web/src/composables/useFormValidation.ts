import { toTypedSchema } from '@vee-validate/zod'
import { useForm } from 'vee-validate'
import type { ZodType } from 'zod'
import { useI18n } from 'vue-i18n'
import { computed } from 'vue'

/**
 * Composable for form validation using vee-validate with Zod schemas.
 * Automatically translates error messages using i18n keys.
 *
 * @param schema - Zod schema for validation
 * @param initialValues - Optional initial form values
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function useFormValidation<TValues extends Record<string, any>>(
  schema: ZodType<TValues>,
  initialValues?: TValues
) {
  const { t } = useI18n()
  const typedSchema = toTypedSchema(schema)

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const form = useForm<any>({
    validationSchema: typedSchema,
    initialValues
  })

  // Translate error messages (they contain i18n keys)
  const errors = computed(() => {
    const translated: Record<string, string | undefined> = {}
    const rawErrs = form.errors.value as Record<string, string | undefined>
    for (const key in rawErrs) {
      const value = rawErrs[key]
      translated[key] = value ? t(value) : undefined
    }
    return translated
  })

  // Helper to check if a specific field has an error
  function hasError(field: string): boolean {
    const rawErrs = form.errors.value as Record<string, string | undefined>
    return !!rawErrs[field]
  }

  // Helper to get error for a specific field (translated)
  function getError(field: string): string | undefined {
    const rawErrs = form.errors.value as Record<string, string | undefined>
    const error = rawErrs[field]
    return error ? t(error) : undefined
  }

  return {
    values: form.values as TValues,
    errors,
    rawErrors: form.errors,
    defineField: form.defineField,
    handleSubmit: form.handleSubmit,
    resetForm: form.resetForm,
    setFieldValue: form.setFieldValue,
    setValues: form.setValues,
    validate: form.validate,
    meta: form.meta,
    hasError,
    getError
  }
}
