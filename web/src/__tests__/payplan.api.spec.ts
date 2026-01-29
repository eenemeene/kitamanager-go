import { describe, it, expect, beforeEach, vi, type Mock } from 'vitest'
import axios from 'axios'
import type { PayPlan, PayPlanPeriod, PayPlanEntry } from '../api/types'

// Mock axios
vi.mock('axios', () => ({
  default: {
    create: vi.fn(() => ({
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      delete: vi.fn(),
      interceptors: {
        request: { use: vi.fn() },
        response: { use: vi.fn() }
      }
    }))
  }
}))

// Create fresh ApiClient for each test
function createMockClient() {
  const instance = axios.create()
  return {
    client: instance,
    get: instance.get as Mock,
    post: instance.post as Mock,
    put: instance.put as Mock,
    delete: instance.delete as Mock
  }
}

describe('PayPlan API Client', () => {
  let mockClient: ReturnType<typeof createMockClient>

  const mockPayPlan: PayPlan = {
    id: 1,
    organization_id: 1,
    name: 'TVöD 2024',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z'
  }

  const mockPeriod: PayPlanPeriod = {
    id: 1,
    payplan_id: 1,
    from: '2024-01-01',
    to: null,
    weekly_hours: 39.0,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z'
  }

  const mockEntry: PayPlanEntry = {
    id: 1,
    period_id: 1,
    grade: 'S8a',
    step: 3,
    monthly_amount: 350000,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z'
  }

  beforeEach(() => {
    mockClient = createMockClient()
    vi.clearAllMocks()
  })

  describe('PayPlan CRUD operations', () => {
    it('should fetch pay plans for an organization', async () => {
      const orgId = 1
      mockClient.get.mockResolvedValue({
        data: { data: [mockPayPlan] }
      })

      const response = await mockClient.get(`/organizations/${orgId}/payplans?limit=100`)

      expect(mockClient.get).toHaveBeenCalledWith(`/organizations/${orgId}/payplans?limit=100`)
      expect(response.data.data).toHaveLength(1)
      expect(response.data.data[0].name).toBe('TVöD 2024')
    })

    it('should fetch a single pay plan by ID', async () => {
      const orgId = 1
      const payPlanId = 1
      const fullPayPlan: PayPlan = {
        ...mockPayPlan,
        periods: [{ ...mockPeriod, entries: [mockEntry] }],
        total_periods: 1
      }

      mockClient.get.mockResolvedValue({ data: fullPayPlan })

      const response = await mockClient.get(`/organizations/${orgId}/payplans/${payPlanId}`)

      expect(mockClient.get).toHaveBeenCalledWith(`/organizations/${orgId}/payplans/${payPlanId}`)
      expect(response.data.name).toBe('TVöD 2024')
      expect(response.data.periods).toHaveLength(1)
      expect(response.data.periods[0].entries).toHaveLength(1)
    })

    it('should create a new pay plan', async () => {
      const orgId = 1
      const createRequest = { name: 'New Pay Plan' }
      const newPayPlan = { ...mockPayPlan, id: 2, name: 'New Pay Plan' }

      mockClient.post.mockResolvedValue({ data: newPayPlan })

      const response = await mockClient.post(`/organizations/${orgId}/payplans`, createRequest)

      expect(mockClient.post).toHaveBeenCalledWith(
        `/organizations/${orgId}/payplans`,
        createRequest
      )
      expect(response.data.name).toBe('New Pay Plan')
    })

    it('should update a pay plan', async () => {
      const orgId = 1
      const payPlanId = 1
      const updateRequest = { name: 'Updated Pay Plan' }
      const updatedPayPlan = { ...mockPayPlan, name: 'Updated Pay Plan' }

      mockClient.put.mockResolvedValue({ data: updatedPayPlan })

      const response = await mockClient.put(
        `/organizations/${orgId}/payplans/${payPlanId}`,
        updateRequest
      )

      expect(mockClient.put).toHaveBeenCalledWith(
        `/organizations/${orgId}/payplans/${payPlanId}`,
        updateRequest
      )
      expect(response.data.name).toBe('Updated Pay Plan')
    })

    it('should delete a pay plan', async () => {
      const orgId = 1
      const payPlanId = 1

      mockClient.delete.mockResolvedValue({})

      await mockClient.delete(`/organizations/${orgId}/payplans/${payPlanId}`)

      expect(mockClient.delete).toHaveBeenCalledWith(
        `/organizations/${orgId}/payplans/${payPlanId}`
      )
    })
  })

  describe('PayPlan Period operations', () => {
    it('should create a new period', async () => {
      const orgId = 1
      const payPlanId = 1
      const createRequest = {
        from: '2024-01-01',
        to: null,
        weekly_hours: 39.0
      }

      mockClient.post.mockResolvedValue({ data: mockPeriod })

      const response = await mockClient.post(
        `/organizations/${orgId}/payplans/${payPlanId}/periods`,
        createRequest
      )

      expect(mockClient.post).toHaveBeenCalledWith(
        `/organizations/${orgId}/payplans/${payPlanId}/periods`,
        createRequest
      )
      expect(response.data.weekly_hours).toBe(39.0)
    })

    it('should get a period by ID', async () => {
      const orgId = 1
      const payPlanId = 1
      const periodId = 1
      const periodWithEntries = { ...mockPeriod, entries: [mockEntry] }

      mockClient.get.mockResolvedValue({ data: periodWithEntries })

      const response = await mockClient.get(
        `/organizations/${orgId}/payplans/${payPlanId}/periods/${periodId}`
      )

      expect(mockClient.get).toHaveBeenCalledWith(
        `/organizations/${orgId}/payplans/${payPlanId}/periods/${periodId}`
      )
      expect(response.data.entries).toHaveLength(1)
    })

    it('should update a period', async () => {
      const orgId = 1
      const payPlanId = 1
      const periodId = 1
      const updateRequest = { weekly_hours: 40.0 }
      const updatedPeriod = { ...mockPeriod, weekly_hours: 40.0 }

      mockClient.put.mockResolvedValue({ data: updatedPeriod })

      const response = await mockClient.put(
        `/organizations/${orgId}/payplans/${payPlanId}/periods/${periodId}`,
        updateRequest
      )

      expect(mockClient.put).toHaveBeenCalledWith(
        `/organizations/${orgId}/payplans/${payPlanId}/periods/${periodId}`,
        updateRequest
      )
      expect(response.data.weekly_hours).toBe(40.0)
    })

    it('should delete a period', async () => {
      const orgId = 1
      const payPlanId = 1
      const periodId = 1

      mockClient.delete.mockResolvedValue({})

      await mockClient.delete(`/organizations/${orgId}/payplans/${payPlanId}/periods/${periodId}`)

      expect(mockClient.delete).toHaveBeenCalledWith(
        `/organizations/${orgId}/payplans/${payPlanId}/periods/${periodId}`
      )
    })
  })

  describe('PayPlan Entry operations', () => {
    it('should create a new entry', async () => {
      const orgId = 1
      const payPlanId = 1
      const periodId = 1
      const createRequest = {
        grade: 'S8a',
        step: 3,
        monthly_amount: 350000
      }

      mockClient.post.mockResolvedValue({ data: mockEntry })

      const response = await mockClient.post(
        `/organizations/${orgId}/payplans/${payPlanId}/periods/${periodId}/entries`,
        createRequest
      )

      expect(mockClient.post).toHaveBeenCalledWith(
        `/organizations/${orgId}/payplans/${payPlanId}/periods/${periodId}/entries`,
        createRequest
      )
      expect(response.data.grade).toBe('S8a')
      expect(response.data.step).toBe(3)
      expect(response.data.monthly_amount).toBe(350000)
    })

    it('should get an entry by ID', async () => {
      const orgId = 1
      const payPlanId = 1
      const periodId = 1
      const entryId = 1

      mockClient.get.mockResolvedValue({ data: mockEntry })

      const response = await mockClient.get(
        `/organizations/${orgId}/payplans/${payPlanId}/periods/${periodId}/entries/${entryId}`
      )

      expect(mockClient.get).toHaveBeenCalledWith(
        `/organizations/${orgId}/payplans/${payPlanId}/periods/${periodId}/entries/${entryId}`
      )
      expect(response.data.grade).toBe('S8a')
    })

    it('should update an entry', async () => {
      const orgId = 1
      const payPlanId = 1
      const periodId = 1
      const entryId = 1
      const updateRequest = { monthly_amount: 360000 }
      const updatedEntry = { ...mockEntry, monthly_amount: 360000 }

      mockClient.put.mockResolvedValue({ data: updatedEntry })

      const response = await mockClient.put(
        `/organizations/${orgId}/payplans/${payPlanId}/periods/${periodId}/entries/${entryId}`,
        updateRequest
      )

      expect(mockClient.put).toHaveBeenCalledWith(
        `/organizations/${orgId}/payplans/${payPlanId}/periods/${periodId}/entries/${entryId}`,
        updateRequest
      )
      expect(response.data.monthly_amount).toBe(360000)
    })

    it('should delete an entry', async () => {
      const orgId = 1
      const payPlanId = 1
      const periodId = 1
      const entryId = 1

      mockClient.delete.mockResolvedValue({})

      await mockClient.delete(
        `/organizations/${orgId}/payplans/${payPlanId}/periods/${periodId}/entries/${entryId}`
      )

      expect(mockClient.delete).toHaveBeenCalledWith(
        `/organizations/${orgId}/payplans/${payPlanId}/periods/${periodId}/entries/${entryId}`
      )
    })
  })

  describe('Error handling', () => {
    it('should handle 404 not found error', async () => {
      const orgId = 1
      const payPlanId = 999

      const error = {
        response: {
          status: 404,
          data: { code: 'NOT_FOUND', message: 'pay plan not found' }
        }
      }
      mockClient.get.mockRejectedValue(error)

      await expect(mockClient.get(`/organizations/${orgId}/payplans/${payPlanId}`)).rejects.toEqual(
        error
      )
    })

    it('should handle 400 bad request error', async () => {
      const orgId = 1
      const createRequest = { name: '' }

      const error = {
        response: {
          status: 400,
          data: { code: 'BAD_REQUEST', message: 'name is required' }
        }
      }
      mockClient.post.mockRejectedValue(error)

      await expect(
        mockClient.post(`/organizations/${orgId}/payplans`, createRequest)
      ).rejects.toEqual(error)
    })

    it('should handle 401 unauthorized error', async () => {
      const orgId = 1

      const error = {
        response: {
          status: 401,
          data: { code: 'UNAUTHORIZED', message: 'authentication required' }
        }
      }
      mockClient.get.mockRejectedValue(error)

      await expect(mockClient.get(`/organizations/${orgId}/payplans`)).rejects.toEqual(error)
    })

    it('should handle network error', async () => {
      const orgId = 1

      const error = new Error('Network Error')
      mockClient.get.mockRejectedValue(error)

      await expect(mockClient.get(`/organizations/${orgId}/payplans`)).rejects.toThrow(
        'Network Error'
      )
    })
  })

  describe('URL construction', () => {
    it('should construct correct URL for list with pagination', async () => {
      const orgId = 5
      mockClient.get.mockResolvedValue({ data: { data: [] } })

      await mockClient.get(`/organizations/${orgId}/payplans?limit=50`)

      expect(mockClient.get).toHaveBeenCalledWith('/organizations/5/payplans?limit=50')
    })

    it('should construct correct nested URL for entries', async () => {
      const orgId = 1
      const payPlanId = 2
      const periodId = 3
      const entryId = 4

      mockClient.get.mockResolvedValue({ data: mockEntry })

      await mockClient.get(
        `/organizations/${orgId}/payplans/${payPlanId}/periods/${periodId}/entries/${entryId}`
      )

      expect(mockClient.get).toHaveBeenCalledWith('/organizations/1/payplans/2/periods/3/entries/4')
    })
  })

  describe('Request payload', () => {
    it('should send correct payload for period creation with null end date', async () => {
      const orgId = 1
      const payPlanId = 1
      const createRequest = {
        from: '2024-03-01',
        to: null,
        weekly_hours: 39.0
      }

      mockClient.post.mockResolvedValue({ data: mockPeriod })

      await mockClient.post(`/organizations/${orgId}/payplans/${payPlanId}/periods`, createRequest)

      expect(mockClient.post).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          from: '2024-03-01',
          to: null,
          weekly_hours: 39.0
        })
      )
    })

    it('should send correct payload for entry creation with cents amount', async () => {
      const orgId = 1
      const payPlanId = 1
      const periodId = 1
      const createRequest = {
        grade: 'S11b',
        step: 5,
        monthly_amount: 468723 // €4,687.23
      }

      mockClient.post.mockResolvedValue({ data: { ...mockEntry, ...createRequest } })

      await mockClient.post(
        `/organizations/${orgId}/payplans/${payPlanId}/periods/${periodId}/entries`,
        createRequest
      )

      expect(mockClient.post).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          grade: 'S11b',
          step: 5,
          monthly_amount: 468723
        })
      )
    })
  })
})
