import { useMutation, useQueryClient, type QueryKey } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useToast } from '@/lib/hooks/use-toast';
import { showErrorToast } from '@/lib/utils/show-error-toast';
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
  const { toast } = useToast();

  return useMutation({
    mutationFn: async (variables: ContractMutationVariables<TCreateData>) => {
      const { entityId, data, entity, endCurrentContract } = variables;

      if (entity && endCurrentContract) {
        const active = getActiveContract(entity.contracts);
        if (active) {
          return config.amendFn(entityId, active.id, active.version, config.toAmendData(data));
        }
      }
      return config.createFn(entityId, data);
    },
    onSuccess: (_data, variables) => {
      for (const key of config.invalidateQueryKeys) {
        queryClient.invalidateQueries({ queryKey: key });
      }
      if (config.extraInvalidateKeys) {
        for (const key of config.extraInvalidateKeys(variables.entityId)) {
          queryClient.invalidateQueries({ queryKey: key });
        }
      }

      toast({
        title: variables.endCurrentContract
          ? t('contracts.previousContractEnded')
          : t('contracts.createSuccess'),
      });

      config.onSuccess?.();
    },
    onError: (error: unknown) => {
      showErrorToast(
        t('common.error'),
        error,
        t('common.failedToCreate', { resource: 'contract' })
      );
    },
  });
}
