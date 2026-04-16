'use client';

import { useRef } from 'react';
import { useMutation, useQueryClient, type QueryKey } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useToast } from './use-toast';
import { showErrorToast } from '@/lib/utils/show-error-toast';

interface UseImportMutationConfig {
  /** API function to call with the file */
  importFn: (file: File) => Promise<unknown>;
  /** Query keys to invalidate on success */
  invalidateQueryKeys: QueryKey[];
  /** i18n key for the resource name (e.g., 'children.title') */
  resourceNameKey: string;
  /** i18n key for the import error fallback (e.g., 'children.importError') */
  errorMessageKey: string;
}

/**
 * Shared hook for YAML file import mutations.
 * Provides mutation state, a file input ref, and consistent toast notifications.
 */
export function useImportMutation({
  importFn,
  invalidateQueryKeys,
  resourceNameKey,
  errorMessageKey,
}: UseImportMutationConfig) {
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const mutation = useMutation({
    mutationFn: importFn,
    onSuccess: () => {
      for (const key of invalidateQueryKeys) {
        queryClient.invalidateQueries({ queryKey: key });
      }
      toast({
        title: t('common.success'),
        description: t('common.createSuccess', { resource: t(resourceNameKey) }),
      });
    },
    onError: (error) => {
      showErrorToast(t('common.error'), error, t(errorMessageKey));
    },
  });

  const triggerFileInput = () => fileInputRef.current?.click();

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      mutation.mutate(file);
      e.target.value = '';
    }
  };

  return {
    mutation,
    fileInputRef,
    triggerFileInput,
    handleFileChange,
    isPending: mutation.isPending,
  };
}
