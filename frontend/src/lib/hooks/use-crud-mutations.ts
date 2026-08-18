'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import type {
  FieldValues,
  UseFormClearErrors,
  UseFormGetValues,
  UseFormSetError,
} from 'react-hook-form';
import { useToast } from './use-toast';
import { showErrorToast } from '@/lib/utils/show-error-toast';
import { applyProblemToForm } from '@/lib/forms/apply-problem-to-form';
import type { InvalidParam } from '@/lib/api/problem';

/**
 * The slice of react-hook-form this hook needs to mark a rejected field. Taking
 * the three callbacks rather than the whole form object keeps the dependency
 * narrow and matches what applyProblemToForm already expects.
 */
export interface CrudMutationForm<T extends FieldValues = FieldValues> {
  setError: UseFormSetError<T>;
  clearErrors: UseFormClearErrors<T>;
  getValues: UseFormGetValues<T>;
}

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
  /**
   * The form whose fields a rejected create or update should mark.
   *
   * Supplying it is what turns "something went wrong" into an outline around the
   * input to change. Every page that owns a form should pass it; the hook then
   * applies the server's field violations, keeps the ones it could not place in
   * `unmappedViolations` for the summary to show, and suppresses its own toast
   * whenever it surfaced anything -- and only then, since a conflict or a
   * network failure has no field violations and still needs the toast.
   *
   * Optional because the hook is also used for delete-only flows, which have no
   * form to mark.
   */
  form?: CrudMutationForm<never>;
  /**
   * Maps an API field name to this form's name, where they differ -- money
   * entered in euros but sent in cents, a date pair prefixed because the form
   * carries two. Without an entry the violation is reported in the summary
   * rather than marked, which is noisy but never silent.
   */
  fieldAliases?: Record<string, string>;
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
  /**
   * Server violations that named a field this form does not have. The summary
   * must still show them -- a violation nobody displays is a rejected submit the
   * user cannot explain.
   */
  unmappedViolations: InvalidParam[];
  /**
   * Clears the previous attempt's collection-level problems. Call it as the form
   * is submitted, so a corrected form does not keep showing stale ones.
   */
  clearUnmappedViolations: () => void;
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
  form,
  fieldAliases,
}: UseCrudMutationsConfig<TItem, TCreate, TUpdate>): UseCrudMutationsResult<
  TItem,
  TCreate,
  TUpdate
> {
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const [unmappedViolations, setUnmappedViolations] = useState<InvalidParam[]>([]);

  /**
   * Marks what the server named, then answers whether the toast is still needed.
   *
   * Runs before any caller-supplied onMutationError so a page can still add its
   * own handling; a caller returning true suppresses the toast regardless.
   */
  const handleFieldViolations = (error: unknown): boolean => {
    if (!form) return false;
    const { applied, unmapped } = applyProblemToForm(error, form, fieldAliases);
    setUnmappedViolations(unmapped);
    return applied + unmapped.length > 0;
  };

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
      const marked = handleFieldViolations(error);
      if (onMutationError?.(error) === true || marked) {
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
      const marked = handleFieldViolations(error);
      if (onMutationError?.(error) === true || marked) {
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
    unmappedViolations,
    clearUnmappedViolations: () => setUnmappedViolations([]),
  };
}
