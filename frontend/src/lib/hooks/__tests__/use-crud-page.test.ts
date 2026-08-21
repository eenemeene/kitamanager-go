import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient } from '@tanstack/react-query';
import { z } from 'zod';
import { useCrudPage } from '../use-crud-page';
import { createTestQueryClient, createHookWrapper } from '@/test-utils';
import type { PaginatedResponse } from '@/lib/api/types';

// Mock next/navigation
jest.mock('next/navigation', () => ({
  useParams: () => ({ orgId: '1' }),
}));

// Mock toast
const mockToast = jest.fn();
jest.mock('../use-toast', () => ({
  useToast: () => ({ toast: mockToast }),
}));

// Mock next-intl
jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string) => key,
}));

// Mock getErrorMessage used by useCrudMutations
jest.mock('@/lib/api/client', () => ({
  getErrorMessage: (_error: unknown, fallback: string) => fallback,
}));

// --- Test types ---
interface TestItem {
  id: number;
  name: string;
}

interface TestFormData {
  name: string;
}

const testSchema = z.object({ name: z.string().min(1) });

const defaultValues: TestFormData = { name: '' };

type TestConfig = Parameters<
  typeof useCrudPage<TestItem, TestFormData, TestFormData, TestFormData>
>[0];

function createMockConfig(overrides: Partial<TestConfig> = {}): TestConfig {
  return {
    resourceName: 'testItems',
    schema: testSchema,
    defaultValues,
    itemToFormData: (item: TestItem): TestFormData => ({ name: item.name }),
    listFn: jest.fn().mockResolvedValue({
      data: [],
      total: 0,
      page: 1,
      limit: 30,
      total_pages: 1,
    } as PaginatedResponse<TestItem>),
    createFn: jest.fn().mockResolvedValue({ id: 1, name: 'Created' }),
    updateFn: jest.fn().mockResolvedValue({ id: 1, name: 'Updated' }),
    deleteFn: jest.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

describe('useCrudPage', () => {
  let queryClient: QueryClient;
  let wrapper: ReturnType<typeof createHookWrapper>;

  beforeEach(() => {
    queryClient = createTestQueryClient();
    wrapper = createHookWrapper(queryClient);
    mockToast.mockClear();
  });

  afterEach(() => {
    queryClient.clear();
  });

  it('returns orgId parsed from params', () => {
    const config = createMockConfig();
    const { result } = renderHook(() => useCrudPage(config), { wrapper });

    expect(result.current.orgId).toBe(1);
  });

  it('returns initial empty state (items undefined, isLoading, page=1)', () => {
    // Use a listFn that never resolves so the query stays loading
    const config = createMockConfig({
      listFn: jest.fn().mockReturnValue(new Promise(() => {})),
    });
    const { result } = renderHook(() => useCrudPage(config), { wrapper });

    expect(result.current.list.items).toBeUndefined();
    expect(result.current.list.isLoading).toBe(true);
    expect(result.current.list.page).toBe(1);
  });

  it('returns form utilities (register, handleSubmit, errors, setValue, watch)', () => {
    const config = createMockConfig();
    const { result } = renderHook(() => useCrudPage(config), { wrapper });

    expect(typeof result.current.form.register).toBe('function');
    expect(typeof result.current.form.handleSubmit).toBe('function');
    expect(result.current.form.errors).toBeDefined();
    expect(typeof result.current.form.setValue).toBe('function');
    expect(typeof result.current.form.watch).toBe('function');
  });

  it('returns dialogs object with expected methods', () => {
    const config = createMockConfig();
    const { result } = renderHook(() => useCrudPage(config), { wrapper });

    const { dialogs } = result.current;
    expect(typeof dialogs.handleCreate).toBe('function');
    expect(typeof dialogs.handleEdit).toBe('function');
    expect(typeof dialogs.handleDelete).toBe('function');
    expect(typeof dialogs.closeDialog).toBe('function');
    expect(typeof dialogs.closeDeleteDialog).toBe('function');
    expect(typeof dialogs.setIsDialogOpen).toBe('function');
    expect(typeof dialogs.setIsDeleteDialogOpen).toBe('function');
    expect(dialogs.isDialogOpen).toBe(false);
    expect(dialogs.isDeleteDialogOpen).toBe(false);
    expect(dialogs.editingItem).toBeNull();
    expect(dialogs.deletingItem).toBeNull();
    expect(dialogs.isEditing).toBe(false);
  });

  it('returns mutations object with expected properties', () => {
    const config = createMockConfig();
    const { result } = renderHook(() => useCrudPage(config), { wrapper });

    const { mutations } = result.current;
    expect(mutations.createMutation).toBeDefined();
    expect(mutations.updateMutation).toBeDefined();
    expect(mutations.deleteMutation).toBeDefined();
    expect(typeof mutations.isMutating).toBe('boolean');
  });

  it('onSubmit calls createMutation when no editingItem', async () => {
    const mockCreateFn = jest.fn().mockResolvedValue({ id: 1, name: 'New' });
    const config = createMockConfig({ createFn: mockCreateFn });
    const { result } = renderHook(() => useCrudPage(config), { wrapper });

    act(() => {
      result.current.form.onSubmit({ name: 'New' });
    });

    await waitFor(() => {
      expect(mockCreateFn).toHaveBeenCalledWith(1, { name: 'New' });
    });
  });

  it('onSubmit calls updateMutation when editingItem is set', async () => {
    const mockUpdateFn = jest.fn().mockResolvedValue({ id: 5, name: 'Edited' });
    const config = createMockConfig({ updateFn: mockUpdateFn });
    const { result } = renderHook(() => useCrudPage(config), { wrapper });

    // First, set an editing item via dialogs.handleEdit
    act(() => {
      result.current.dialogs.handleEdit({ id: 5, name: 'Old' });
    });

    act(() => {
      result.current.form.onSubmit({ name: 'Edited' });
    });

    await waitFor(() => {
      expect(mockUpdateFn).toHaveBeenCalledWith(1, 5, { name: 'Edited' });
    });
  });

  it('setPage updates page', () => {
    const config = createMockConfig();
    const { result } = renderHook(() => useCrudPage(config), { wrapper });

    expect(result.current.list.page).toBe(1);

    act(() => {
      result.current.list.setPage(3);
    });

    expect(result.current.list.page).toBe(3);
  });
});
