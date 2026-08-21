import { useMutation, useQueryClient, type QueryKey } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useMutationFeedback } from './use-mutation-feedback';
import { getActiveContract } from '@/lib/utils/contracts';

interface ContractMutationConfig<TCreateData, TAmendData, TContract, TAmendResult> {
  /** Create a new contract for an entity. */
  createFn: (entityId: number, data: TCreateData) => Promise<TContract>;
  /**
   * Amend an existing contract: close it the day before the effective date and
   * create a successor carrying the changes. `version` is the contract's
   * optimistic-concurrency token, sent as an If-Match precondition.
   */
  amendFn: (
    entityId: number,
    contractId: number,
    version: number,
    data: TAmendData
  ) => Promise<TAmendResult>;
  /**
   * Convert the form's create payload into an amendment. The form's `from` date
   * becomes `effective_from`, which is the fix for a long-standing bug: this used
   * to *strip* `from`, and the server then started the successor today — so the
   * dialog let the user pick a start date and silently used a different one.
   */
  toAmendData: (createData: TCreateData) => TAmendData;
  /** Query keys to invalidate on success. */
  invalidateQueryKeys: QueryKey[];
  /** Additional query keys to invalidate, given the entity ID (e.g., per-entity contract list). */
  extraInvalidateKeys?: (entityId: number) => QueryKey[];
  /** Called after a successful mutation (use for closing dialogs, resetting state, etc.). */
  onSuccess?: () => void;
  /**
   * Called when the mutation is rejected, before the toast. Returning true
   * suppresses it.
   *
   * Pass `suppressesToast` from lib/forms when the rejection is already being
   * shown on a form. Like the other mutation hooks, this one knows nothing about
   * forms -- marking fields is done by watching the mutation's error.
   */
  onMutationError?: (error: unknown) => boolean | void;
}

interface ContractMutationVariables<TCreateData> {
  entityId: number;
  data: TCreateData;
  /** The entity with its contracts array, used to find the active contract. */
  entity: {
    contracts?: Array<{ id: number; from: string; to?: string | null; version: number }>;
  } | null;
  /** Whether to end the current contract (update) instead of creating a new one. */
  endCurrentContract: boolean;
}

/**
 * Shared hook for contract create/amend mutations, backing the "End current
 * contract and create a new one" choice in the new-contract dialogs.
 *
 * That checkbox is the user answering the question no server can infer: did the
 * facts change as of a date (amend, keeping the old ones on record for the months
 * they applied to), or is this simply a new contract (create)?
 */
export function useContractMutation<TCreateData, TAmendData, TContract, TAmendResult>(
  config: ContractMutationConfig<TCreateData, TAmendData, TContract, TAmendResult>
) {
  const queryClient = useQueryClient();
  const t = useTranslations();
  const feedback = useMutationFeedback();

  return useMutation({
    mutationFn: async (variables: ContractMutationVariables<TCreateData>) => {
      const { entityId, data, entity, endCurrentContract } = variables;

      if (entity && endCurrentContract) {
        const active = getActiveContract(entity.contracts);
        if (active) {
          const result = await config.amendFn(
            entityId,
            active.id,
            active.version,
            config.toAmendData(data)
          );
          return { result, amended: true };
        }
      }
      return { result: await config.createFn(entityId, data), amended: false };
    },
    onSuccess: (data, variables) => {
      feedback.invalidate(config.invalidateQueryKeys);
      if (config.extraInvalidateKeys) {
        feedback.invalidate(config.extraInvalidateKeys(variables.entityId));
      }
      // What actually happened, not what was asked for. Ticking the box with no
      // active contract to end falls through to a plain create, and the toast
      // still announced that the previous contract had been ended -- naming an
      // operation that did not occur, on a screen about contract history.
      feedback.notifySuccess(
        data.amended ? t('contracts.previousContractEnded') : t('contracts.createSuccess')
      );
      config.onSuccess?.();
    },
    onError: (error: unknown) => {
      feedback.notifyError(
        error,
        t('common.failedToCreate', { resource: 'contract' }),
        config.onMutationError
      );
    },
  });
}
