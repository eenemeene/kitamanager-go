'use client';

import { useRef } from 'react';
import { useMutation, type QueryKey } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useMutationFeedback } from './use-mutation-feedback';

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
  const feedback = useMutationFeedback();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const mutation = useMutation({
    mutationFn: importFn,
    onSuccess: () => {
      feedback.invalidate(invalidateQueryKeys);
      feedback.notifySuccess(
        t('common.success'),
        t('common.createSuccess', { resource: t(resourceNameKey) })
      );
    },
    onError: (error) => {
      feedback.notifyError(error, t(errorMessageKey));
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
