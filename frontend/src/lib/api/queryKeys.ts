export const queryKeys = {
  organizations: {
    all: () => ['organizations'] as const,
    list: (page: number) => ['organizations', 'list', page] as const,
  },
  users: {
    all: () => ['users'] as const,
    list: (page: number) => ['users', 'list', page] as const,
    memberships: (userId: number) => ['users', 'memberships', userId] as const,
  },
  employees: {
    all: (orgId: number) => ['employees', orgId] as const,
    list: (orgId: number, ...filters: unknown[]) =>
      ['employees', orgId, 'list', ...filters] as const,
    allUnpaginated: (orgId: number) => ['employees', orgId, 'allUnpaginated'] as const,
    detail: (orgId: number, employeeId: number) =>
      ['employees', orgId, 'detail', employeeId] as const,
    contracts: (orgId: number, employeeId: number) =>
      ['employees', orgId, 'contracts', employeeId] as const,
  },
  children: {
    all: (orgId: number) => ['children', orgId] as const,
    list: (orgId: number, ...filters: unknown[]) =>
      ['children', orgId, 'list', ...filters] as const,
    allUnpaginated: (orgId: number) => ['children', orgId, 'allUnpaginated'] as const,
    detail: (orgId: number, childId: number) => ['children', orgId, 'detail', childId] as const,
    contracts: (orgId: number, childId: number) =>
      ['children', orgId, 'contracts', childId] as const,
    billingHistory: (orgId: number, childId: number) =>
      ['children', orgId, 'billingHistory', childId] as const,
    billingSummary: (orgId: number) => ['children', orgId, 'billingSummary'] as const,
    withoutVouchers: (orgId: number) => ['children', orgId, 'withoutVouchers'] as const,
    vouchers: (orgId: number, childId: number) => ['children', orgId, 'vouchers', childId] as const,
    funding: (orgId: number) => ['children', orgId, 'funding'] as const,
    upcoming: (orgId: number) => ['children', orgId, 'upcoming'] as const,
  },
  payPlans: {
    all: (orgId: number) => ['payPlans', orgId] as const,
    list: (orgId: number, page: number, search?: string) =>
      ['payPlans', orgId, 'list', page, search] as const,
    detail: (orgId: number, payPlanId: number) => ['payPlans', orgId, 'detail', payPlanId] as const,
    details: (orgId: number, payPlanIds: number[]) =>
      ['payPlans', orgId, 'details', payPlanIds] as const,
  },
  sections: {
    list: (orgId: number) => ['sections', orgId] as const,
  },
  governmentFundings: {
    all: () => ['governmentFundings'] as const,
    list: (page: number, search?: string) => ['governmentFundings', 'list', page, search] as const,
    detail: (fundingId: number) => ['governmentFundings', 'detail', fundingId] as const,
    lookup: () => ['governmentFundings', 'lookup'] as const,
    lookupDetail: (fundingId: number | undefined) =>
      ['governmentFundings', 'lookupDetail', fundingId] as const,
  },
  governmentFundingBillPeriods: {
    all: (orgId: number) => ['governmentFundingBillPeriods', orgId] as const,
    list: (orgId: number, page: number) =>
      ['governmentFundingBillPeriods', orgId, 'list', page] as const,
    detail: (orgId: number, id: number) =>
      ['governmentFundingBillPeriods', orgId, 'detail', id] as const,
    compare: (orgId: number, id: number) =>
      ['governmentFundingBillPeriods', orgId, 'compare', id] as const,
    compareLatest: (orgId: number) =>
      ['governmentFundingBillPeriods', orgId, 'compareLatest'] as const,
    compareRange: (orgId: number, from: string, to: string) =>
      ['governmentFundingBillPeriods', orgId, 'compareRange', from, to] as const,
    unmatchedChildren: (orgId: number) =>
      ['governmentFundingBillPeriods', orgId, 'unmatchedChildren'] as const,
  },
  budgetItems: {
    all: (orgId: number) => ['budgetItems', orgId] as const,
    list: (orgId: number, page: number, search?: string) =>
      ['budgetItems', orgId, 'list', page, search] as const,
    detail: (orgId: number, budgetItemId: number) =>
      ['budgetItems', orgId, 'detail', budgetItemId] as const,
  },
  statistics: {
    all: (orgId: number) => ['statistics', orgId] as const,
    ageDistribution: (orgId: number) => ['statistics', orgId, 'ageDistribution'] as const,
    contractProperties: (orgId: number) => ['statistics', orgId, 'contractProperties'] as const,
    staffingHours: (orgId: number, sectionId?: number, from?: string, to?: string) =>
      ['statistics', orgId, 'staffingHours', sectionId, from, to] as const,
    financials: (orgId: number, from?: string, to?: string) =>
      ['statistics', orgId, 'financials', from, to] as const,
    occupancy: (orgId: number, sectionId?: number, from?: string, to?: string) =>
      ['statistics', orgId, 'occupancy', sectionId, from, to] as const,
    employeeStaffingHours: (orgId: number, sectionId?: number, from?: string, to?: string) =>
      ['statistics', orgId, 'employeeStaffingHours', sectionId, from, to] as const,
    forecast: (orgId: number) => ['statistics', orgId, 'forecast'] as const,
  },
  attendance: {
    all: (orgId: number) => ['attendance', orgId] as const,
    byDate: (orgId: number, date: string) => ['attendance', orgId, 'byDate', date] as const,
    summary: (orgId: number, date: string) => ['attendance', orgId, 'summary', date] as const,
    byWeek: (orgId: number, weekStart: string) =>
      ['attendance', orgId, 'byWeek', weekStart] as const,
  },
  stepPromotions: (orgId: number) => ['stepPromotions', orgId] as const,
  factors: {
    all: () => ['factors', 'me'] as const,
  },
  auditLogs: {
    all: (orgId: number) => ['auditLogs', orgId] as const,
    list: (orgId: number, ...filters: unknown[]) =>
      ['auditLogs', orgId, 'list', ...filters] as const,
  },
} as const;
