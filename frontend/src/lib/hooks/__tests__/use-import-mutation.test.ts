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

  // The file-input branch of this hook (triggerFileInput, handleFileChange)
  // was previously at 0% branch coverage despite being on the user-visible
  // import flow. These tests exercise it directly so a regression in
  // "click button → pick file → mutation fires" surfaces in CI rather than
  // when a kita admin can no longer import their YAML.

  it('triggerFileInput clicks the file input ref', () => {
    const { result } = renderHook(
      () =>
        useImportMutation({
          importFn: jest.fn(),
          invalidateQueryKeys: [['children', 1]],
          resourceNameKey: 'children.title',
          errorMessageKey: 'children.importError',
        }),
      { wrapper }
    );

    const click = jest.fn();
    // The ref is normally bound to a real <input> by JSX; for a unit test we
    // inject a stub so we can observe the click without a DOM mount.
    (result.current.fileInputRef as unknown as { current: { click: () => void } }).current = {
      click,
    };

    result.current.triggerFileInput();
    expect(click).toHaveBeenCalledTimes(1);
  });

  it('triggerFileInput is a no-op when ref is unbound', () => {
    const { result } = renderHook(
      () =>
        useImportMutation({
          importFn: jest.fn(),
          invalidateQueryKeys: [['children', 1]],
          resourceNameKey: 'children.title',
          errorMessageKey: 'children.importError',
        }),
      { wrapper }
    );

    expect(() => result.current.triggerFileInput()).not.toThrow();
  });

  it('handleFileChange fires the mutation with the picked file and resets the input', async () => {
    const importFn = jest.fn().mockResolvedValue({});

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

    const file = new File(['payload'], 'kids.yaml', { type: 'text/yaml' });
    const target = { files: [file], value: 'kids.yaml' } as unknown as HTMLInputElement;
    const event = { target } as unknown as React.ChangeEvent<HTMLInputElement>;

    act(() => {
      result.current.handleFileChange(event);
    });

    await waitFor(() => {
      expect(result.current.mutation.isSuccess).toBe(true);
    });

    // TanStack Query's mutationFn is invoked with (variables, context); the
    // first arg is the file, the second is a context object we don't care
    // about here. Check the first arg explicitly rather than via
    // toHaveBeenCalledWith, which is positional and would force matching the
    // context too.
    expect(importFn).toHaveBeenCalledTimes(1);
    expect(importFn.mock.calls[0][0]).toBe(file);
    // Input value must be cleared so picking the same file twice still fires
    // a change event the second time.
    expect(target.value).toBe('');
  });

  it('handleFileChange is a no-op when no file is selected', () => {
    const importFn = jest.fn();

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

    const event = {
      target: { files: [], value: '' },
    } as unknown as React.ChangeEvent<HTMLInputElement>;

    act(() => {
      result.current.handleFileChange(event);
    });

    expect(importFn).not.toHaveBeenCalled();
  });
});
