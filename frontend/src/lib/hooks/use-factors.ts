'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiClient } from '@/lib/api/client';
import { useAuthStore } from '@/stores/auth-store';
import { queryKeys } from '@/lib/api/queryKeys';
import type {
  FactorActivateResponse,
  FactorListResponse,
  FactorResponse,
  BackupCodesPayload,
} from '@/lib/api/types';

// useFactors fetches the caller's active factor list. The hook is
// cheap (one query, backed by the REST /users/me/factors endpoint)
// and the result is the exact shape the Settings TwoFactorCard needs.
export function useFactors() {
  // Keyed by who is asking. `/users/me/factors` resolves the user from the
  // session cookie, so the request is right either way -- but the cache entry
  // was not, and a same-tab account switch served the previous user's factors.
  const userId = useAuthStore((s) => s.user?.id);
  return useQuery<FactorListResponse>({
    queryKey: queryKeys.factors.all(userId),
    queryFn: () => apiClient.listMyFactors(),
    // Deliberately not gated on `userId`: the request resolves the user from
    // the session cookie, so it is valid before the store has hydrated. Gating
    // it would mean a failed `loadUser` left the settings page permanently
    // empty. The id is here to separate cache entries, nothing more.
  });
}

// useEnrolTotp posts the password + type=totp, returning the factor
// row + base32 secret + otpauth URI. The caller holds the secret in
// component state while rendering the QR and code-entry form —
// NEVER in localStorage or the query cache. Invalidation intentionally
// skipped: no ACTIVE factor exists until /activate lands, and the
// list query is filtered to active factors only.
export function useEnrolTotp() {
  return useMutation<FactorResponse, Error, { password: string; label?: string }>({
    mutationFn: ({ password, label }) => apiClient.enrolTotp(password, label),
  });
}

// useEnrolWebAuthn starts a WebAuthn registration ceremony. The
// returned FactorResponse carries a `creation_options` payload the
// caller decodes + hands to navigator.credentials.create(). Like
// useEnrolTotp, no invalidation here — activation lands via
// useActivateFactor with a `webauthnResponse` arg.
export function useEnrolWebAuthn() {
  return useMutation<FactorResponse, Error, { password: string; label?: string }>({
    mutationFn: ({ password, label }) => apiClient.enrolWebAuthn(password, label),
  });
}

// useActivateFactor verifies the first code and flips enabled_at. On
// success it returns the BackupCodesPayload (if this is the user's
// first primary factor). The caller keeps the codes in component
// state for the one-shot display, then invalidates the list.
export function useActivateFactor() {
  const queryClient = useQueryClient();
  return useMutation<
    FactorActivateResponse,
    Error,
    { factorId: number; code?: string; webauthnResponse?: unknown }
  >({
    mutationFn: ({ factorId, code, webauthnResponse }) =>
      apiClient.activateFactor(factorId, { code, webauthnResponse }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.factors.root() });
    },
  });
}

// useRegenerateBackupCodes replaces the user's backup code set.
// Password AND a current TOTP/backup code are both required — backup
// code factors only exist post-first-primary, so the step-up gate
// always fires. Closes audit finding A-H-3 (2026-05-01).
export function useRegenerateBackupCodes() {
  const queryClient = useQueryClient();
  return useMutation<
    BackupCodesPayload,
    Error,
    { factorId: number; password: string; code: string }
  >({
    mutationFn: ({ factorId, password, code }) =>
      apiClient.regenerateBackupCodes(factorId, password, code),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.factors.root() });
    },
  });
}

// useDeleteFactor removes a factor. Password is always required;
// `code` is additionally required by the backend when the factor is
// the user's last primary — the dialog always collects it. Returns
// void on 204 success.
export function useDeleteFactor() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, { factorId: number; password: string; code?: string }>({
    mutationFn: ({ factorId, password, code }) => apiClient.deleteFactor(factorId, password, code),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.factors.root() });
    },
  });
}
