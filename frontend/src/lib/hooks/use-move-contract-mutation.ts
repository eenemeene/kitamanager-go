import { useMutation, useQueryClient, type QueryKey } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useMutationFeedback } from './use-mutation-feedback';
import { todayBerlinString } from '@/lib/utils/contracts';

interface HasContracts {
  id: number;
  contracts?: { id: number; section_id: number; version: number }[];
}

interface MoveContractConfig<T extends HasContracts> {
  orgId: number;
  /**
   * Amend the contract: close it yesterday and open a successor in the target
   * section from today. This is the normal path, because which section someone
   * was in is history worth keeping — occupancy and staffing reports are per
   * section, so rewriting it in place would restate months that already passed.
   */
  amendFn: (
    entityId: number,
    contractId: number,
    sectionId: number,
    version: number,
    effectiveFrom: string
  ) => Promise<unknown>;
  /**
   * Correct the contract in place, used only when it has not started yet: there
   * is no past to preserve, and an amendment would be refused anyway because its
   * effective date has to fall after the contract's own start.
   *
   * `version` is the optimistic-concurrency token both paths must send, so moving
   * someone between sections cannot overwrite another user's concurrent edit.
   */
  correctFn: (
    entityId: number,
    contractId: number,
    sectionId: number,
    version: number
  ) => Promise<unknown>;
  /** Query key for the unpaginated list used for optimistic updates. */
  allUnpaginatedKey: QueryKey;
  /** Additional query keys to invalidate on settle (all, detail, contracts, etc.). */
  invalidateKeys: (entityId: number) => QueryKey[];
  /** i18n key for the success toast. */
  successMessage: string;
  /** i18n key for the error toast. */
  errorMessage: string;
}

export interface MoveContractVariables {
  entityId: number;
  contractId: number;
  sectionId: number;
  /** The contract version as read, sent as the If-Match precondition. */
  version: number;
  /** The contract's start date, which decides amend versus correct. */
  from: string;
}

export function useMoveContractMutation<T extends HasContracts>(config: MoveContractConfig<T>) {
  const feedback = useMutationFeedback();
  const t = useTranslations();
  const queryClient = useQueryClient();

  return useMutation<unknown, Error, MoveContractVariables, { previous?: T[] }>({
    mutationFn: (variables) => {
      // The server no longer infers this from the dates — that inference was the
      // whole problem — so the client states which it means. It can: it knows when
      // the contract starts.
      const today = todayBerlinString();
      const startsLater = variables.from.slice(0, 10) > today;
      if (startsLater) {
        return config.correctFn(
          variables.entityId,
          variables.contractId,
          variables.sectionId,
          variables.version
        );
      }
      return config.amendFn(
        variables.entityId,
        variables.contractId,
        variables.sectionId,
        variables.version,
        today
      );
    },
    onMutate: async ({ entityId, contractId, sectionId }) => {
      await queryClient.cancelQueries({ queryKey: config.allUnpaginatedKey });
      const previous = queryClient.getQueryData<T[]>(config.allUnpaginatedKey);
      queryClient.setQueryData<T[]>(config.allUnpaginatedKey, (old) =>
        old?.map((item) =>
          item.id === entityId
            ? {
                ...item,
                contracts: item.contracts?.map((ct) =>
                  ct.id === contractId ? { ...ct, section_id: sectionId } : ct
                ),
              }
            : item
        )
      );
      return { previous };
    },
    onSuccess: () => {
      feedback.notifySuccess(t(config.successMessage));
    },
    onError: (error, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(config.allUnpaginatedKey, context.previous);
      }
      // Say why, not just that it failed. Dragging a card here corrects or
      // amends a contract, and the refusals carry information the fixed string
      // throws away -- most usefully the stale-version one, which a board left
      // open for a while will hit: "this contract was changed by someone else
      // (you have version 3, current is 4) -- reload and reapply your change"
      // tells the user what to do, where "Failed to move child" does not.
      //
      // The configured message stays as the fallback, for the failures that
      // carry no message of their own.
      feedback.notifyError(error, t(config.errorMessage));
    },
    onSettled: (_data, _error, variables) => {
      // On settle rather than on success: the optimistic update has to be rolled
      // back before the refetch, or the two race.
      feedback.invalidate(config.allUnpaginatedKey);
      if (variables) {
        for (const key of config.invalidateKeys(variables.entityId)) {
          queryClient.invalidateQueries({ queryKey: key });
        }
      }
    },
  });
}
