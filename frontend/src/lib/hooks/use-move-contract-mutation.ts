import { useMutation, useQueryClient, type QueryKey } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toLocalDateString } from '@/lib/utils/formatting';
import { useToast } from '@/lib/hooks/use-toast';

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
  const { toast } = useToast();
  const t = useTranslations();
  const queryClient = useQueryClient();

  return useMutation<unknown, Error, MoveContractVariables, { previous?: T[] }>({
    mutationFn: (variables) => {
      // The server no longer infers this from the dates — that inference was the
      // whole problem — so the client states which it means. It can: it knows when
      // the contract starts.
      const today = toLocalDateString(new Date());
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
      toast({ title: t(config.successMessage) });
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(config.allUnpaginatedKey, context.previous);
      }
      toast({ title: t(config.errorMessage), variant: 'destructive' });
    },
    onSettled: (_data, _error, variables) => {
      queryClient.invalidateQueries({ queryKey: config.allUnpaginatedKey });
      if (variables) {
        for (const key of config.invalidateKeys(variables.entityId)) {
          queryClient.invalidateQueries({ queryKey: key });
        }
      }
    },
  });
}
