import { z } from 'zod';

export const loginSchema = z.object({
  email: z.string().email(),
  password: z.string().min(1),
});

export type LoginFormData = z.infer<typeof loginSchema>;

// mfaVerifySchema validates the code the user types on the login
// MFA step. TOTP codes are 6 digits; backup codes are hyphenated
// alphanumeric strings the user pastes. Accept either shape by
// keeping the regex permissive — the server is the ground truth for
// what counts as a valid code, so an overly strict client check
// would just block legitimate backup-code attempts.
export const mfaVerifySchema = z.object({
  code: z.string().trim().min(1, 'Code is required').max(32, 'Code is too long'),
});

export type MfaVerifyFormData = z.infer<typeof mfaVerifySchema>;

// Step-up password re-entry for enrol / regenerate.
export const factorPasswordStepUpSchema = z.object({
  password: z.string().min(1, 'Password is required'),
});

export type FactorPasswordStepUpFormData = z.infer<typeof factorPasswordStepUpSchema>;

// Disable-factor form: password + code, both required because the
// backend service enforces password + a valid code from any active
// factor when the user is deleting their last primary factor. The
// UI collects both always — if the server rejects the code as
// unnecessary (not last primary) the call still succeeds.
export const factorDisableSchema = z.object({
  password: z.string().min(1, 'Password is required'),
  code: z.string().trim().min(1, 'Code is required').max(32, 'Code is too long'),
});

export type FactorDisableFormData = z.infer<typeof factorDisableSchema>;

// Enrol TOTP: a single password-only step. Label is optional — the
// server accepts null — but we keep it off the form for now to
// minimise the enrolment flow's complexity. A future label-editor
// can live in the factor list entry itself.
export const factorEnrolSchema = z.object({
  password: z.string().min(1, 'Password is required'),
});

export type FactorEnrolFormData = z.infer<typeof factorEnrolSchema>;

// Activation code form — posted to /activate right after the user
// has scanned the QR and read their first 6-digit code off the
// authenticator. Same permissive regex as mfaVerifySchema so a
// copy-paste with surrounding whitespace works.
export const factorActivateSchema = z.object({
  code: z.string().trim().min(1, 'Code is required').max(32, 'Code is too long'),
});

export type FactorActivateFormData = z.infer<typeof factorActivateSchema>;
