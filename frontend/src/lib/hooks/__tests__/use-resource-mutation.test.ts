import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient } from '@tanstack/react-query';
import { useResourceMutation } from '../use-resource-mutation';
import { createTestQueryClient, createHookWrapper } from '@/test-utils';

const mockToast = jest.fn();
jest.mock('../use-toast', () => ({
  useToast: () => ({ toast: mockToast }),
}));

const mockShowErrorToast = jest.fn();
jest.mock('@/lib/utils/show-error-toast', () => ({
  showErrorToast: (...args: unknown[]) => mockShowErrorToast(...args),
}));

describe('useResourceMutation', () => {
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

  it('invalidates a single query key on success', async () => {
    const mockFn = jest.fn().mockResolvedValue({ id: 1 });

    const { result } = renderHook(
      () =>
        useResourceMutation({
          mutationFn: mockFn,
          invalidateQueryKey: ['items', 1],
          successMessage: 'Created!',
          errorMessage: 'Failed!',
        }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({ name: 'test' });
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(queryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: ['items', 1],
    });
    expect(mockToast).toHaveBeenCalledWith({ title: 'Created!' });
  });

  it('invalidates multiple query keys on success', async () => {
    const mockFn = jest.fn().mockResolvedValue({ id: 1 });
    const key1 = ['budgetItem', 1, 2];
    const key2 = ['financials', 1, undefined, undefined];

    const { result } = renderHook(
      () =>
        useResourceMutation({
          mutationFn: mockFn,
          invalidateQueryKey: [key1, key2],
          successMessage: 'Updated!',
          errorMessage: 'Failed!',
        }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({ amount: 100 });
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(queryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: ['budgetItem', 1, 2],
    });
    expect(queryClient.invalidateQueries).toHaveBeenCalledWith({
      queryKey: ['financials', 1, undefined, undefined],
    });
  });

  it('shows error toast on failure', async () => {
    const mockFn = jest.fn().mockRejectedValue(new Error('Server error'));

    const { result } = renderHook(
      () =>
        useResourceMutation({
          mutationFn: mockFn,
          invalidateQueryKey: ['items'],
          successMessage: 'OK',
          errorMessage: 'Failed to save',
        }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({ name: 'test' });
    });

    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });

    expect(mockShowErrorToast).toHaveBeenCalledWith(
      'common.error',
      expect.any(Error),
      'Failed to save'
    );
  });

  it('calls onSuccess callback after mutation', async () => {
    const mockFn = jest.fn().mockResolvedValue({ id: 1 });
    const onSuccess = jest.fn();

    const { result } = renderHook(
      () =>
        useResourceMutation({
          mutationFn: mockFn,
          invalidateQueryKey: ['items'],
          successMessage: 'OK',
          errorMessage: 'Failed',
          onSuccess,
        }),
      { wrapper }
    );

    act(() => {
      result.current.mutate({ name: 'test' });
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(onSuccess).toHaveBeenCalled();
  });
});
