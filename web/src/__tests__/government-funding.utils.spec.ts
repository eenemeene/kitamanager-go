import { describe, it, expect } from 'vitest'
import { flattenGovernmentFundingToRows } from '../utils/government-funding'
import type { GovernmentFunding } from '../api/types'

describe('flattenGovernmentFundingToRows', () => {
  it('should return empty array for null government funding', () => {
    const result = flattenGovernmentFundingToRows(null)
    expect(result).toEqual([])
  })

  it('should return empty array for government funding without periods', () => {
    const governmentFunding: GovernmentFunding = {
      id: 1,
      name: 'Test',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z'
    }
    const result = flattenGovernmentFundingToRows(governmentFunding)
    expect(result).toEqual([])
  })

  it('should handle period with no entries', () => {
    const governmentFunding: GovernmentFunding = {
      id: 1,
      name: 'Test',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      periods: [
        {
          id: 1,
          government_funding_id: 1,
          from: '2024-01-01',
          to: '2024-12-31',
          comment: 'Test period',
          created_at: '2024-01-01T00:00:00Z',
          entries: []
        }
      ]
    }

    const result = flattenGovernmentFundingToRows(governmentFunding)

    expect(result).toHaveLength(1)
    expect(result[0].periodId).toBe(1)
    expect(result[0].periodFrom).toBe('2024-01-01')
    expect(result[0].periodTo).toBe('2024-12-31')
    expect(result[0].entryId).toBe(0)
    expect(result[0].ageRange).toBe('-')
    expect(result[0].propertyName).toBe('-')
    expect(result[0].isFirstEntryInPeriod).toBe(true)
  })

  it('should handle entry with no properties', () => {
    const governmentFunding: GovernmentFunding = {
      id: 1,
      name: 'Test',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      periods: [
        {
          id: 1,
          government_funding_id: 1,
          from: '2024-01-01',
          to: null,
          comment: '',
          created_at: '2024-01-01T00:00:00Z',
          entries: [
            {
              id: 1,
              period_id: 1,
              min_age: 0,
              max_age: 2,
              created_at: '2024-01-01T00:00:00Z',
              properties: []
            }
          ]
        }
      ]
    }

    const result = flattenGovernmentFundingToRows(governmentFunding)

    expect(result).toHaveLength(1)
    expect(result[0].entryId).toBe(1)
    expect(result[0].ageRange).toBe('0 - 2')
    expect(result[0].minAge).toBe(0)
    expect(result[0].maxAge).toBe(2)
    expect(result[0].propertyId).toBe(0)
    expect(result[0].propertyName).toBe('-')
    expect(result[0].periodTo).toBeNull()
  })

  it('should flatten full hierarchy correctly', () => {
    const governmentFunding: GovernmentFunding = {
      id: 1,
      name: 'Berlin',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      periods: [
        {
          id: 1,
          government_funding_id: 1,
          from: '2024-01-01',
          to: '2024-12-31',
          comment: 'Period 2024',
          created_at: '2024-01-01T00:00:00Z',
          entries: [
            {
              id: 1,
              period_id: 1,
              min_age: 0,
              max_age: 2,
              created_at: '2024-01-01T00:00:00Z',
              properties: [
                {
                  id: 1,
                  entry_id: 1,
                  name: 'ganztag',
                  payment: 166847,
                  requirement: 0.261,
                  comment: 'Full-day',
                  created_at: '2024-01-01T00:00:00Z'
                },
                {
                  id: 2,
                  entry_id: 1,
                  name: 'halbtag',
                  payment: 120000,
                  requirement: 0.18,
                  comment: 'Half-day',
                  created_at: '2024-01-01T00:00:00Z'
                }
              ]
            },
            {
              id: 2,
              period_id: 1,
              min_age: 2,
              max_age: 7,
              created_at: '2024-01-01T00:00:00Z',
              properties: [
                {
                  id: 3,
                  entry_id: 2,
                  name: 'ganztag',
                  payment: 150000,
                  requirement: 0.2,
                  comment: '',
                  created_at: '2024-01-01T00:00:00Z'
                }
              ]
            }
          ]
        }
      ]
    }

    const result = flattenGovernmentFundingToRows(governmentFunding)

    // Should have 3 rows: 2 properties for first entry + 1 property for second entry
    expect(result).toHaveLength(3)

    // First row: first property of first entry
    expect(result[0].periodId).toBe(1)
    expect(result[0].periodFrom).toBe('2024-01-01')
    expect(result[0].periodComment).toBe('Period 2024')
    expect(result[0].entryId).toBe(1)
    expect(result[0].ageRange).toBe('0 - 2')
    expect(result[0].propertyName).toBe('ganztag')
    expect(result[0].payment).toBe(166847)
    expect(result[0].requirement).toBe(0.261)
    expect(result[0].isFirstEntryInPeriod).toBe(true)
    expect(result[0].isFirstPropertyInEntry).toBe(true)
    expect(result[0].periodRowSpan).toBe(3) // Total 3 rows in this period
    expect(result[0].entryRowSpan).toBe(2) // 2 properties in this entry

    // Second row: second property of first entry
    expect(result[1].entryId).toBe(1)
    expect(result[1].propertyName).toBe('halbtag')
    expect(result[1].payment).toBe(120000)
    expect(result[1].isFirstEntryInPeriod).toBe(false)
    expect(result[1].isFirstPropertyInEntry).toBe(false)
    expect(result[1].periodRowSpan).toBe(0) // Not first row of period
    expect(result[1].entryRowSpan).toBe(0) // Not first row of entry

    // Third row: property of second entry
    expect(result[2].entryId).toBe(2)
    expect(result[2].ageRange).toBe('2 - 7')
    expect(result[2].propertyName).toBe('ganztag')
    expect(result[2].payment).toBe(150000)
    expect(result[2].isFirstEntryInPeriod).toBe(false)
    expect(result[2].isFirstPropertyInEntry).toBe(true)
    expect(result[2].periodRowSpan).toBe(0) // Not first row of period
    expect(result[2].entryRowSpan).toBe(1) // 1 property in this entry
  })

  it('should handle multiple periods', () => {
    const governmentFunding: GovernmentFunding = {
      id: 1,
      name: 'Test',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      periods: [
        {
          id: 1,
          government_funding_id: 1,
          from: '2024-01-01',
          to: '2024-06-30',
          comment: '',
          created_at: '2024-01-01T00:00:00Z',
          entries: [
            {
              id: 1,
              period_id: 1,
              min_age: 0,
              max_age: 3,
              created_at: '2024-01-01T00:00:00Z',
              properties: [
                {
                  id: 1,
                  entry_id: 1,
                  name: 'ganztag',
                  payment: 100000,
                  requirement: 0.25,
                  comment: '',
                  created_at: '2024-01-01T00:00:00Z'
                }
              ]
            }
          ]
        },
        {
          id: 2,
          government_funding_id: 1,
          from: '2024-07-01',
          to: null,
          comment: 'New rates',
          created_at: '2024-07-01T00:00:00Z',
          entries: [
            {
              id: 2,
              period_id: 2,
              min_age: 0,
              max_age: 3,
              created_at: '2024-07-01T00:00:00Z',
              properties: [
                {
                  id: 2,
                  entry_id: 2,
                  name: 'ganztag',
                  payment: 110000,
                  requirement: 0.27,
                  comment: 'Updated rate',
                  created_at: '2024-07-01T00:00:00Z'
                }
              ]
            }
          ]
        }
      ]
    }

    const result = flattenGovernmentFundingToRows(governmentFunding)

    expect(result).toHaveLength(2)

    // First period
    expect(result[0].periodId).toBe(1)
    expect(result[0].periodFrom).toBe('2024-01-01')
    expect(result[0].periodTo).toBe('2024-06-30')
    expect(result[0].isFirstEntryInPeriod).toBe(true)
    expect(result[0].periodRowSpan).toBe(1)

    // Second period
    expect(result[1].periodId).toBe(2)
    expect(result[1].periodFrom).toBe('2024-07-01')
    expect(result[1].periodTo).toBeNull() // Ongoing period
    expect(result[1].periodComment).toBe('New rates')
    expect(result[1].isFirstEntryInPeriod).toBe(true)
    expect(result[1].periodRowSpan).toBe(1)
    expect(result[1].payment).toBe(110000)
  })

  it('should correctly calculate row spans for complex structure', () => {
    // Period with 2 entries: first has 3 properties, second has 1 property
    const governmentFunding: GovernmentFunding = {
      id: 1,
      name: 'Test',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      periods: [
        {
          id: 1,
          government_funding_id: 1,
          from: '2024-01-01',
          to: null,
          comment: '',
          created_at: '2024-01-01T00:00:00Z',
          entries: [
            {
              id: 1,
              period_id: 1,
              min_age: 0,
              max_age: 2,
              created_at: '2024-01-01T00:00:00Z',
              properties: [
                {
                  id: 1,
                  entry_id: 1,
                  name: 'a',
                  payment: 100,
                  requirement: 0.1,
                  comment: '',
                  created_at: ''
                },
                {
                  id: 2,
                  entry_id: 1,
                  name: 'b',
                  payment: 200,
                  requirement: 0.2,
                  comment: '',
                  created_at: ''
                },
                {
                  id: 3,
                  entry_id: 1,
                  name: 'c',
                  payment: 300,
                  requirement: 0.3,
                  comment: '',
                  created_at: ''
                }
              ]
            },
            {
              id: 2,
              period_id: 1,
              min_age: 2,
              max_age: 7,
              created_at: '2024-01-01T00:00:00Z',
              properties: [
                {
                  id: 4,
                  entry_id: 2,
                  name: 'd',
                  payment: 400,
                  requirement: 0.4,
                  comment: '',
                  created_at: ''
                }
              ]
            }
          ]
        }
      ]
    }

    const result = flattenGovernmentFundingToRows(governmentFunding)

    expect(result).toHaveLength(4)

    // First row should have periodRowSpan = 4 (total rows) and entryRowSpan = 3
    expect(result[0].isFirstEntryInPeriod).toBe(true)
    expect(result[0].isFirstPropertyInEntry).toBe(true)
    expect(result[0].periodRowSpan).toBe(4)
    expect(result[0].entryRowSpan).toBe(3)

    // Second and third rows should have periodRowSpan = 0 and entryRowSpan = 0
    expect(result[1].periodRowSpan).toBe(0)
    expect(result[1].entryRowSpan).toBe(0)
    expect(result[2].periodRowSpan).toBe(0)
    expect(result[2].entryRowSpan).toBe(0)

    // Fourth row is first property of second entry
    expect(result[3].isFirstEntryInPeriod).toBe(false)
    expect(result[3].isFirstPropertyInEntry).toBe(true)
    expect(result[3].periodRowSpan).toBe(0)
    expect(result[3].entryRowSpan).toBe(1)
  })
})
