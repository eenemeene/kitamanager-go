<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useToast } from 'primevue/usetoast'
import { apiClient } from '@/api/client'
import { useUiStore } from '@/stores/ui'
import type { User, Group, Organization } from '@/api/types'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import PickList from 'primevue/picklist'
import TabView from 'primevue/tabview'
import TabPanel from 'primevue/tabpanel'

const props = defineProps<{
  visible: boolean
  user: User | null
}>()

const emit = defineEmits<{
  close: []
  updated: []
}>()

const toast = useToast()
const uiStore = useUiStore()
const loading = ref(false)

// Groups data: [available, assigned]
const groupsData = ref<[Group[], Group[]]>([[], []])
// Organizations data: [available, assigned]
const orgsData = ref<[Organization[], Organization[]]>([[], []])

const dialogTitle = computed(() =>
  props.user ? `Manage Memberships: ${props.user.name}` : 'Manage Memberships'
)

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
    const [allGroups, allOrgs, userData] = await Promise.all([
      apiClient.getGroups(),
      apiClient.getOrganizations(),
      apiClient.getUser(props.user.id)
    ])

    const assignedGroupIds = new Set((userData.groups || []).map((g) => g.id))
    const assignedOrgIds = new Set((userData.organizations || []).map((o) => o.id))

    groupsData.value = [allGroups.filter((g) => !assignedGroupIds.has(g.id)), userData.groups || []]

    orgsData.value = [
      allOrgs.filter((o) => !assignedOrgIds.has(o.id)),
      userData.organizations || []
    ]
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to load membership data',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  if (!props.user) return

  loading.value = true
  try {
    // Get current assignments from API
    const userData = await apiClient.getUser(props.user.id)
    const currentGroupIds = new Set((userData.groups || []).map((g) => g.id))
    const currentOrgIds = new Set((userData.organizations || []).map((o) => o.id))

    // New assignments from PickList
    const newGroupIds = new Set(groupsData.value[1].map((g) => g.id))
    const newOrgIds = new Set(orgsData.value[1].map((o) => o.id))

    // Calculate additions and removals for groups
    const groupsToAdd = [...newGroupIds].filter((id) => !currentGroupIds.has(id))
    const groupsToRemove = [...currentGroupIds].filter((id) => !newGroupIds.has(id))

    // Calculate additions and removals for organizations
    const orgsToAdd = [...newOrgIds].filter((id) => !currentOrgIds.has(id))
    const orgsToRemove = [...currentOrgIds].filter((id) => !newOrgIds.has(id))

    // Apply changes
    await Promise.all([
      ...groupsToAdd.map((gid) => apiClient.addUserToGroup(props.user!.id, gid)),
      ...groupsToRemove.map((gid) => apiClient.removeUserFromGroup(props.user!.id, gid)),
      ...orgsToAdd.map((oid) => apiClient.addUserToOrganization(props.user!.id, oid)),
      ...orgsToRemove.map((oid) => apiClient.removeUserFromOrganization(props.user!.id, oid))
    ])

    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Memberships updated successfully',
      life: 3000
    })

    // Refresh organizations in the UI store to update the sidebar selector
    await uiStore.fetchOrganizations()

    emit('updated')
    emit('close')
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to update memberships',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <Dialog
    :visible="visible"
    :header="dialogTitle"
    modal
    :closable="true"
    :style="{ width: '700px' }"
    @update:visible="$emit('close')"
  >
    <TabView>
      <TabPanel value="groups" header="Groups">
        <p class="mb-3">Move groups between Available and Assigned lists:</p>
        <PickList
          v-model="groupsData"
          data-key="id"
          breakpoint="575px"
          :show-source-controls="false"
          :show-target-controls="false"
        >
          <template #sourceheader>Available Groups</template>
          <template #targetheader>Assigned Groups</template>
          <template #item="{ item }">
            <span>{{ item.name }}</span>
          </template>
        </PickList>
      </TabPanel>
      <TabPanel value="organizations" header="Organizations">
        <p class="mb-3">Move organizations between Available and Assigned lists:</p>
        <PickList
          v-model="orgsData"
          data-key="id"
          breakpoint="575px"
          :show-source-controls="false"
          :show-target-controls="false"
        >
          <template #sourceheader>Available Organizations</template>
          <template #targetheader>Assigned Organizations</template>
          <template #item="{ item }">
            <span>{{ item.name }}</span>
          </template>
        </PickList>
      </TabPanel>
    </TabView>

    <template #footer>
      <div class="dialog-footer">
        <Button label="Cancel" text @click="$emit('close')" />
        <Button label="Save" :loading="loading" @click="handleSave" />
      </div>
    </template>
  </Dialog>
</template>

<style scoped>
.mb-3 {
  margin-bottom: 1rem;
}
</style>
