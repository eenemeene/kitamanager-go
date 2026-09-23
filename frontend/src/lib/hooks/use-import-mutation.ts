'use client';

import { useRef } from 'react';
import { useMutation, type QueryKey } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useMutationFeedback } from './use-mutation-feedback';

interface UseImportMutationConfig {
  /**
   * API function to call with the file. It answers with the rows the import
   * settled, which is what the success message counts.
   */
  importFn: (file: File) => Promise<unknown[]>;
  /** Query keys to invalidate on success */
  invalidateQueryKeys: QueryKey[];
  /**
   * i18n key for the success message, given a `count` (e.g.
   * 'children.importSuccess'). It has to describe an import rather than a
   * creation: both importers upsert on name and birthdate, so a re-import of
   * the same file creates nothing at all.
   */
  successMessageKey: string;
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
  successMessageKey,
  errorMessageKey,
}: UseImportMutationConfig) {
  const t = useTranslations();
  const feedback = useMutationFeedback();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const mutation = useMutation({
    mutationFn: importFn,
    onSuccess: (data) => {
      feedback.invalidate(invalidateQueryKeys);
      // What happened, not what was asked for. The import upserts on name and
      // birthdate, so announcing a creation was wrong every time a file was
      // re-imported to update the rows already there -- naming an operation
      // that did not occur, which is the same thing the contract-amend toast
      // was fixed for.
      feedback.notifySuccess(t('common.success'), t(successMessageKey, { count: data.length }));
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
