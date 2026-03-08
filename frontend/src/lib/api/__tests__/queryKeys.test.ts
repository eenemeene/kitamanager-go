import { queryKeys } from '../queryKeys';

describe('queryKeys', () => {
  describe('organizations', () => {
    it('all returns consistent key', () => {
      expect(queryKeys.organizations.all()).toEqual(['organizations']);
    });

    it('list includes page number', () => {
      expect(queryKeys.organizations.list(2)).toEqual(['organizations', 2]);
    });
  });

  describe('users', () => {
    it('all returns consistent key', () => {
      expect(queryKeys.users.all()).toEqual(['users']);
    });

    it('memberships includes userId', () => {
      expect(queryKeys.users.memberships(5)).toEqual(['userMemberships', 5]);
    });
  });

  describe('employees', () => {
    it('all includes orgId', () => {
      expect(queryKeys.employees.all(1)).toEqual(['employees', 1]);
    });

    it('list includes orgId and filters', () => {
      expect(queryKeys.employees.list(1, 'search', 'page')).toEqual([
        'employees',
        1,
        'search',
        'page',
      ]);
    });

    it('detail includes orgId and employeeId', () => {
      expect(queryKeys.employees.detail(1, 42)).toEqual(['employee', 1, 42]);
    });

    it('contracts includes orgId and employeeId', () => {
      expect(queryKeys.employees.contracts(1, 42)).toEqual(['employeeContracts', 1, 42]);
    });
  });

  describe('children', () => {
    it('all includes orgId', () => {
      expect(queryKeys.children.all(1)).toEqual(['children', 1]);
    });

    it('detail includes orgId and childId', () => {
      expect(queryKeys.children.detail(1, 99)).toEqual(['child', 1, 99]);
    });

    it('funding includes orgId', () => {
      expect(queryKeys.children.funding(1)).toEqual(['childrenFunding', 1]);
    });
  });

  describe('sections', () => {
    it('list includes orgId', () => {
      expect(queryKeys.sections.list(3)).toEqual(['sections', 3]);
    });
  });

  describe('attendance', () => {
    it('byDate includes orgId and date', () => {
      expect(queryKeys.attendance.byDate(1, '2025-06-15')).toEqual(['attendance', 1, '2025-06-15']);
    });

    it('summary includes orgId and date', () => {
      expect(queryKeys.attendance.summary(1, '2025-06-15')).toEqual([
        'attendanceSummary',
        1,
        '2025-06-15',
      ]);
    });
  });

  describe('statistics', () => {
    it('staffingHours includes optional params', () => {
      expect(queryKeys.statistics.staffingHours(1, 2, '2025-01-01', '2025-12-31')).toEqual([
        'staffingHours',
        1,
        2,
        '2025-01-01',
        '2025-12-31',
      ]);
    });

    it('staffingHours works without optional params', () => {
      expect(queryKeys.statistics.staffingHours(1)).toEqual([
        'staffingHours',
        1,
        undefined,
        undefined,
        undefined,
      ]);
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
});
