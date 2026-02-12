export const queryKeys = {
  organizations: {
    all: () => ['organizations'] as const,
    list: (page: number) => ['organizations', page] as const,
  },
  users: {
    all: () => ['users'] as const,
    list: (page: number) => ['users', page] as const,
  },
  employees: {
    all: (orgId: number) => ['employees', orgId] as const,
    list: (orgId: number, ...filters: unknown[]) => ['employees', orgId, ...filters] as const,
    allUnpaginated: (orgId: number) => ['employees-all', orgId] as const,
    detail: (orgId: number, employeeId: number) => ['employee', orgId, employeeId] as const,
    contracts: (orgId: number, employeeId: number) =>
      ['employeeContracts', orgId, employeeId] as const,
  },
  children: {
    all: (orgId: number) => ['children', orgId] as const,
    list: (orgId: number, ...filters: unknown[]) => ['children', orgId, ...filters] as const,
    allUnpaginated: (orgId: number) => ['children-all', orgId] as const,
    detail: (orgId: number, childId: number) => ['child', orgId, childId] as const,
    contracts: (orgId: number, childId: number) => ['childContracts', orgId, childId] as const,
    funding: (orgId: number) => ['childrenFunding', orgId] as const,
    upcoming: (orgId: number) => ['children-upcoming', orgId] as const,
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
    all: () => ['government-fundings'] as const,
    list: (page: number) => ['government-fundings', page] as const,
    detail: (fundingId: number) => ['government-funding', fundingId] as const,
    allCached: () => ['governmentFundings', 'all'] as const,
    detailCached: (fundingId: number | undefined) =>
      ['governmentFunding', fundingId, 'details'] as const,
  },
  statistics: {
    ageDistribution: (orgId: number) => ['age-distribution', orgId] as const,
    contractCounts: (orgId: number) => ['contract-counts', orgId] as const,
  },
  stepPromotions: (orgId: number) => ['step-promotions', orgId] as const,
} as const;
