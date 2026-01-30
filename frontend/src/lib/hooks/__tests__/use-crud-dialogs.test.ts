import { renderHook, act } from '@testing-library/react';
import type { FieldValues } from 'react-hook-form';
import { useCrudDialogs } from '../use-crud-dialogs';

interface TestItem {
  id: number;
  name: string;
  active: boolean;
}

interface TestFormData extends FieldValues {
  name: string;
  active: boolean;
}

describe('useCrudDialogs', () => {
  const defaultValues: TestFormData = { name: '', active: true };
  const mockReset = jest.fn();
  const itemToFormData = (item: TestItem): TestFormData => ({
    name: item.name,
    active: item.active,
  });

  beforeEach(() => {
    mockReset.mockClear();
  });

  it('initializes with all dialogs closed', () => {
    const { result } = renderHook(() =>
      useCrudDialogs<TestItem, TestFormData>({
        reset: mockReset,
        itemToFormData,
        defaultValues,
      })
    );

    expect(result.current.isDialogOpen).toBe(false);
    expect(result.current.isDeleteDialogOpen).toBe(false);
    expect(result.current.editingItem).toBeNull();
    expect(result.current.deletingItem).toBeNull();
    expect(result.current.isEditing).toBe(false);
  });

  describe('handleCreate', () => {
    it('opens dialog with default values', () => {
      const { result } = renderHook(() =>
        useCrudDialogs<TestItem, TestFormData>({
          reset: mockReset,
          itemToFormData,
          defaultValues,
        })
      );

      act(() => {
        result.current.handleCreate();
      });

      expect(result.current.isDialogOpen).toBe(true);
      expect(result.current.editingItem).toBeNull();
      expect(result.current.isEditing).toBe(false);
      expect(mockReset).toHaveBeenCalledWith(defaultValues);
    });
  });

  describe('handleEdit', () => {
    it('opens dialog with item values', () => {
      const testItem: TestItem = { id: 1, name: 'Test Item', active: false };

      const { result } = renderHook(() =>
        useCrudDialogs<TestItem, TestFormData>({
          reset: mockReset,
          itemToFormData,
          defaultValues,
        })
      );

      act(() => {
        result.current.handleEdit(testItem);
      });

      expect(result.current.isDialogOpen).toBe(true);
      expect(result.current.editingItem).toBe(testItem);
      expect(result.current.isEditing).toBe(true);
      expect(mockReset).toHaveBeenCalledWith({ name: 'Test Item', active: false });
    });
  });

  describe('handleDelete', () => {
    it('opens delete dialog with item', () => {
      const testItem: TestItem = { id: 1, name: 'Test Item', active: true };

      const { result } = renderHook(() =>
        useCrudDialogs<TestItem, TestFormData>({
          reset: mockReset,
          itemToFormData,
          defaultValues,
        })
      );

      act(() => {
        result.current.handleDelete(testItem);
      });

      expect(result.current.isDeleteDialogOpen).toBe(true);
      expect(result.current.deletingItem).toBe(testItem);
    });
  });

  describe('closeDialog', () => {
    it('closes dialog and resets state', () => {
      const testItem: TestItem = { id: 1, name: 'Test Item', active: true };

      const { result } = renderHook(() =>
        useCrudDialogs<TestItem, TestFormData>({
          reset: mockReset,
          itemToFormData,
          defaultValues,
        })
      );

      // Open edit dialog first
      act(() => {
        result.current.handleEdit(testItem);
      });

      expect(result.current.isDialogOpen).toBe(true);
      expect(result.current.editingItem).toBe(testItem);

      // Clear mock to track only closeDialog reset call
      mockReset.mockClear();

      act(() => {
        result.current.closeDialog();
      });

      expect(result.current.isDialogOpen).toBe(false);
      expect(result.current.editingItem).toBeNull();
      expect(result.current.isEditing).toBe(false);
      expect(mockReset).toHaveBeenCalledWith(defaultValues);
    });
  });

  describe('closeDeleteDialog', () => {
    it('closes delete dialog and resets state', () => {
      const testItem: TestItem = { id: 1, name: 'Test Item', active: true };

      const { result } = renderHook(() =>
        useCrudDialogs<TestItem, TestFormData>({
          reset: mockReset,
          itemToFormData,
          defaultValues,
        })
      );

      act(() => {
        result.current.handleDelete(testItem);
      });

      expect(result.current.isDeleteDialogOpen).toBe(true);
      expect(result.current.deletingItem).toBe(testItem);

      act(() => {
        result.current.closeDeleteDialog();
      });

      expect(result.current.isDeleteDialogOpen).toBe(false);
      expect(result.current.deletingItem).toBeNull();
    });
  });

  describe('setIsDialogOpen', () => {
    it('allows manually controlling dialog state', () => {
      const { result } = renderHook(() =>
        useCrudDialogs<TestItem, TestFormData>({
          reset: mockReset,
          itemToFormData,
          defaultValues,
        })
      );

      act(() => {
        result.current.setIsDialogOpen(true);
      });

      expect(result.current.isDialogOpen).toBe(true);

      act(() => {
        result.current.setIsDialogOpen(false);
      });

      expect(result.current.isDialogOpen).toBe(false);
    });
  });

  describe('setIsDeleteDialogOpen', () => {
    it('allows manually controlling delete dialog state', () => {
      const { result } = renderHook(() =>
        useCrudDialogs<TestItem, TestFormData>({
          reset: mockReset,
          itemToFormData,
          defaultValues,
        })
      );

      act(() => {
        result.current.setIsDeleteDialogOpen(true);
      });

      expect(result.current.isDeleteDialogOpen).toBe(true);

      act(() => {
        result.current.setIsDeleteDialogOpen(false);
      });

      expect(result.current.isDeleteDialogOpen).toBe(false);
    });
  });
});
