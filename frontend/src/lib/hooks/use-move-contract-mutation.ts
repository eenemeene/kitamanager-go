import { useMutation, useQueryClient, type QueryKey } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useToast } from '@/lib/hooks/use-toast';

interface HasContracts {
  id: number;
  contracts?: { id: number; section_id: number }[];
}

interface MoveContractConfig<T extends HasContracts> {
  orgId: number;
  /** API call to update the contract's section. */
  updateFn: (entityId: number, contractId: number, sectionId: number) => Promise<unknown>;
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
}

export function useMoveContractMutation<T extends HasContracts>(config: MoveContractConfig<T>) {
  const { toast } = useToast();
  const t = useTranslations();
  const queryClient = useQueryClient();

  return useMutation<unknown, Error, MoveContractVariables, { previous?: T[] }>({
    mutationFn: (variables) =>
      config.updateFn(variables.entityId, variables.contractId, variables.sectionId),
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
