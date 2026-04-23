'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiClient } from '@/lib/api/client';
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
  return useQuery<FactorListResponse>({
    queryKey: queryKeys.factors.all(),
    queryFn: () => apiClient.listMyFactors(),
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

// useActivateFactor verifies the first code and flips enabled_at. On
// success it returns the BackupCodesPayload (if this is the user's
// first primary factor). The caller keeps the codes in component
// state for the one-shot display, then invalidates the list.
export function useActivateFactor() {
  const queryClient = useQueryClient();
  return useMutation<FactorActivateResponse, Error, { factorId: number; code: string }>({
    mutationFn: ({ factorId, code }) => apiClient.activateFactor(factorId, code),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.factors.all() });
    },
  });
}

// useRegenerateBackupCodes replaces the user's backup code set.
// Password step-up required. Returns the new codes for one-time
// display. We invalidate on success so the factor list refreshes
// any backup_codes_remaining counter the UI shows.
export function useRegenerateBackupCodes() {
  const queryClient = useQueryClient();
  return useMutation<BackupCodesPayload, Error, { factorId: number; password: string }>({
    mutationFn: ({ factorId, password }) => apiClient.regenerateBackupCodes(factorId, password),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.factors.all() });
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
      void queryClient.invalidateQueries({ queryKey: queryKeys.factors.all() });
    },
  });
}
