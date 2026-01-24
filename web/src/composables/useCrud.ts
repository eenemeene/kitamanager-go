import { ref, type Ref } from 'vue'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'

interface CrudConfig<T, CreateDto, UpdateDto> {
  entityName: string
  fetchAll: () => Promise<T[]>
  create: (data: CreateDto) => Promise<T>
  update: (id: number, data: UpdateDto) => Promise<T>
  remove: (id: number) => Promise<void>
  getId?: (item: T) => number
}

export function useCrud<T, CreateDto, UpdateDto>(config: CrudConfig<T, CreateDto, UpdateDto>) {
  const toast = useToast()
  const confirm = useConfirm()

  const items: Ref<T[]> = ref([])
  const loading = ref(false)
  const dialogVisible = ref(false)
  const editingItem: Ref<T | null> = ref(null)

  const getId = config.getId || ((item: T) => (item as { id: number }).id)

  async function fetchItems() {
    loading.value = true
    try {
      items.value = await config.fetchAll()
    } catch {
      toast.add({
        severity: 'error',
        summary: 'Error',
        detail: `Failed to load ${config.entityName}s`,
        life: 3000
      })
    } finally {
      loading.value = false
    }
  }

  function openCreateDialog() {
    editingItem.value = null
    dialogVisible.value = true
  }

  function openEditDialog(item: T) {
    editingItem.value = item
    dialogVisible.value = true
  }

  function closeDialog() {
    dialogVisible.value = false
    editingItem.value = null
  }

  async function saveItem(data: CreateDto | UpdateDto) {
    try {
      if (editingItem.value) {
        await config.update(getId(editingItem.value), data as UpdateDto)
        toast.add({
          severity: 'success',
          summary: 'Success',
          detail: `${config.entityName} updated successfully`,
          life: 3000
        })
      } else {
        await config.create(data as CreateDto)
        toast.add({
          severity: 'success',
          summary: 'Success',
          detail: `${config.entityName} created successfully`,
          life: 3000
        })
      }
      closeDialog()
      await fetchItems()
    } catch {
      toast.add({
        severity: 'error',
        summary: 'Error',
        detail: `Failed to save ${config.entityName}`,
        life: 3000
      })
    }
  }

  function confirmDelete(item: T) {
    confirm.require({
      message: `Are you sure you want to delete this ${config.entityName}?`,
      header: 'Confirm Delete',
      icon: 'pi pi-exclamation-triangle',
      acceptClass: 'p-button-danger',
      accept: async () => {
        try {
          await config.remove(getId(item))
          toast.add({
            severity: 'success',
            summary: 'Success',
            detail: `${config.entityName} deleted successfully`,
            life: 3000
          })
          await fetchItems()
        } catch {
          toast.add({
            severity: 'error',
            summary: 'Error',
            detail: `Failed to delete ${config.entityName}`,
            life: 3000
          })
        }
      }
    })
  }

  return {
    items,
    loading,
    dialogVisible,
    editingItem,
    fetchItems,
    openCreateDialog,
    openEditDialog,
    closeDialog,
    saveItem,
    confirmDelete
  }
}
