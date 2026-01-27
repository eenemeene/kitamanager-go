<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'
import type { User, Group, UserMembership, Role } from '@/api/types'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Dropdown from 'primevue/dropdown'
import Tag from 'primevue/tag'

const props = defineProps<{
  visible: boolean
  user: User | null
}>()

const emit = defineEmits<{
  close: []
  updated: []
}>()

const toast = useToast()
const confirm = useConfirm()
const { t } = useI18n()
const loading = ref(false)

// Memberships data
const memberships = ref<UserMembership[]>([])

// All available groups (for add dialog)
const allGroups = ref<Group[]>([])

// Add to group dialog state
const addGroupDialogVisible = ref(false)
const selectedGroupForAdd = ref<Group | null>(null)
const selectedRoleForAdd = ref<Role>('member')

// Edit role dialog state
const editRoleDialogVisible = ref(false)
const editingMembership = ref<UserMembership | null>(null)
const editingRole = ref<Role>('member')

const roleOptions = computed(() => [
  { label: t('roles.admin'), value: 'admin' as Role },
  { label: t('roles.manager'), value: 'manager' as Role },
  { label: t('roles.member'), value: 'member' as Role }
])

const dialogTitle = computed(() =>
  props.user
    ? t('users.groupMembershipsFor', { name: props.user.name })
    : t('users.groupMemberships')
)

// Groups available to add (not already a member of)
const availableGroups = computed(() => {
  const memberGroupIds = new Set(memberships.value.map((m) => m.group_id))
  return allGroups.value.filter((g) => !memberGroupIds.has(g.id))
})

// Format group for dropdown display
function formatGroupOption(group: Group): string {
  const orgName = group.organization?.name || t('common.unknown')
  return `${group.name} (${orgName})`
}

watch(
  () => props.visible,
  async (visible) => {
    if (visible && props.user) {
      await loadData()
    }
  }
)

