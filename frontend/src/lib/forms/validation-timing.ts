/**
 * When a form tells the user something is wrong.
 *
 * Spread into `useForm` so the decision lives in one place rather than being
 * re-made, or forgotten, at each of the twenty-odd call sites:
 *
 *     useForm<ChildFormData>({ ...validationTiming, resolver: zodResolver(schema) })
 *
 * # Validate late, re-validate early
 *
 * `onTouched` runs validation when the user leaves a field, and `onChange`
 * re-runs it from then on. So a field is judged once the user has finished with
 * it, and the error clears as they fix it rather than after another submit.
 *
 * The two defaults this replaces are both worse in opposite directions.
 * React Hook Form's own default, `onSubmit`, says nothing until the whole form
 * is submitted — the user finds out about the first field after filling in ten.
 * Validating on every keystroke tells them their email address is invalid after
 * they have typed one character, which is the pattern usability research
 * consistently finds harmful: the user is told they are wrong before they have
 * had a chance to be right.
 *
 * # Where this is deliberately not used
 *
 * The MFA, password and WebAuthn forms keep `mode: 'onChange'`. They gate a
 * submit button on `formState.isValid`, and with `onTouched` that stays false
 * until the field is blurred — so someone who types a six-digit code and reaches
 * straight for the button finds it disabled. Those forms are short, their fields
 * are fixed-length, and there is no "told you are wrong too early" problem to
 * solve; the live validity is the point. Each carries a comment saying so, to
 * stop this being tidied into consistency later.
 */
export const validationTiming = {
  mode: 'onTouched',
  reValidateMode: 'onChange',
  /**
   * The error summary takes focus after a rejected submit, not the first bad
   * input.
   *
   * React Hook Form focuses the first invalid field by default, which competes
   * with the summary and wins — and on a phone or a tablet in portrait it also
   * summons the on-screen keyboard, taking roughly 40% of the viewport and
   * potentially hiding the field it just focused. The summary is a div, so
   * focusing it announces the problem to a screen reader and scrolls to the top
   * of the form without opening anything.
   *
   * Forms that show no summary keep react-hook-form's behaviour by not
   * spreading this.
   */
  shouldFocusError: false,
} as const;
