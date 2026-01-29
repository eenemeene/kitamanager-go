import { useI18n } from 'vue-i18n'

/**
 * Base interface for any contract with a time period.
 * Both ChildContract and EmployeeContract extend this.
 */
export interface ContractWithPeriod {
  id: number
  from: string
  to?: string | null
}

export type ContractStatus = 'active' | 'upcoming' | 'ended'

/**
 * Composable for contract status logic.
 * Provides utilities for determining contract status and formatting.
 */
export function useContractStatus() {
  const { t } = useI18n()

  /**
   * Determines the status of a contract based on its dates.
   */
  function getContractStatus(contract: ContractWithPeriod): ContractStatus {
    const now = new Date()
    now.setHours(0, 0, 0, 0) // Normalize to start of day

    const from = new Date(contract.from)
    from.setHours(0, 0, 0, 0)

    const to = contract.to ? new Date(contract.to) : null
    if (to) {
      to.setHours(0, 0, 0, 0)
    }

    // Future contract (hasn't started yet)
    if (from > now) {
      return 'upcoming'
    }

    // Active contract (started and hasn't ended)
    if (from <= now && (!to || to >= now)) {
      return 'active'
    }

    // Ended contract
    return 'ended'
  }

  /**
   * Gets the localized label for a contract status.
   */
  function getStatusLabel(status: ContractStatus): string {
    switch (status) {
      case 'active':
        return t('common.active')
      case 'upcoming':
        return t('common.upcoming')
      case 'ended':
        return t('common.ended')
    }
  }

  /**
   * Gets the PrimeVue Tag severity for a contract status.
   */
  function getStatusSeverity(status: ContractStatus): 'success' | 'info' | 'secondary' {
    switch (status) {
      case 'active':
        return 'success'
      case 'upcoming':
        return 'info'
      case 'ended':
        return 'secondary'
    }
  }

  /**
   * Determines if a contract is currently active.
   */
  function isActive(contract: ContractWithPeriod): boolean {
    return getContractStatus(contract) === 'active'
  }

  /**
   * Returns a row class for highlighting active contracts.
   */
  function getRowClass(contract: ContractWithPeriod): string | undefined {
    return isActive(contract) ? 'active-contract-row' : undefined
  }

  /**
   * Sorts contracts by start date (newest first).
   */
  function sortByDateDesc<T extends ContractWithPeriod>(contracts: T[]): T[] {
    return [...contracts].sort((a, b) => {
      return new Date(b.from).getTime() - new Date(a.from).getTime()
    })
  }

  return {
    getContractStatus,
    getStatusLabel,
    getStatusSeverity,
    isActive,
    getRowClass,
    sortByDateDesc
  }
}
