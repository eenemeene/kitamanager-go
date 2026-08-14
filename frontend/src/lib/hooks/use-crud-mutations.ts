'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useToast } from './use-toast';
import { showErrorToast } from '@/lib/utils/show-error-toast';

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
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const invalidateAll = () => {
    queryClient.invalidateQueries({ queryKey });
    if (extraInvalidateKeys) {
      for (const key of extraInvalidateKeys) {
        queryClient.invalidateQueries({ queryKey: key });
      }
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
      toast({ title: t(`${resourceName}.createSuccess`) });
      onSuccess?.();
      onCreateSuccess?.(item);
    },
    onError: (error: Error) => {
      if (onMutationError?.(error) === true) {
        return;
      }
      showErrorToast(
        t('common.error'),
        error,
        t('common.failedToCreate', { resource: resourceName })
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
      toast({ title: t(`${resourceName}.updateSuccess`) });
      onSuccess?.();
      onUpdateSuccess?.(item);
    },
    onError: (error: Error) => {
      if (onMutationError?.(error) === true) {
        return;
      }
      showErrorToast(
        t('common.error'),
        error,
        t('common.failedToSave', { resource: resourceName })
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
      toast({ title: t(`${resourceName}.deleteSuccess`) });
      onDeleteSuccess?.();
    },
    onError: (error: Error, _id, context) => {
      // Roll back to previous data on failure
      if (context?.previousQueries) {
        for (const [key, data] of context.previousQueries) {
          queryClient.setQueryData(key, data);
        }
      }
      showErrorToast(
        t('common.error'),
        error,
        t('common.failedToDelete', { resource: resourceName })
      );
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
