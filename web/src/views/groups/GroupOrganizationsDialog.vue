<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useToast } from 'primevue/usetoast'
import { apiClient } from '@/api/client'
import type { Group, Organization } from '@/api/types'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import PickList from 'primevue/picklist'

const props = defineProps<{
  visible: boolean
  group: Group | null
}>()

const emit = defineEmits<{
  close: []
  updated: []
}>()

const toast = useToast()
const loading = ref(false)

// Organizations data: [available, assigned]
const orgsData = ref<[Organization[], Organization[]]>([[], []])

const dialogTitle = computed(() =>
  props.group ? `Manage Organizations: ${props.group.name}` : 'Manage Organizations'
)

watch(
  () => props.visible,
  async (visible) => {
    if (visible && props.group) {
      await loadData()
    }
  }
)

async function loadData() {
  if (!props.group) return

  loading.value = true
  try {
    const [allOrgs, groupData] = await Promise.all([
      apiClient.getOrganizations(),
      apiClient.getGroup(props.group.id)
    ])

    const assignedOrgIds = new Set((groupData.organizations || []).map((o) => o.id))

    orgsData.value = [
      allOrgs.filter((o) => !assignedOrgIds.has(o.id)),
      groupData.organizations || []
    ]
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to load organization data',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  if (!props.group) return

  loading.value = true
  try {
    // Get current assignments from API
    const groupData = await apiClient.getGroup(props.group.id)
    const currentOrgIds = new Set((groupData.organizations || []).map((o) => o.id))

    // New assignments from PickList
    const newOrgIds = new Set(orgsData.value[1].map((o) => o.id))

    // Calculate additions and removals
    const orgsToAdd = [...newOrgIds].filter((id) => !currentOrgIds.has(id))
    const orgsToRemove = [...currentOrgIds].filter((id) => !newOrgIds.has(id))

    // Apply changes
    await Promise.all([
      ...orgsToAdd.map((oid) => apiClient.addGroupToOrganization(props.group!.id, oid)),
      ...orgsToRemove.map((oid) => apiClient.removeGroupFromOrganization(props.group!.id, oid))
    ])

    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Organizations updated successfully',
      life: 3000
    })

    emit('updated')
    emit('close')
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to update organizations',
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
