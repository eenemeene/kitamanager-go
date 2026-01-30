'use client';

import { useState, useCallback } from 'react';
import type { UseFormReset, FieldValues } from 'react-hook-form';

export interface UseCrudDialogsConfig<TItem, TFormData extends FieldValues> {
  /** Form reset function from react-hook-form */
  reset: UseFormReset<TFormData>;
  /** Convert an item to form data for editing */
  itemToFormData: (item: TItem) => TFormData;
  /** Default values for creating a new item */
  defaultValues: TFormData;
}

export interface UseCrudDialogsResult<TItem> {
  /** Whether the create/edit dialog is open */
  isDialogOpen: boolean;
  /** Set the dialog open state */
  setIsDialogOpen: (open: boolean) => void;
  /** Whether the delete confirmation dialog is open */
  isDeleteDialogOpen: boolean;
  /** Set the delete dialog open state */
  setIsDeleteDialogOpen: (open: boolean) => void;
  /** The item currently being edited (null if creating) */
  editingItem: TItem | null;
  /** The item pending deletion */
  deletingItem: TItem | null;
  /** Handler for creating a new item - opens dialog with default values */
  handleCreate: () => void;
  /** Handler for editing an item - opens dialog with item values */
  handleEdit: (item: TItem) => void;
  /** Handler for deleting an item - opens delete confirmation dialog */
  handleDelete: (item: TItem) => void;
  /** Close the create/edit dialog and reset state */
  closeDialog: () => void;
  /** Close the delete dialog and reset state */
  closeDeleteDialog: () => void;
  /** True if editing an existing item, false if creating new */
  isEditing: boolean;
}

/**
 * Custom hook for managing CRUD dialog state.
 * Handles create/edit dialog and delete confirmation dialog state.
 */
export function useCrudDialogs<TItem, TFormData extends FieldValues>({
  reset,
  itemToFormData,
  defaultValues,
}: UseCrudDialogsConfig<TItem, TFormData>): UseCrudDialogsResult<TItem> {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<TItem | null>(null);
  const [deletingItem, setDeletingItem] = useState<TItem | null>(null);

  const handleCreate = useCallback(() => {
    setEditingItem(null);
    reset(defaultValues);
    setIsDialogOpen(true);
  }, [reset, defaultValues]);

  const handleEdit = useCallback(
    (item: TItem) => {
      setEditingItem(item);
      reset(itemToFormData(item));
      setIsDialogOpen(true);
    },
    [reset, itemToFormData]
  );

  const handleDelete = useCallback((item: TItem) => {
    setDeletingItem(item);
    setIsDeleteDialogOpen(true);
  }, []);

  const closeDialog = useCallback(() => {
    setIsDialogOpen(false);
    setEditingItem(null);
    reset(defaultValues);
  }, [reset, defaultValues]);

  const closeDeleteDialog = useCallback(() => {
    setIsDeleteDialogOpen(false);
    setDeletingItem(null);
  }, []);

  return {
    isDialogOpen,
    setIsDialogOpen,
    isDeleteDialogOpen,
    setIsDeleteDialogOpen,
    editingItem,
    deletingItem,
    handleCreate,
    handleEdit,
    handleDelete,
    closeDialog,
    closeDeleteDialog,
    isEditing: editingItem !== null,
  };
}
