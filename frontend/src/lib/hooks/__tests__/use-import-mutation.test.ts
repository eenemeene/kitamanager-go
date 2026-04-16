import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient } from '@tanstack/react-query';
import { useImportMutation } from '../use-import-mutation';
import { createTestQueryClient, createHookWrapper } from '@/test-utils';

const mockToast = jest.fn();
jest.mock('../use-toast', () => ({
  useToast: () => ({ toast: mockToast }),
}));

const mockShowErrorToast = jest.fn();
jest.mock('@/lib/utils/show-error-toast', () => ({
  showErrorToast: (...args: unknown[]) => mockShowErrorToast(...args),
}));

describe('useImportMutation', () => {
  let queryClient: QueryClient;
  let wrapper: ReturnType<typeof createHookWrapper>;

  beforeEach(() => {
    queryClient = createTestQueryClient();
    wrapper = createHookWrapper(queryClient);
    mockToast.mockClear();
    mockShowErrorToast.mockClear();
    jest.spyOn(queryClient, 'invalidateQueries');
  });

  afterEach(() => {
    queryClient.clear();
  });

  it('invalidates all provided query keys on success', async () => {
    const importFn = jest.fn().mockResolvedValue({});
    const childrenKey = ['children', 1];
    const statisticsKey = ['statistics', 1];

    const { result } = renderHook(
      () =>
        useImportMutation({
          importFn,
          invalidateQueryKeys: [childrenKey, statisticsKey],
          resourceNameKey: 'children.title',
          errorMessageKey: 'children.importError',
        }),
      { wrapper }
    );

    const file = new File(['test'], 'import.yaml', { type: 'text/yaml' });
    act(() => {
      result.current.mutation.mutate(file);
    });

    await waitFor(() => {
      expect(result.current.mutation.isSuccess).toBe(true);
    });

    expect(queryClient.invalidateQueries).toHaveBeenCalledWith({ queryKey: childrenKey });
    expect(queryClient.invalidateQueries).toHaveBeenCalledWith({ queryKey: statisticsKey });
    expect(mockToast).toHaveBeenCalledWith(expect.objectContaining({ title: 'common.success' }));
  });

  it('shows error toast on failure', async () => {
    const importFn = jest.fn().mockRejectedValue(new Error('Parse error'));

    const { result } = renderHook(
      () =>
        useImportMutation({
          importFn,
          invalidateQueryKeys: [['children', 1]],
          resourceNameKey: 'children.title',
          errorMessageKey: 'children.importError',
        }),
      { wrapper }
    );

    const file = new File(['bad'], 'import.yaml', { type: 'text/yaml' });
    act(() => {
      result.current.mutation.mutate(file);
    });

    await waitFor(() => {
      expect(result.current.mutation.isError).toBe(true);
    });

    expect(mockShowErrorToast).toHaveBeenCalledWith(
      'common.error',
      expect.any(Error),
      'children.importError'
    );
  });
});
