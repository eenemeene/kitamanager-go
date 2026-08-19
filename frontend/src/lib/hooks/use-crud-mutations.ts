'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useMutationFeedback } from './use-mutation-feedback';

export interface UseCrudMutationsConfig<TItem, TCreate, TUpdate> {
  /** Resource name for i18n keys (e.g., 'groups', 'organizations') */
  resourceName: string;
  /** Query key to invalidate on success */
  queryKey: readonly (string | number | undefined)[];
  /** Additional query keys to invalidate on success (e.g., related statistics) */
  extraInvalidateKeys?: readonly (string | number | undefined)[][];
  /** Function to create a new item */
  createFn?: (data: TCreate) => Promise<TItem>;
  /** Function to update an existing item */
  updateFn?: (id: number, data: TUpdate) => Promise<TItem>;
  /** Function to delete an item */
  deleteFn?: (id: number) => Promise<void>;
  /** Callback when any mutation succeeds (e.g., close dialog, reset form) */
  onSuccess?: () => void;
  /** Callback when create mutation succeeds */
  onCreateSuccess?: (item: TItem) => void;
  /** Callback when update mutation succeeds */
  onUpdateSuccess?: (item: TItem) => void;
  /** Callback when delete mutation succeeds */
  onDeleteSuccess?: () => void;
  /**
   * Called when a create or update is rejected, before the toast.
   *
   * Exists so a caller can mark the fields the server named. Returning true
   * means the caller displayed the problem itself and the toast is suppressed —
   * which is only correct if every violation was surfaced, since a toast is the
   * last thing standing between a rejected submit and silence.
   */
  /**
   * Called when a create or update is rejected, before the toast. Returning true
   * suppresses it.
   *
   * Pass `suppressesToast` from lib/forms when the rejection is already being
   * shown on a form -- this hook deliberately knows nothing about forms, so that
   * marking fields works the same way for every mutation in the codebase.
   */
  onMutationError?: (error: unknown) => boolean | void;
}

export interface UseCrudMutationsResult<TItem, TCreate, TUpdate> {
  /** Mutation for creating items */
  createMutation: ReturnType<typeof useMutation<TItem, Error, TCreate>>;
  /** Mutation for updating items */
  updateMutation: ReturnType<typeof useMutation<TItem, Error, { id: number; data: TUpdate }>>;
  /** Mutation for deleting items */
  deleteMutation: ReturnType<typeof useMutation<void, Error, number>>;
  /** True if any mutation is currently pending */
  isMutating: boolean;
}

/**
 * Custom hook for managing CRUD mutations with consistent toast notifications
 * and query invalidation.
 */
export function useCrudMutations<TItem, TCreate, TUpdate>({
  resourceName,
  queryKey,
  extraInvalidateKeys,
  createFn,
  updateFn,
  deleteFn,
  onSuccess,
  onCreateSuccess,
  onUpdateSuccess,
  onDeleteSuccess,
  onMutationError,
}: UseCrudMutationsConfig<TItem, TCreate, TUpdate>): UseCrudMutationsResult<
  TItem,
  TCreate,
  TUpdate
> {
  const t = useTranslations();
  const feedback = useMutationFeedback();
  // Still needed directly: the optimistic delete reads and rewrites the cache,
  // which is beyond invalidating it.
  const queryClient = useQueryClient();

  const invalidateAll = () => {
    feedback.invalidate(queryKey);
    if (extraInvalidateKeys) {
      feedback.invalidate(extraInvalidateKeys);
    }
  };

  const createMutation = useMutation({
    mutationFn: (data: TCreate) => {
      if (!createFn) {
        throw new Error('createFn not provided');
      }
      return createFn(data);
    },
    onSuccess: (item) => {
      invalidateAll();
      feedback.notifySuccess(t(`${resourceName}.createSuccess`));
      onSuccess?.();
      onCreateSuccess?.(item);
    },
    onError: (error: Error) => {
      feedback.notifyError(
        error,
        t('common.failedToCreate', { resource: resourceName }),
        onMutationError
      );
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: TUpdate }) => {
      if (!updateFn) {
        throw new Error('updateFn not provided');
      }
      return updateFn(id, data);
    },
    onSuccess: (item) => {
      invalidateAll();
      feedback.notifySuccess(t(`${resourceName}.updateSuccess`));
      onSuccess?.();
      onUpdateSuccess?.(item);
    },
    onError: (error: Error) => {
      feedback.notifyError(
        error,
        t('common.failedToSave', { resource: resourceName }),
        onMutationError
      );
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => {
      if (!deleteFn) {
        throw new Error('deleteFn not provided');
      }
      return deleteFn(id);
    },
    onMutate: async (id: number) => {
      // Cancel outgoing refetches so they don't overwrite the optimistic update
      await queryClient.cancelQueries({ queryKey });

      // Snapshot previous data for rollback
      const previousQueries = queryClient.getQueriesData<unknown>({ queryKey });

      // Optimistically remove the item from all matching cached queries
      queryClient.setQueriesData<unknown>({ queryKey }, (old: unknown) => {
        if (!old || typeof old !== 'object') return old;
        // PaginatedResponse shape: { data: T[], total, ... }
        if ('data' in old && Array.isArray((old as { data: unknown[] }).data)) {
          const paginated = old as { data: Array<{ id: number }>; total: number };
          return {
            ...paginated,
            data: paginated.data.filter((item) => item.id !== id),
            total: paginated.total - 1,
          };
        }
        // Plain array shape
        if (Array.isArray(old)) {
          return (old as Array<{ id: number }>).filter((item) => item.id !== id);
        }
        return old;
      });

      return { previousQueries };
    },
    onSuccess: () => {
      feedback.notifySuccess(t(`${resourceName}.deleteSuccess`));
      onDeleteSuccess?.();
    },
    onError: (error: Error, _id, context) => {
      // Roll back to previous data on failure
      if (context?.previousQueries) {
        for (const [key, data] of context.previousQueries) {
          queryClient.setQueryData(key, data);
        }
      }
      feedback.notifyError(error, t('common.failedToDelete', { resource: resourceName }));
    },
    onSettled: () => {
      // Always refetch after delete to ensure server state consistency
      invalidateAll();
    },
  });

  const isMutating =
    createMutation.isPending || updateMutation.isPending || deleteMutation.isPending;

  return {
    createMutation,
    updateMutation,
    deleteMutation,
    isMutating,
  };
}
