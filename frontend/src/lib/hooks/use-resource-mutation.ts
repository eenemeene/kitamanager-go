import { useMutation, useQueryClient, type QueryKey } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useToast } from '@/lib/hooks/use-toast';
import { showErrorToast } from '@/lib/utils/show-error-toast';

interface ResourceMutationConfig<TData, TResponse = unknown> {
  /** The mutation function to call. */
  mutationFn: (data: TData) => Promise<TResponse>;
  /** Query key(s) to invalidate on success. Accepts a single key or an array of keys. */
  invalidateQueryKey: QueryKey | QueryKey[];
  /** Toast message shown on success. */
  successMessage: string;
  /** Fallback error message shown on failure. */
  errorMessage: string;
  /** Called after a successful mutation (e.g., close dialog, reset form). */
  onSuccess?: () => void;
  /**
   * Called when the mutation is rejected, before the toast. Returning true
   * suppresses it.
   *
   * Pass `suppressesToast` from lib/forms when the rejection is already being
   * shown on a form. Like useCrudMutations, this hook knows nothing about forms
   * -- marking fields is done by watching the mutation's error, so it works the
   * same way for every mutation in the codebase.
   */
  onMutationError?: (error: unknown) => boolean | void;
}

/**
 * Lightweight mutation hook for nested resource operations on detail pages.
 * Wraps useMutation with automatic query invalidation, success/error toasts,
 * and an optional onSuccess callback for UI cleanup.
 */
export function useResourceMutation<TData, TResponse = unknown>(
  config: ResourceMutationConfig<TData, TResponse>
) {
  const queryClient = useQueryClient();
  const t = useTranslations();
  const { toast } = useToast();

  return useMutation({
    mutationFn: config.mutationFn,
    onSuccess: () => {
      const keys = Array.isArray(config.invalidateQueryKey[0])
        ? (config.invalidateQueryKey as QueryKey[])
        : [config.invalidateQueryKey as QueryKey];
      keys.forEach((key) => queryClient.invalidateQueries({ queryKey: key }));
      toast({ title: config.successMessage });
      config.onSuccess?.();
    },
    onError: (error: unknown) => {
      if (config.onMutationError?.(error) === true) {
        return;
      }
      showErrorToast(t('common.error'), error, config.errorMessage);
    },
  });
}
