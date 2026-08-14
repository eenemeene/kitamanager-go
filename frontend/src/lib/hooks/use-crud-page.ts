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
import { useResourceListFilters } from './use-resource-list-filters';
import { validationTiming } from '@/lib/forms/validation-timing';
import { applyProblemToForm } from '@/lib/forms/apply-problem-to-form';

interface UseCrudPageConfig<
  TItem extends { id: number },
  TFormData extends FieldValues,
  TCreate,
  TUpdate,
> {
  resourceName: string;
  /**
   * Maps an API field name to this form's name, where they differ — money
   * entered in euros but sent in cents, a date pair prefixed because the form
   * carries two. Without an entry the violation is reported rather than marked,
   * which is noisy but never silent.
   */
  fieldAliases?: Record<string, string>;
  schema: ZodType<TFormData>;
  defaultValues: TFormData;
  itemToFormData: (item: TItem) => TFormData;
  listFn: (orgId: number, params: PaginationParams) => Promise<PaginatedResponse<TItem>>;
  createFn: (orgId: number, data: TCreate) => Promise<TItem>;
  updateFn: (orgId: number, id: number, data: TUpdate) => Promise<TItem>;
  deleteFn: (orgId: number, id: number) => Promise<void>;
  /** Enable search input support (debounced, auto-resets page) */
  searchable?: boolean;
  /** Optional query key functions for proper cache alignment with queryKeys factory */
  queryKeys?: {
    list: (orgId: number, page: number, search?: string) => readonly unknown[];
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
  error: Error | null;
  refetch: () => void;
  page: number;
  setPage: (page: number) => void;
  /** Search input value (raw, for binding to SearchInput) — only present when searchable */
  searchInput: string;
  /** Set search input (auto-resets page to 1) — only present when searchable */
  setSearchInput: (value: string) => void;
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

  const filters = useResourceListFilters();
  const [simplePage, setSimplePage] = useState(1);

  // When searchable, use the filters hook (debounced search + auto page reset).
  // Otherwise, use simple page state for backward compatibility.
  const page = config.searchable ? filters.page : simplePage;
  const setPage = config.searchable ? filters.setPage : setSimplePage;
  const searchInput = filters.searchInput;
  const setSearchInput = filters.setSearchInput;
  const search = config.searchable ? filters.search : undefined;

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    setError,
    clearErrors,
    getValues,
    watch,
    formState: { errors },
  } = useForm<TFormData>({
    ...validationTiming,
    resolver: zodResolver(config.schema as any),
    defaultValues: config.defaultValues as DefaultValues<TFormData>,
  });

  const listQueryKey = config.queryKeys
    ? config.queryKeys.list(orgId, page, search)
    : [config.resourceName, orgId, page, search];
  const invalidateQueryKey: readonly (string | number | undefined)[] = config.queryKeys
    ? (config.queryKeys.invalidate(orgId) as readonly (string | number | undefined)[])
    : [config.resourceName, orgId];

  const {
    data: paginatedData,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: listQueryKey,
    queryFn: () => config.listFn(orgId, { page, ...(search ? { search } : {}) }),
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
    onMutationError: (error) => {
      // Mark the fields the server named, so a rejected submit points at the
      // inputs to change instead of describing them in a sentence.
      const { applied, unmapped } = applyProblemToForm(
        error,
        { setError, clearErrors, getValues },
        config.fieldAliases
      );
      // The toast is suppressed only when every violation landed on a field.
      // Anything unmapped — a collection-level failure, a field this form does
      // not collect — still needs saying, and the toast is the only thing that
      // says it until the error summary lands.
      return applied > 0 && unmapped.length === 0;
    },
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
    error,
    refetch,
    page,
    setPage,
    searchInput,
    setSearchInput,
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
