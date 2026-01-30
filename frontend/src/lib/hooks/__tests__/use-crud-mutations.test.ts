import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useCrudMutations } from '../use-crud-mutations';
import React from 'react';

// Mock the toast hook
const mockToast = jest.fn();
jest.mock('../use-toast', () => ({
  useToast: () => ({ toast: mockToast }),
}));

// Mock getErrorMessage
jest.mock('@/lib/api/client', () => ({
  getErrorMessage: (error: unknown, fallback: string) => {
    if (error && typeof error === 'object' && 'message' in error) {
      return (error as { message: string }).message;
    }
    return fallback;
  },
}));

interface TestItem {
  id: number;
  name: string;
}

interface TestCreateData {
  name: string;
}

interface TestUpdateData {
  name?: string;
}

describe('useCrudMutations', () => {
  let queryClient: QueryClient;

  const wrapper = ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    mockToast.mockClear();
    jest.spyOn(queryClient, 'invalidateQueries');
  });

  afterEach(() => {
    queryClient.clear();
  });

  describe('createMutation', () => {
    it('calls createFn and shows success toast on success', async () => {
      const mockCreateFn = jest.fn().mockResolvedValue({ id: 1, name: 'New Item' });
      const mockOnSuccess = jest.fn();
      const mockOnCreateSuccess = jest.fn();

      const { result } = renderHook(
        () =>
          useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
            resourceName: 'items',
            queryKey: ['items'],
            createFn: mockCreateFn,
            onSuccess: mockOnSuccess,
            onCreateSuccess: mockOnCreateSuccess,
          }),
        { wrapper }
      );

      act(() => {
        result.current.createMutation.mutate({ name: 'New Item' });
      });

      await waitFor(() => {
        expect(result.current.createMutation.isSuccess).toBe(true);
      });

      expect(mockCreateFn).toHaveBeenCalledWith({ name: 'New Item' });
      expect(mockToast).toHaveBeenCalledWith({ title: 'items.createSuccess' });
      expect(queryClient.invalidateQueries).toHaveBeenCalledWith({ queryKey: ['items'] });
      expect(mockOnSuccess).toHaveBeenCalled();
      expect(mockOnCreateSuccess).toHaveBeenCalledWith({ id: 1, name: 'New Item' });
    });

    it('shows error toast on failure', async () => {
      const mockCreateFn = jest.fn().mockRejectedValue(new Error('Create failed'));

      const { result } = renderHook(
        () =>
          useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
            resourceName: 'items',
            queryKey: ['items'],
            createFn: mockCreateFn,
          }),
        { wrapper }
      );

      act(() => {
        result.current.createMutation.mutate({ name: 'New Item' });
      });

      await waitFor(() => {
        expect(result.current.createMutation.isError).toBe(true);
      });

      expect(mockToast).toHaveBeenCalledWith({
        title: 'common.error',
        description: 'Create failed',
        variant: 'destructive',
      });
    });

    it('throws error if createFn not provided', async () => {
      const { result } = renderHook(
        () =>
          useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
            resourceName: 'items',
            queryKey: ['items'],
          }),
        { wrapper }
      );

      act(() => {
        result.current.createMutation.mutate({ name: 'New Item' });
      });

      await waitFor(() => {
        expect(result.current.createMutation.isError).toBe(true);
      });
    });
  });

  describe('updateMutation', () => {
    it('calls updateFn and shows success toast on success', async () => {
      const mockUpdateFn = jest.fn().mockResolvedValue({ id: 1, name: 'Updated Item' });
      const mockOnSuccess = jest.fn();
      const mockOnUpdateSuccess = jest.fn();

      const { result } = renderHook(
        () =>
          useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
            resourceName: 'items',
            queryKey: ['items'],
            updateFn: mockUpdateFn,
            onSuccess: mockOnSuccess,
            onUpdateSuccess: mockOnUpdateSuccess,
          }),
        { wrapper }
      );

      act(() => {
        result.current.updateMutation.mutate({ id: 1, data: { name: 'Updated Item' } });
      });

      await waitFor(() => {
        expect(result.current.updateMutation.isSuccess).toBe(true);
      });

      expect(mockUpdateFn).toHaveBeenCalledWith(1, { name: 'Updated Item' });
      expect(mockToast).toHaveBeenCalledWith({ title: 'items.updateSuccess' });
      expect(queryClient.invalidateQueries).toHaveBeenCalledWith({ queryKey: ['items'] });
      expect(mockOnSuccess).toHaveBeenCalled();
      expect(mockOnUpdateSuccess).toHaveBeenCalledWith({ id: 1, name: 'Updated Item' });
    });

    it('shows error toast on failure', async () => {
      const mockUpdateFn = jest.fn().mockRejectedValue(new Error('Update failed'));

      const { result } = renderHook(
        () =>
          useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
            resourceName: 'items',
            queryKey: ['items'],
            updateFn: mockUpdateFn,
          }),
        { wrapper }
      );

      act(() => {
        result.current.updateMutation.mutate({ id: 1, data: { name: 'Updated' } });
      });

      await waitFor(() => {
        expect(result.current.updateMutation.isError).toBe(true);
      });

      expect(mockToast).toHaveBeenCalledWith({
        title: 'common.error',
        description: 'Update failed',
        variant: 'destructive',
      });
    });
  });

  describe('deleteMutation', () => {
    it('calls deleteFn and shows success toast on success', async () => {
      const mockDeleteFn = jest.fn().mockResolvedValue(undefined);
      const mockOnDeleteSuccess = jest.fn();

      const { result } = renderHook(
        () =>
          useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
            resourceName: 'items',
            queryKey: ['items'],
            deleteFn: mockDeleteFn,
            onDeleteSuccess: mockOnDeleteSuccess,
          }),
        { wrapper }
      );

      act(() => {
        result.current.deleteMutation.mutate(1);
      });

      await waitFor(() => {
        expect(result.current.deleteMutation.isSuccess).toBe(true);
      });

      expect(mockDeleteFn).toHaveBeenCalledWith(1);
      expect(mockToast).toHaveBeenCalledWith({ title: 'items.deleteSuccess' });
      expect(queryClient.invalidateQueries).toHaveBeenCalledWith({ queryKey: ['items'] });
      expect(mockOnDeleteSuccess).toHaveBeenCalled();
    });

    it('shows error toast on failure', async () => {
      const mockDeleteFn = jest.fn().mockRejectedValue(new Error('Delete failed'));

      const { result } = renderHook(
        () =>
          useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
            resourceName: 'items',
            queryKey: ['items'],
            deleteFn: mockDeleteFn,
          }),
        { wrapper }
      );

      act(() => {
        result.current.deleteMutation.mutate(1);
      });

      await waitFor(() => {
        expect(result.current.deleteMutation.isError).toBe(true);
      });

      expect(mockToast).toHaveBeenCalledWith({
        title: 'common.error',
        description: 'Delete failed',
        variant: 'destructive',
      });
    });
  });

  describe('isMutating', () => {
    it('returns true when any mutation is pending', async () => {
      let resolvePromise: (value: TestItem) => void;
      const mockCreateFn = jest.fn().mockImplementation(
        () =>
          new Promise<TestItem>((resolve) => {
            resolvePromise = resolve;
          })
      );

      const { result } = renderHook(
        () =>
          useCrudMutations<TestItem, TestCreateData, TestUpdateData>({
            resourceName: 'items',
            queryKey: ['items'],
            createFn: mockCreateFn,
          }),
        { wrapper }
      );

      expect(result.current.isMutating).toBe(false);

      act(() => {
        result.current.createMutation.mutate({ name: 'New Item' });
      });

      // Wait for mutation to be pending
      await waitFor(() => {
        expect(result.current.isMutating).toBe(true);
      });

      // Resolve the promise
      act(() => {
        resolvePromise!({ id: 1, name: 'Item' });
      });

      await waitFor(() => {
        expect(result.current.isMutating).toBe(false);
      });
    });
  });
});
