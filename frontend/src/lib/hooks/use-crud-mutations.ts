'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useToast } from './use-toast';
import { getErrorMessage } from '@/lib/api/client';

export interface UseCrudMutationsConfig<TItem, TCreate, TUpdate> {
  /** Resource name for i18n keys (e.g., 'groups', 'organizations') */
  resourceName: string;
  /** Query key to invalidate on success */
  queryKey: (string | number | undefined)[];
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
  createFn,
  updateFn,
  deleteFn,
  onSuccess,
  onCreateSuccess,
  onUpdateSuccess,
  onDeleteSuccess,
}: UseCrudMutationsConfig<TItem, TCreate, TUpdate>): UseCrudMutationsResult<
  TItem,
  TCreate,
  TUpdate
> {
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const createMutation = useMutation({
    mutationFn: (data: TCreate) => {
      if (!createFn) {
        throw new Error('createFn not provided');
      }
      return createFn(data);
    },
    onSuccess: (item) => {
      queryClient.invalidateQueries({ queryKey });
      toast({ title: t(`${resourceName}.createSuccess`) });
      onSuccess?.();
      onCreateSuccess?.(item);
    },
    onError: (error: Error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToCreate', { resource: resourceName })),
        variant: 'destructive',
      });
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
      queryClient.invalidateQueries({ queryKey });
      toast({ title: t(`${resourceName}.updateSuccess`) });
      onSuccess?.();
      onUpdateSuccess?.(item);
    },
    onError: (error: Error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToSave', { resource: resourceName })),
        variant: 'destructive',
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => {
      if (!deleteFn) {
        throw new Error('deleteFn not provided');
      }
      return deleteFn(id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
      toast({ title: t(`${resourceName}.deleteSuccess`) });
      onDeleteSuccess?.();
    },
    onError: (error: Error) => {
      toast({
        title: t('common.error'),
        description: getErrorMessage(error, t('common.failedToDelete', { resource: resourceName })),
        variant: 'destructive',
      });
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
