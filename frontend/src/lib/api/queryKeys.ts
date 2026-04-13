export const queryKeys = {
  organizations: {
    all: () => ['organizations'] as const,
    list: (page: number) => ['organizations', page] as const,
  },
  users: {
    all: () => ['users'] as const,
    list: (page: number) => ['users', page] as const,
    memberships: (userId: number) => ['userMemberships', userId] as const,
  },
  employees: {
    all: (orgId: number) => ['employees', orgId] as const,
    list: (orgId: number, ...filters: unknown[]) => ['employees', orgId, ...filters] as const,
    allUnpaginated: (orgId: number) => ['employeesAll', orgId] as const,
    detail: (orgId: number, employeeId: number) => ['employee', orgId, employeeId] as const,
    contracts: (orgId: number, employeeId: number) =>
      ['employeeContracts', orgId, employeeId] as const,
  },
  children: {
    all: (orgId: number) => ['children', orgId] as const,
    list: (orgId: number, ...filters: unknown[]) => ['children', orgId, ...filters] as const,
    allUnpaginated: (orgId: number) => ['childrenAll', orgId] as const,
    detail: (orgId: number, childId: number) => ['child', orgId, childId] as const,
    contracts: (orgId: number, childId: number) => ['childContracts', orgId, childId] as const,
    billingHistory: (orgId: number, childId: number) =>
      ['childBillingHistory', orgId, childId] as const,
    funding: (orgId: number) => ['childrenFunding', orgId] as const,
    upcoming: (orgId: number) => ['childrenUpcoming', orgId] as const,
  },
  payPlans: {
    all: (orgId: number) => ['payplans', orgId] as const,
    list: (orgId: number, page: number) => ['payplans', orgId, page] as const,
    detail: (orgId: number, payPlanId: number) => ['payplan', orgId, payPlanId] as const,
    details: (orgId: number, payPlanIds: number[]) =>
      ['payplanDetails', orgId, payPlanIds] as const,
  },
  sections: {
    list: (orgId: number) => ['sections', orgId] as const,
  },
  governmentFundings: {
    all: () => ['governmentFundings'] as const,
    list: (page: number) => ['governmentFundings', page] as const,
    detail: (fundingId: number) => ['governmentFunding', fundingId] as const,
    allCached: () => ['governmentFundings', 'all'] as const,
    detailCached: (fundingId: number | undefined) =>
      ['governmentFunding', fundingId, 'details'] as const,
  },
  governmentFundingBillPeriods: {
    all: (orgId: number) => ['governmentFundingBillPeriods', orgId] as const,
    list: (orgId: number, page: number) => ['governmentFundingBillPeriods', orgId, page] as const,
    detail: (orgId: number, id: number) => ['governmentFundingBillPeriod', orgId, id] as const,
    compare: (orgId: number, id: number) =>
      ['governmentFundingBillPeriodCompare', orgId, id] as const,
  },
  budgetItems: {
    all: (orgId: number) => ['budgetItems', orgId] as const,
    list: (orgId: number, page: number) => ['budgetItems', orgId, page] as const,
    detail: (orgId: number, budgetItemId: number) => ['budgetItem', orgId, budgetItemId] as const,
  },
  statistics: {
    ageDistribution: (orgId: number) => ['ageDistribution', orgId] as const,
    contractProperties: (orgId: number) => ['contractProperties', orgId] as const,
    staffingHours: (orgId: number, sectionId?: number, from?: string, to?: string) =>
      ['staffingHours', orgId, sectionId, from, to] as const,
    financials: (orgId: number, from?: string, to?: string) =>
      ['financials', orgId, from, to] as const,
    occupancy: (orgId: number, sectionId?: number, from?: string, to?: string) =>
      ['occupancy', orgId, sectionId, from, to] as const,
    employeeStaffingHours: (orgId: number, sectionId?: number, from?: string, to?: string) =>
      ['employeeStaffingHours', orgId, sectionId, from, to] as const,
    forecast: (orgId: number) => ['forecast', orgId] as const,
  },
  attendance: {
    byDate: (orgId: number, date: string) => ['attendance', orgId, date] as const,
    summary: (orgId: number, date: string) => ['attendanceSummary', orgId, date] as const,
    all: (orgId: number) => ['attendance', orgId] as const,
    byWeek: (orgId: number, weekStart: string) => ['attendanceWeek', orgId, weekStart] as const,
  },
  stepPromotions: (orgId: number) => ['stepPromotions', orgId] as const,
} as const;
