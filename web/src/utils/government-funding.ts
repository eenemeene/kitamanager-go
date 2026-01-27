import type { GovernmentFunding } from '@/api/types'

/**
 * Represents a flattened row in the government funding table view.
 * Each row combines data from period, entry, and property levels.
 */
export interface FlattenedGovernmentFundingRow {
  periodId: number
  periodFrom: string
  periodTo: string | null
  periodComment: string
  entryId: number
  ageRange: string
  minAge: number
  maxAge: number
  propertyId: number
  propertyName: string
  payment: number
  requirement: number
  propertyComment: string
  // For visual grouping in table
  isFirstEntryInPeriod: boolean
  isFirstPropertyInEntry: boolean
  periodRowSpan: number
  entryRowSpan: number
}

/**
 * Flattens the hierarchical government funding structure into rows for table display.
 *
 * The government funding hierarchy is: GovernmentFunding -> Periods -> Entries -> Properties
 * This function flattens it into one row per property, with flags indicating
 * group boundaries for visual display.
 *
 * @param governmentFunding - The government funding object with nested periods, entries, and properties
 * @returns Array of flattened rows suitable for table display
 */
export function flattenGovernmentFundingToRows(
  governmentFunding: GovernmentFunding | null
): FlattenedGovernmentFundingRow[] {
  if (!governmentFunding?.periods) return []

  const rows: FlattenedGovernmentFundingRow[] = []

  for (const period of governmentFunding.periods) {
    const entries = period.entries || []
    let periodRowCount = 0

    // Calculate total rows for this period
    for (const entry of entries) {
      const propCount = entry.properties?.length || 1
      periodRowCount += propCount
    }
    if (entries.length === 0) periodRowCount = 1

    let isFirstEntryInPeriod = true

    for (const entry of entries) {
      const properties = entry.properties || []
      const entryRowCount = properties.length || 1
      let isFirstPropertyInEntry = true

      if (properties.length === 0) {
        // Entry with no properties
        rows.push({
          periodId: period.id,
          periodFrom: period.from,
          periodTo: period.to || null,
          periodComment: period.comment || '',
          entryId: entry.id,
          ageRange: `${entry.min_age} - ${entry.max_age}`,
          minAge: entry.min_age,
          maxAge: entry.max_age,
          propertyId: 0,
          propertyName: '-',
          payment: 0,
          requirement: 0,
          propertyComment: '',
          isFirstEntryInPeriod,
          isFirstPropertyInEntry: true,
          periodRowSpan: isFirstEntryInPeriod ? periodRowCount : 0,
          entryRowSpan: entryRowCount
        })
        isFirstEntryInPeriod = false
      } else {
        for (const prop of properties) {
          rows.push({
            periodId: period.id,
            periodFrom: period.from,
            periodTo: period.to || null,
            periodComment: period.comment || '',
            entryId: entry.id,
            ageRange: `${entry.min_age} - ${entry.max_age}`,
            minAge: entry.min_age,
            maxAge: entry.max_age,
            propertyId: prop.id,
            propertyName: prop.name,
            payment: prop.payment,
            requirement: prop.requirement,
            propertyComment: prop.comment || '',
            isFirstEntryInPeriod,
            isFirstPropertyInEntry,
            periodRowSpan: isFirstEntryInPeriod ? periodRowCount : 0,
            entryRowSpan: isFirstPropertyInEntry ? entryRowCount : 0
          })
          isFirstEntryInPeriod = false
          isFirstPropertyInEntry = false
        }
      }
    }

    // Period with no entries
    if (entries.length === 0) {
      rows.push({
        periodId: period.id,
        periodFrom: period.from,
        periodTo: period.to || null,
        periodComment: period.comment || '',
        entryId: 0,
        ageRange: '-',
        minAge: 0,
        maxAge: 0,
        propertyId: 0,
        propertyName: '-',
        payment: 0,
        requirement: 0,
        propertyComment: '',
        isFirstEntryInPeriod: true,
        isFirstPropertyInEntry: true,
        periodRowSpan: 1,
        entryRowSpan: 1
      })
    }
  }

  return rows
}
