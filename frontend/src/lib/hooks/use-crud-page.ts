'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import type {
  FieldValues,
  UseFormRegister,
  UseFormHandleSubmit,
  FieldErrors,
  UseFormSetValue,
  UseFormSetError,
  UseFormWatch,
  DefaultValues,
} from 'react-hook-form';
import type { z, ZodType } from 'zod';
import type { PaginatedResponse, PaginationParams } from '@/lib/api/types';
import { useCrudDialogs, type UseCrudDialogsResult } from './use-crud-dialogs';
import { useCrudMutations, type UseCrudMutationsResult } from './use-crud-mutations';

interface UseCrudPageConfig<
  TItem extends { id: number },
  TFormData extends FieldValues,
  TCreate,
  TUpdate,
> {
  resourceName: string;
  schema: ZodType<TFormData>;
  defaultValues: TFormData;
  itemToFormData: (item: TItem) => TFormData;
  listFn: (orgId: number, params: PaginationParams) => Promise<PaginatedResponse<TItem>>;
  createFn: (orgId: number, data: TCreate) => Promise<TItem>;
  updateFn: (orgId: number, id: number, data: TUpdate) => Promise<TItem>;
  deleteFn: (orgId: number, id: number) => Promise<void>;
  /** Optional query key functions for proper cache alignment with queryKeys factory */
  queryKeys?: {
    list: (orgId: number, page: number) => readonly unknown[];
    invalidate: (orgId: number) => readonly unknown[];
    /** Additional query keys to invalidate on success (e.g., related statistics) */
    extraInvalidate?: (orgId: number) => readonly (string | number | undefined)[][];
  };
}

interface UseCrudPageResult<
  TItem extends { id: number },
  TFormData extends FieldValues,
  TCreate,
  TUpdate,
> {
  orgId: number;
  items: TItem[] | undefined;
  paginatedData: PaginatedResponse<TItem> | undefined;
  isLoading: boolean;
  page: number;
  setPage: (page: number) => void;
  register: UseFormRegister<TFormData>;

  handleSubmit: UseFormHandleSubmit<TFormData, any>;
  errors: FieldErrors<TFormData>;
  setValue: UseFormSetValue<TFormData>;
  setError: UseFormSetError<TFormData>;
  watch: UseFormWatch<TFormData>;
  dialogs: UseCrudDialogsResult<TItem>;
  mutations: UseCrudMutationsResult<TItem, TCreate, TUpdate>;
  onSubmit: (data: TFormData) => void;
}

export function useCrudPage<
  TItem extends { id: number },
  TFormData extends FieldValues,
  TCreate,
  TUpdate,
>(
  config: UseCrudPageConfig<TItem, TFormData, TCreate, TUpdate>
): UseCrudPageResult<TItem, TFormData, TCreate, TUpdate> {
  const params = useParams();
  const orgId = Number(params.orgId);
  const [page, setPage] = useState(1);

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    setError,
    watch,
    formState: { errors },
  } = useForm<TFormData>({
    resolver: zodResolver(config.schema as any),
    defaultValues: config.defaultValues as DefaultValues<TFormData>,
  });

  const listQueryKey = config.queryKeys
    ? config.queryKeys.list(orgId, page)
    : [config.resourceName, orgId, page];
  const invalidateQueryKey: readonly (string | number | undefined)[] = config.queryKeys
    ? (config.queryKeys.invalidate(orgId) as readonly (string | number | undefined)[])
    : [config.resourceName, orgId];

  const { data: paginatedData, isLoading } = useQuery({
    queryKey: listQueryKey,
    queryFn: () => config.listFn(orgId, { page }),
    enabled: !!orgId,
  });

  const items = paginatedData?.data;

  const dialogs = useCrudDialogs<TItem, TFormData>({
    reset,
    itemToFormData: config.itemToFormData,
    defaultValues: config.defaultValues,
  });

  const extraInvalidateKeys = config.queryKeys?.extraInvalidate?.(orgId);

  const mutations = useCrudMutations<TItem, TCreate, TUpdate>({
    resourceName: config.resourceName,
    queryKey: invalidateQueryKey,
    extraInvalidateKeys,
    createFn: (data) => config.createFn(orgId, data),
    updateFn: (id, data) => config.updateFn(orgId, id, data),
    deleteFn: (id) => config.deleteFn(orgId, id),
    onSuccess: dialogs.closeDialog,
    onDeleteSuccess: dialogs.closeDeleteDialog,
  });

  const onSubmit = (data: TFormData) => {
    if (dialogs.editingItem) {
      mutations.updateMutation.mutate({
        id: dialogs.editingItem.id,
        data: data as unknown as TUpdate,
      });
    } else {
      mutations.createMutation.mutate(data as unknown as TCreate);
    }
  };

  return {
    orgId,
    items,
    paginatedData,
    isLoading,
    page,
    setPage,
    register,
    handleSubmit,
    errors,
    setValue,
    setError,
    watch,
    dialogs,
    mutations,
    onSubmit,
  };
}