async function loadData() {
  if (!props.user) return

  loading.value = true
  try {
    // Load memberships and all organizations
    const [membershipsResponse, orgs] = await Promise.all([
      apiClient.getUserMemberships(props.user.id),
      apiClient.getOrganizations()
    ])

    memberships.value = membershipsResponse.memberships || []

    // Load groups from all organizations
    const groupPromises = orgs.map((org) => apiClient.getGroups(org.id))
    const groupsArrays = await Promise.all(groupPromises)
    allGroups.value = groupsArrays.flat()
  } catch {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: t('users.failedToLoadMemberships'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

function getRoleSeverity(role: Role): 'success' | 'info' | 'secondary' {
  switch (role) {
    case 'admin':
      return 'success'
    case 'manager':
      return 'info'
    default:
      return 'secondary'
  }
}

function getRoleLabel(role: Role): string {
  switch (role) {
    case 'admin':
      return t('roles.admin')
    case 'manager':
      return t('roles.manager')
    default:
      return t('roles.member')
  }
}

// Add to group functions
function openAddGroupDialog() {
  selectedGroupForAdd.value = null
  selectedRoleForAdd.value = 'member'
  addGroupDialogVisible.value = true
}

async function handleAddToGroup() {
  if (!props.user || !selectedGroupForAdd.value) return

  loading.value = true
  try {
    await apiClient.addUserToGroup(
      props.user.id,
      selectedGroupForAdd.value.id,
      selectedRoleForAdd.value
    )
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('users.addedToGroup'),
      life: 3000
    })
    addGroupDialogVisible.value = false
    await loadData()
    emit('updated')
  } catch {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: t('users.failedToAddToGroup'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Edit role functions
function openEditRoleDialog(membership: UserMembership) {
  editingMembership.value = membership
  editingRole.value = membership.role
  editRoleDialogVisible.value = true
}

async function handleUpdateRole() {
  if (!props.user || !editingMembership.value) return

  loading.value = true
  try {
    await apiClient.updateUserGroupRole(
      props.user.id,
      editingMembership.value.group_id,
      editingRole.value
    )
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('users.roleUpdated'),
      life: 3000
    })
    editRoleDialogVisible.value = false
    await loadData()
    emit('updated')
  } catch {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: t('users.failedToUpdateRole'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Remove from group
function confirmRemoveFromGroup(membership: UserMembership) {
  confirm.require({
    message: t('users.removeFromGroupConfirm', { name: membership.group.name }),
    header: t('users.confirmRemoval'),
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      if (!props.user) return

      loading.value = true
      try {
        await apiClient.removeUserFromGroup(props.user.id, membership.group_id)
        toast.add({
          severity: 'success',
          summary: t('common.success'),
          detail: t('users.removedFromGroup'),
          life: 3000
        })
        await loadData()
        emit('updated')
      } catch {
        toast.add({
          severity: 'error',
          summary: t('common.error'),
          detail: t('users.failedToRemoveFromGroup'),
          life: 3000
        })
      } finally {
        loading.value = false
      }
    }
  })
}
</script>

<template>
  <Dialog
    :visible="visible"
    :header="dialogTitle"
    modal
    :closable="true"
    :style="{ width: '750px' }"
    @update:visible="$emit('close')"
  >
    <div class="mb-3 flex justify-between">
      <p>{{ t('users.groupsUserBelongsTo') }}</p>
      <Button
        :label="t('users.addToGroup')"
        icon="pi pi-plus"
        size="small"
        :disabled="availableGroups.length === 0"
        @click="openAddGroupDialog"
      />
    </div>

    <DataTable
      :value="memberships"
      :loading="loading"
      striped-rows
      :paginator="memberships.length > 10"
      :rows="10"
    >
      <Column :header="t('common.organization')" style="width: 30%">
        <template #body="{ data }">
          {{ data.group.organization?.name || t('common.unknown') }}
        </template>
      </Column>
      <Column field="group.name" :header="t('groups.group')" style="width: 25%"></Column>
      <Column :header="t('roles.role')" style="width: 20%">
        <template #body="{ data }">
          <Tag :value="getRoleLabel(data.role)" :severity="getRoleSeverity(data.role)" />
        </template>
      </Column>
      <Column :header="t('common.actions')" style="width: 25%">
        <template #body="{ data }">
          <Button
            icon="pi pi-pencil"
            text
            rounded
            size="small"
            :title="t('users.editRole')"
            @click="openEditRoleDialog(data)"
          />
          <Button
            icon="pi pi-trash"
            text
            rounded
            size="small"
            severity="danger"
            :title="t('common.remove')"
            @click="confirmRemoveFromGroup(data)"
          />
        </template>
      </Column>
      <template #empty>
        <div class="text-center text-muted py-4">{{ t('users.notMemberOfAnyGroups') }}</div>
      </template>
    </DataTable>

    <template #footer>
      <div class="dialog-footer">
        <Button :label="t('common.close')" text @click="$emit('close')" />
      </div>
    </template>
  </Dialog>

  <!-- Add to Group Dialog -->
  <Dialog
    :visible="addGroupDialogVisible"
    :header="t('users.addUserToGroup')"
    modal
    :closable="true"
    :style="{ width: '450px' }"
    @update:visible="addGroupDialogVisible = false"
  >
    <div class="form-grid">
      <div class="field">
        <label for="add-group">{{ t('groups.group') }}</label>
        <Dropdown
          id="add-group"
          v-model="selectedGroupForAdd"
          :options="availableGroups"
          :option-label="formatGroupOption"
          :placeholder="t('users.selectGroup')"
          filter
          class="w-full"
        />
      </div>

      <div class="field">
        <label for="add-role">{{ t('roles.role') }}</label>
        <Dropdown
          id="add-role"
          v-model="selectedRoleForAdd"
          :options="roleOptions"
          option-label="label"
          option-value="value"
          :placeholder="t('users.selectRole')"
          class="w-full"
        />
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <Button :label="t('common.cancel')" text @click="addGroupDialogVisible = false" />
        <Button
          :label="t('common.add')"
          :loading="loading"
          :disabled="!selectedGroupForAdd"
          @click="handleAddToGroup"
        />
      </div>
    </template>
  </Dialog>

  <!-- Edit Role Dialog -->
  <Dialog
    :visible="editRoleDialogVisible"
    :header="t('users.editRole')"
    modal
    :closable="true"
    :style="{ width: '400px' }"
    @update:visible="editRoleDialogVisible = false"
  >
    <div class="form-grid">
      <p v-if="editingMembership">
        {{ t('users.changeRoleFor') }} <strong>{{ editingMembership.group.name }}</strong
        >:
      </p>
      <div class="field">
        <label for="edit-role">{{ t('roles.role') }}</label>
        <Dropdown
          id="edit-role"
          v-model="editingRole"
          :options="roleOptions"
          option-label="label"
          option-value="value"
          class="w-full"
        />
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <Button :label="t('common.cancel')" text @click="editRoleDialogVisible = false" />
        <Button :label="t('common.save')" :loading="loading" @click="handleUpdateRole" />
      </div>
    </template>
  </Dialog>
</template>

<style scoped>
.mb-3 {
  margin-bottom: 1rem;
}

.py-4 {
  padding-top: 1.5rem;
  padding-bottom: 1.5rem;
}

.flex {
  display: flex;
}

.justify-between {
  justify-content: space-between;
  align-items: center;
}

.text-center {
  text-align: center;
}

.text-muted {
  color: var(--text-color-secondary);
}

.w-full {
  width: 100%;
}
</style>
