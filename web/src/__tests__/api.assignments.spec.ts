import { describe, it, expect, vi, beforeEach } from 'vitest'

// Create a mock axios instance
const mockAxiosInstance = {
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
  interceptors: {
    request: { use: vi.fn() },
    response: { use: vi.fn() }
  }
}

// Mock axios before importing the client
vi.mock('axios', () => ({
  default: {
    create: () => mockAxiosInstance
  }
}))

describe('API Client - Assignment Methods', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('User-Group Assignments', () => {
    it('should call POST /users/:id/groups for addUserToGroup', async () => {
      mockAxiosInstance.post.mockResolvedValue({ data: { message: 'user added to group' } })

      // Dynamic import to get fresh instance with mocks
      const { apiClient } = await import('../api/client')
      await apiClient.addUserToGroup(1, 2)

      expect(mockAxiosInstance.post).toHaveBeenCalledWith('/users/1/groups', { group_id: 2 })
    })

    it('should call DELETE /users/:id/groups/:gid for removeUserFromGroup', async () => {
      mockAxiosInstance.delete.mockResolvedValue({ data: {} })

      const { apiClient } = await import('../api/client')
      await apiClient.removeUserFromGroup(1, 2)

      expect(mockAxiosInstance.delete).toHaveBeenCalledWith('/users/1/groups/2')
    })
  })

  describe('User-Organization Assignments', () => {
    it('should call POST /users/:id/organizations for addUserToOrganization', async () => {
      mockAxiosInstance.post.mockResolvedValue({
        data: { message: 'user added to organization' }
      })

      const { apiClient } = await import('../api/client')
      await apiClient.addUserToOrganization(1, 3)

      expect(mockAxiosInstance.post).toHaveBeenCalledWith('/users/1/organizations', {
        organization_id: 3
      })
    })

    it('should call DELETE /users/:id/organizations/:oid for removeUserFromOrganization', async () => {
      mockAxiosInstance.delete.mockResolvedValue({ data: {} })

      const { apiClient } = await import('../api/client')
      await apiClient.removeUserFromOrganization(1, 3)

      expect(mockAxiosInstance.delete).toHaveBeenCalledWith('/users/1/organizations/3')
    })
  })

  describe('Group-Organization Assignments', () => {
    it('should call POST /groups/:id/organizations for addGroupToOrganization', async () => {
      mockAxiosInstance.post.mockResolvedValue({
        data: { message: 'group added to organization' }
      })

      const { apiClient } = await import('../api/client')
      await apiClient.addGroupToOrganization(2, 3)

      expect(mockAxiosInstance.post).toHaveBeenCalledWith('/groups/2/organizations', {
        organization_id: 3
      })
    })

    it('should call DELETE /groups/:id/organizations/:oid for removeGroupFromOrganization', async () => {
      mockAxiosInstance.delete.mockResolvedValue({ data: {} })

      const { apiClient } = await import('../api/client')
      await apiClient.removeGroupFromOrganization(2, 3)

      expect(mockAxiosInstance.delete).toHaveBeenCalledWith('/groups/2/organizations/3')
    })
  })
})
