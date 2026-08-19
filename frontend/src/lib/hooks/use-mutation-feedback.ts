'use client';

import { useQueryClient, type QueryKey } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useToast } from './use-toast';
import { showErrorToast } from '@/lib/utils/show-error-toast';

/**
 * What every mutation hook in this codebase does either side of the request:
 * drop the caches the write invalidated, say it worked, or say why it did not.
 *
 * Five hooks wrap useMutation here -- CRUD, resource, contract, boundary move,
 * import -- and each had its own copy of these three lines. They are not the same
 * hook and should not be merged: the boundary move invalidates on settle because
 * it updates optimistically and has to roll back first, the import owns a file
 * input, CRUD carries an optimistic delete. What they share is the feedback, so
 * that is what lives here.
 *
 * The reason to centralise it is the suppression rule below rather than the line
 * count. Three hooks had grown their own copy of "a rejection already shown on a
 * form should not also raise a toast", which is exactly the kind of rule that
 * gets fixed in two places out of three.
 */
export function useMutationFeedback() {
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  return {
    /** Drops every listed cache. Accepts one key or many, since callers have both. */
    invalidate: (keys: QueryKey | QueryKey[]) => {
      const list = Array.isArray(keys[0]) ? (keys as QueryKey[]) : [keys as QueryKey];
      for (const key of list) {
        queryClient.invalidateQueries({ queryKey: key });
      }
    },

    /** The write landed. */
    notifySuccess: (title: string, description?: string) => {
      toast({ title, description });
    },

    /**
     * The write was rejected.
     *
     * `handled` is the seam a caller uses to say the rejection is already on
     * screen -- marked on a form, most often -- in which case a toast repeating
     * it is noise. Returning anything other than true still raises the toast,
     * because silence after a failed submit is the worse mistake.
     */
    notifyError: (
      error: unknown,
      fallback: string,
      handled?: (error: unknown) => boolean | void
    ) => {
      if (handled?.(error) === true) {
        return;
      }
      showErrorToast(t('common.error'), error, fallback);
    },
  };
}
