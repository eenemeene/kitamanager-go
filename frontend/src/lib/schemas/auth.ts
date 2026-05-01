import { z } from 'zod';
import type { LoginRequest } from '@/lib/api/types';

export const loginSchema = z.object({
  email: z.string().email(),
  password: z.string().min(1),
}) satisfies z.ZodType<LoginRequest>;

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

// Step-up password re-entry — kept for callers that legitimately need
// only the password (the very first factor enrolment, when the user
// has no primary factor to verify against yet).
export const factorPasswordStepUpSchema = z.object({
  password: z.string().min(1, 'Password is required'),
});

export type FactorPasswordStepUpFormData = z.infer<typeof factorPasswordStepUpSchema>;

// Step-up password + current MFA code. The backend requires both for
// every factor mutation (enrol/delete/regenerate) once the user has at
// least one active primary factor — see security audit findings
// A-H-1/2/3 (2026-05-01). The UI surfaces this as two inputs.
export const factorStepUpWithCodeSchema = z.object({
  password: z.string().min(1, 'Password is required'),
  code: z.string().trim().min(1, 'Code is required').max(32, 'Code is too long'),
});

export type FactorStepUpWithCodeFormData = z.infer<typeof factorStepUpWithCodeSchema>;

// Disable-factor (delete) form: same shape as factorStepUpWithCodeSchema.
// Kept as a separate alias so the dialog imports stay descriptive.
export const factorDisableSchema = factorStepUpWithCodeSchema;
export type FactorDisableFormData = FactorStepUpWithCodeFormData;

// Enrol-form is the first-enrolment shape: password only. The
// has-primary path uses factorStepUpWithCodeSchema instead.
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
