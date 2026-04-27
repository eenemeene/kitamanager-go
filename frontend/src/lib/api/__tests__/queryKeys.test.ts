import { queryKeys } from '../queryKeys';

describe('queryKeys', () => {
  describe('organizations', () => {
    it('all returns consistent key', () => {
      expect(queryKeys.organizations.all()).toEqual(['organizations']);
    });

    it('list includes page number', () => {
      expect(queryKeys.organizations.list(2)).toEqual(['organizations', 'list', 2]);
    });
  });

  describe('users', () => {
    it('all returns consistent key', () => {
      expect(queryKeys.users.all()).toEqual(['users']);
    });

    it('list includes page', () => {
      expect(queryKeys.users.list(3)).toEqual(['users', 'list', 3]);
    });

    it('memberships includes userId under users prefix', () => {
      expect(queryKeys.users.memberships(5)).toEqual(['users', 'memberships', 5]);
    });
  });

  describe('employees', () => {
    it('all includes orgId', () => {
      expect(queryKeys.employees.all(1)).toEqual(['employees', 1]);
    });

    it('list includes orgId and filters under list segment', () => {
      expect(queryKeys.employees.list(1, 'search', 'page')).toEqual([
        'employees',
        1,
        'list',
        'search',
        'page',
      ]);
    });

    it('allUnpaginated shares employees prefix', () => {
      expect(queryKeys.employees.allUnpaginated(1)).toEqual(['employees', 1, 'allUnpaginated']);
    });

    it('detail shares employees prefix', () => {
      expect(queryKeys.employees.detail(1, 42)).toEqual(['employees', 1, 'detail', 42]);
    });

    it('contracts shares employees prefix', () => {
      expect(queryKeys.employees.contracts(1, 42)).toEqual(['employees', 1, 'contracts', 42]);
    });
  });

  describe('children', () => {
    it('all includes orgId', () => {
      expect(queryKeys.children.all(1)).toEqual(['children', 1]);
    });

    it('detail shares children prefix', () => {
      expect(queryKeys.children.detail(1, 99)).toEqual(['children', 1, 'detail', 99]);
    });

    it('contracts shares children prefix', () => {
      expect(queryKeys.children.contracts(1, 99)).toEqual(['children', 1, 'contracts', 99]);
    });

    it('funding shares children prefix', () => {
      expect(queryKeys.children.funding(1)).toEqual(['children', 1, 'funding']);
    });

    it('billingSummary shares children prefix', () => {
      expect(queryKeys.children.billingSummary(1)).toEqual(['children', 1, 'billingSummary']);
    });

    it('withoutVouchers shares children prefix', () => {
      expect(queryKeys.children.withoutVouchers(1)).toEqual(['children', 1, 'withoutVouchers']);
    });
  });

  describe('sections', () => {
    it('list includes orgId', () => {
      expect(queryKeys.sections.list(3)).toEqual(['sections', 3]);
    });
  });

  describe('governmentFundings', () => {
    it('all returns base key', () => {
      expect(queryKeys.governmentFundings.all()).toEqual(['governmentFundings']);
    });

    it('list shares governmentFundings prefix', () => {
      expect(queryKeys.governmentFundings.list(2)).toEqual(['governmentFundings', 'list', 2]);
    });

    it('detail shares governmentFundings prefix', () => {
      expect(queryKeys.governmentFundings.detail(5)).toEqual(['governmentFundings', 'detail', 5]);
    });

    it('lookup shares governmentFundings prefix', () => {
      expect(queryKeys.governmentFundings.lookup()).toEqual(['governmentFundings', 'lookup']);
    });

    it('lookupDetail shares governmentFundings prefix', () => {
      expect(queryKeys.governmentFundings.lookupDetail(5)).toEqual([
        'governmentFundings',
        'lookupDetail',
        5,
      ]);
    });
  });

  describe('governmentFundingBillPeriods', () => {
    it('detail shares base prefix', () => {
      expect(queryKeys.governmentFundingBillPeriods.detail(1, 5)).toEqual([
        'governmentFundingBillPeriods',
        1,
        'detail',
        5,
      ]);
    });

    it('compare shares base prefix', () => {
      expect(queryKeys.governmentFundingBillPeriods.compare(1, 5)).toEqual([
        'governmentFundingBillPeriods',
        1,
        'compare',
        5,
      ]);
    });

    it('compareLatest shares base prefix', () => {
      expect(queryKeys.governmentFundingBillPeriods.compareLatest(1)).toEqual([
        'governmentFundingBillPeriods',
        1,
        'compareLatest',
      ]);
    });
  });

  describe('budgetItems', () => {
    it('detail shares budgetItems prefix', () => {
      expect(queryKeys.budgetItems.detail(1, 7)).toEqual(['budgetItems', 1, 'detail', 7]);
    });
  });

  describe('attendance', () => {
    it('byDate shares attendance prefix', () => {
      expect(queryKeys.attendance.byDate(1, '2025-06-15')).toEqual([
        'attendance',
        1,
        'byDate',
        '2025-06-15',
      ]);
    });

    it('summary shares attendance prefix', () => {
      expect(queryKeys.attendance.summary(1, '2025-06-15')).toEqual([
        'attendance',
        1,
        'summary',
        '2025-06-15',
      ]);
    });

    it('byWeek shares attendance prefix', () => {
      expect(queryKeys.attendance.byWeek(1, '2025-06-09')).toEqual([
        'attendance',
        1,
        'byWeek',
        '2025-06-09',
      ]);
    });
  });

  describe('statistics', () => {
    it('all includes orgId', () => {
      expect(queryKeys.statistics.all(1)).toEqual(['statistics', 1]);
    });

    it('staffingHours shares statistics prefix', () => {
      expect(queryKeys.statistics.staffingHours(1, 2, '2025-01-01', '2025-12-31')).toEqual([
        'statistics',
        1,
        'staffingHours',
        2,
        '2025-01-01',
        '2025-12-31',
      ]);
    });

    it('staffingHours works without optional params', () => {
      expect(queryKeys.statistics.staffingHours(1)).toEqual([
        'statistics',
        1,
        'staffingHours',
        undefined,
        undefined,
        undefined,
      ]);
    });

    it('financials shares statistics prefix', () => {
      expect(queryKeys.statistics.financials(1)).toEqual([
        'statistics',
        1,
        'financials',
        undefined,
        undefined,
      ]);
    });

    // Regression test for the stale-financials-after-budget-item-edit
    // bug. TanStack Query invalidates by prefix. The right way to
    // invalidate every concrete financials query (each parameterised
    // by from/to) is via `queryKeys.statistics.all(orgId)` — the bare
    // ['statistics', orgId] tuple. The previous frontend code used
    // `['financials', orgId]`, which matches nothing because
    // financials keys actually live under the 'statistics' namespace.
    //
    // This test runs a real QueryClient with two parameterised
    // financials queries and asserts both get invalidated by the
    // statistics prefix. If somebody re-introduces the per-financials
    // key (or worse, the bare ['financials', orgId] tuple), this
    // test fails loud.
    it('statistics.all prefix invalidates every parameterised financials query', async () => {
      const { QueryClient } = await import('@tanstack/react-query');
      const client = new QueryClient();
      // Seed two concrete financials queries under different date
      // ranges, plus a budgetItems query that must NOT be invalidated.
      client.setQueryData(queryKeys.statistics.financials(1, '2025-01-01', '2025-12-01'), {
        seeded: 'a',
      });
      client.setQueryData(queryKeys.statistics.financials(1, '2026-01-01', '2026-12-01'), {
        seeded: 'b',
      });
      client.setQueryData(queryKeys.budgetItems.all(1), { seeded: 'budgets' });

      // Invalidate via the statistics-namespace prefix.
      await client.invalidateQueries({ queryKey: queryKeys.statistics.all(1) });

      // Both financials queries are marked stale.
      const a = client.getQueryState(
        queryKeys.statistics.financials(1, '2025-01-01', '2025-12-01')
      );
      const b = client.getQueryState(
        queryKeys.statistics.financials(1, '2026-01-01', '2026-12-01')
      );
      expect(a?.isInvalidated).toBe(true);
      expect(b?.isInvalidated).toBe(true);

      // budgetItems is NOT invalidated — the prefix only matches
      // queries under the 'statistics' branch.
      const budgets = client.getQueryState(queryKeys.budgetItems.all(1));
      expect(budgets?.isInvalidated).toBe(false);
    });

    // Negative regression: the buggy historical key did not match.
    // Lock this in so a "let's restore the simpler key" PR is
    // immediately exposed.
    it('the legacy ["financials", orgId] tuple does NOT match parameterised queries', async () => {
      const { QueryClient } = await import('@tanstack/react-query');
      const client = new QueryClient();
      client.setQueryData(queryKeys.statistics.financials(1, '2025-01-01', '2025-12-01'), {
        seeded: 'a',
      });

      await client.invalidateQueries({ queryKey: ['financials', 1] });

      const a = client.getQueryState(
        queryKeys.statistics.financials(1, '2025-01-01', '2025-12-01')
      );
      // BUG REPRODUCTION: nothing matches.
      expect(a?.isInvalidated).toBe(false);
    });
  });

  describe('stepPromotions', () => {
    it('includes orgId', () => {
      expect(queryKeys.stepPromotions(1)).toEqual(['stepPromotions', 1]);
    });
  });

  describe('key uniqueness', () => {
    it('different resources produce different keys', () => {
      const empAll = queryKeys.employees.all(1);
      const childAll = queryKeys.children.all(1);
      expect(empAll).not.toEqual(childAll);
    });

    it('different orgIds produce different keys', () => {
      expect(queryKeys.employees.all(1)).not.toEqual(queryKeys.employees.all(2));
    });
  });

  describe('hierarchical prefix invalidation', () => {
    /** Helper: returns true when TanStack Query's fuzzy matching would treat
     *  `prefix` as a prefix of `key` (i.e., every element of prefix equals
     *  the corresponding element of key). */
    function isPrefix(prefix: readonly unknown[], key: readonly unknown[]): boolean {
      if (prefix.length > key.length) return false;
      return prefix.every((v, i) => v === key[i]);
    }

    it('employees.all(orgId) is a prefix of all employee sub-keys', () => {
      const prefix = queryKeys.employees.all(1);
      expect(isPrefix(prefix, queryKeys.employees.list(1, 'a'))).toBe(true);
      expect(isPrefix(prefix, queryKeys.employees.allUnpaginated(1))).toBe(true);
      expect(isPrefix(prefix, queryKeys.employees.detail(1, 42))).toBe(true);
      expect(isPrefix(prefix, queryKeys.employees.contracts(1, 42))).toBe(true);
    });

    it('employees.all(orgId) does NOT match a different orgId', () => {
      const prefix = queryKeys.employees.all(1);
      expect(isPrefix(prefix, queryKeys.employees.detail(2, 42))).toBe(false);
    });

    it('children.all(orgId) is a prefix of all children sub-keys', () => {
      const prefix = queryKeys.children.all(1);
      expect(isPrefix(prefix, queryKeys.children.list(1, 'b'))).toBe(true);
      expect(isPrefix(prefix, queryKeys.children.allUnpaginated(1))).toBe(true);
      expect(isPrefix(prefix, queryKeys.children.detail(1, 99))).toBe(true);
      expect(isPrefix(prefix, queryKeys.children.contracts(1, 99))).toBe(true);
      expect(isPrefix(prefix, queryKeys.children.billingHistory(1, 99))).toBe(true);
      expect(isPrefix(prefix, queryKeys.children.billingSummary(1))).toBe(true);
      expect(isPrefix(prefix, queryKeys.children.withoutVouchers(1))).toBe(true);
      expect(isPrefix(prefix, queryKeys.children.funding(1))).toBe(true);
      expect(isPrefix(prefix, queryKeys.children.upcoming(1))).toBe(true);
    });

    it('users.all() is a prefix of all user sub-keys', () => {
      const prefix = queryKeys.users.all();
      expect(isPrefix(prefix, queryKeys.users.list(1))).toBe(true);
      expect(isPrefix(prefix, queryKeys.users.memberships(5))).toBe(true);
    });

    it('organizations.all() is a prefix of organizations.list()', () => {
      const prefix = queryKeys.organizations.all();
      expect(isPrefix(prefix, queryKeys.organizations.list(1))).toBe(true);
    });

    it('attendance.all(orgId) is a prefix of all attendance sub-keys', () => {
      const prefix = queryKeys.attendance.all(1);
      expect(isPrefix(prefix, queryKeys.attendance.byDate(1, '2025-06-15'))).toBe(true);
      expect(isPrefix(prefix, queryKeys.attendance.summary(1, '2025-06-15'))).toBe(true);
      expect(isPrefix(prefix, queryKeys.attendance.byWeek(1, '2025-06-09'))).toBe(true);
    });

    it('statistics.all(orgId) is a prefix of all statistics sub-keys', () => {
      const prefix = queryKeys.statistics.all(1);
      expect(isPrefix(prefix, queryKeys.statistics.ageDistribution(1))).toBe(true);
      expect(isPrefix(prefix, queryKeys.statistics.contractProperties(1))).toBe(true);
      expect(isPrefix(prefix, queryKeys.statistics.staffingHours(1))).toBe(true);
      expect(isPrefix(prefix, queryKeys.statistics.financials(1))).toBe(true);
      expect(isPrefix(prefix, queryKeys.statistics.occupancy(1))).toBe(true);
      expect(isPrefix(prefix, queryKeys.statistics.employeeStaffingHours(1))).toBe(true);
      expect(isPrefix(prefix, queryKeys.statistics.forecast(1))).toBe(true);
    });

    it('governmentFundings.all() is a prefix of all governmentFundings sub-keys', () => {
      const prefix = queryKeys.governmentFundings.all();
      expect(isPrefix(prefix, queryKeys.governmentFundings.list(1))).toBe(true);
      expect(isPrefix(prefix, queryKeys.governmentFundings.detail(5))).toBe(true);
      expect(isPrefix(prefix, queryKeys.governmentFundings.lookup())).toBe(true);
      expect(isPrefix(prefix, queryKeys.governmentFundings.lookupDetail(5))).toBe(true);
    });

    it('governmentFundingBillPeriods.all(orgId) is a prefix of all sub-keys', () => {
      const prefix = queryKeys.governmentFundingBillPeriods.all(1);
      expect(isPrefix(prefix, queryKeys.governmentFundingBillPeriods.list(1, 2))).toBe(true);
      expect(isPrefix(prefix, queryKeys.governmentFundingBillPeriods.detail(1, 5))).toBe(true);
      expect(isPrefix(prefix, queryKeys.governmentFundingBillPeriods.compare(1, 5))).toBe(true);
      expect(isPrefix(prefix, queryKeys.governmentFundingBillPeriods.compareLatest(1))).toBe(true);
      expect(
        isPrefix(
          prefix,
          queryKeys.governmentFundingBillPeriods.compareRange(1, '2025-01', '2025-06')
        )
      ).toBe(true);
    });

    it('budgetItems.all(orgId) is a prefix of all budgetItems sub-keys', () => {
      const prefix = queryKeys.budgetItems.all(1);
      expect(isPrefix(prefix, queryKeys.budgetItems.list(1, 2))).toBe(true);
      expect(isPrefix(prefix, queryKeys.budgetItems.detail(1, 7))).toBe(true);
    });

    it('payPlans.all(orgId) is a prefix of all payPlans sub-keys', () => {
      const prefix = queryKeys.payPlans.all(1);
      expect(isPrefix(prefix, queryKeys.payPlans.list(1, 2))).toBe(true);
      expect(isPrefix(prefix, queryKeys.payPlans.detail(1, 5))).toBe(true);
      expect(isPrefix(prefix, queryKeys.payPlans.details(1, [5, 6]))).toBe(true);
    });
  });
});
