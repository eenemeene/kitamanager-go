'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import {
  LayoutDashboard,
  LayoutGrid,
  Building2,
  Users,
  Baby,
  BarChart3,
  TrendingUp,
  Landmark,
  Wallet,
  Settings,
  CalendarCheck,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  type LucideIcon,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { useUiStore } from '@/stores/ui-store';
import { useCurrentRole, hasMinimumRole, type EffectiveRole } from '@/hooks/use-current-role';
import { apiClient } from '@/lib/api/client';
import { OrgSelector } from './org-selector';

interface NavChild {
  name: string;
  href: string;
  exact?: boolean;
  minRole?: EffectiveRole;
}

interface NavItem {
  name: string;
  href: string;
  icon: LucideIcon;
  requiresOrg?: boolean;
  minRole?: EffectiveRole;
  children?: NavChild[];
}

interface NavGroup {
  label: string;
  minRole?: EffectiveRole;
  items: NavItem[];
}

const globalNavigation: NavItem[] = [
  {
    name: 'nav.organizations',
    href: '/organizations',
    icon: Building2,
    requiresOrg: false,
    minRole: 'superadmin',
  },
  {
    name: 'nav.governmentFundings',
    href: '/government-funding-rates',
    icon: Landmark,
    requiresOrg: false,
    minRole: 'superadmin',
  },
];

const orgNavigationGroups: NavGroup[] = [
  {
    label: 'nav.groupDailyOperations',
    minRole: 'member',
    items: [
      { name: 'nav.dashboard', href: '/dashboard', icon: LayoutDashboard, minRole: 'member' },
      { name: 'nav.attendance', href: '/attendance', icon: CalendarCheck, minRole: 'member' },
      { name: 'nav.sections', href: '/sections', icon: LayoutGrid, minRole: 'manager' },
    ],
  },
  {
    label: 'nav.groupPeople',
    minRole: 'member',
    items: [
      { name: 'nav.children', href: '/children', icon: Baby, minRole: 'member' },
      { name: 'nav.employees', href: '/employees', icon: Users, minRole: 'manager' },
    ],
  },
  {
    label: 'nav.groupFinance',
    minRole: 'admin',
    items: [
      {
        name: 'nav.governmentFundingBills',
        href: '/government-funding-bills',
        icon: Landmark,
        minRole: 'admin',
      },
      { name: 'nav.budgetItems', href: '/budget-items', icon: Wallet, minRole: 'admin' },
      {
        name: 'nav.statistics',
        href: '/statistics',
        icon: BarChart3,
        minRole: 'admin',
        children: [
          { name: 'nav.statisticsOverview', href: '/statistics', exact: true },
          { name: 'nav.statisticsFinancials', href: '/statistics/financials' },
          { name: 'nav.statisticsStaffing', href: '/statistics/staffing' },
          { name: 'nav.statisticsChildren', href: '/statistics/children' },
          { name: 'nav.statisticsOccupancy', href: '/statistics/occupancy' },
        ],
      },
      {
        name: 'nav.statisticsForecast',
        href: '/statistics/forecast',
        icon: TrendingUp,
        minRole: 'admin',
      },
    ],
  },
  {
    label: 'nav.groupSettings',
    minRole: 'admin',
    items: [
      { name: 'nav.payPlans', href: '/payplans', icon: Settings, minRole: 'admin' },
      { name: 'nav.users', href: '/users', icon: Users, minRole: 'admin' },
    ],
  },
];

export function AppSidebar() {
  const t = useTranslations();
  const pathname = usePathname();
  const {
    sidebarCollapsed,
    toggleSidebar,
    selectedOrganizationId,
    sidebarMobileOpen,
    setMobileSidebarOpen,
  } = useUiStore();
  const currentRole = useCurrentRole();
  const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set());

  const { data: health } = useQuery({
    queryKey: ['health'],
    queryFn: () => apiClient.getHealth(),
    staleTime: Infinity,
    retry: false,
  });

  const filteredGlobalNavigation = globalNavigation.filter(
    (item) => !item.minRole || hasMinimumRole(currentRole, item.minRole)
  );

  const filteredOrgGroups = orgNavigationGroups
    .map((group) => {
      const filteredItems = group.items
        .filter((item) => !item.minRole || hasMinimumRole(currentRole, item.minRole))
        .map((item) => {
          if (!item.children) return item;
          const filteredChildren = item.children.filter(
            (child) => !child.minRole || hasMinimumRole(currentRole, child.minRole)
          );
          return { ...item, children: filteredChildren };
        });
      return { ...group, items: filteredItems };
    })
    .filter((group) => group.items.length > 0);

  const isActive = (href: string) => {
    return pathname.startsWith(href);
  };

  const getOrgHref = (path: string) => {
    if (!selectedOrganizationId) return '#';
    return `/organizations/${selectedOrganizationId}${path}`;
  };

  const isChildActive = (child: NavChild) => {
    const fullHref = getOrgHref(child.href);
    if (child.exact) {
      return pathname === fullHref;
    }
    return pathname.startsWith(fullHref);
  };

  const isAnyChildActive = (item: NavItem) => {
    if (!item.children) return false;
    return item.children.some((child) => isChildActive(child));
  };

  const toggleExpanded = (name: string) => {
    setExpandedItems((prev) => {
      const next = new Set(prev);
      if (next.has(name)) {
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  };

  // Auto-expand parent when a child route is active
  useEffect(() => {
    for (const group of filteredOrgGroups) {
      for (const item of group.items) {
        if (item.children && item.children.length > 0 && isAnyChildActive(item)) {
          setExpandedItems((prev) => {
            if (prev.has(item.name)) return prev;
            const next = new Set(prev);
            next.add(item.name);
            return next;
          });
        }
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pathname, selectedOrganizationId]);

  // Close mobile sidebar on navigation
  useEffect(() => {
    setMobileSidebarOpen(false);
  }, [pathname, setMobileSidebarOpen]);

  const sidebarContent = (
    <>
      {/* Header */}
      <div className="flex h-16 items-center justify-between border-b px-4">
        {!sidebarCollapsed && (
          <Link href="/" className="text-xl font-bold">
            {t('common.appName')}
          </Link>
        )}
        <Button
          variant="ghost"
          size="icon"
          onClick={toggleSidebar}
          aria-label={t('common.toggleSidebar')}
          className={cn('hidden md:inline-flex', sidebarCollapsed && 'mx-auto')}
        >
          {sidebarCollapsed ? (
            <ChevronRight className="h-4 w-4" />
          ) : (
            <ChevronLeft className="h-4 w-4" />
          )}
        </Button>
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto p-2">
        {/* Global navigation (superadmin) */}
        {filteredGlobalNavigation.length > 0 && (
          <ul className="space-y-1">
            {filteredGlobalNavigation.map((item) => {
              const Icon = item.icon;
              const active = isActive(item.href);
              return (
                <li key={item.name}>
                  <Link
                    href={item.href}
                    className={cn(
                      'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                      active
                        ? 'bg-sidebar-active text-sidebar-active-foreground'
                        : 'text-sidebar-foreground hover:bg-accent hover:text-accent-foreground'
                    )}
                  >
                    <Icon className="h-5 w-5 shrink-0" />
                    {!sidebarCollapsed && <span>{t(item.name)}</span>}
                  </Link>
                </li>
              );
            })}
          </ul>
        )}

        {/* Organization Selector */}
        {!sidebarCollapsed && (
          <div className="mt-6 px-3">
            <OrgSelector />
          </div>
        )}

        {/* Organization-scoped navigation grouped by section */}
        {selectedOrganizationId &&
          filteredOrgGroups.map((group) => (
            <div key={group.label} className="mt-4">
              {!sidebarCollapsed && (
                <div className="text-sidebar-foreground/50 px-3 pb-1 text-[11px] font-semibold tracking-wider uppercase">
                  {t(group.label)}
                </div>
              )}
              <ul className="space-y-1">
                {group.items.map((item) => {
                  const Icon = item.icon;
                  const href = getOrgHref(item.href);
                  const hasChildren = item.children && item.children.length > 0;
                  const anyChildActive = isAnyChildActive(item);
                  const isExpanded = expandedItems.has(item.name);
                  const parentActive = pathname.includes(
                    `/organizations/${selectedOrganizationId}${item.href}`
                  );

                  if (hasChildren && !sidebarCollapsed) {
                    return (
                      <li key={item.name}>
                        <div className="flex items-center">
                          <Link
                            href={href}
                            className={cn(
                              'flex flex-1 items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                              anyChildActive
                                ? 'bg-sidebar-active/10 text-sidebar-foreground'
                                : 'text-sidebar-foreground hover:bg-accent hover:text-accent-foreground'
                            )}
                          >
                            <Icon className="h-5 w-5 shrink-0" />
                            <span className="flex-1">{t(item.name)}</span>
                          </Link>
                          <button
                            onClick={() => toggleExpanded(item.name)}
                            className="text-sidebar-foreground hover:bg-accent hover:text-accent-foreground mr-1 rounded-md p-1"
                          >
                            <ChevronDown
                              className={cn(
                                'h-4 w-4 transition-transform',
                                isExpanded && 'rotate-180'
                              )}
                            />
                          </button>
                        </div>
                        {isExpanded && (
                          <ul className="mt-1 ml-6 space-y-1">
                            {item.children!.map((child) => {
                              const childHref = getOrgHref(child.href);
                              const childActive = isChildActive(child);
                              return (
                                <li key={child.name}>
                                  <Link
                                    href={childHref}
                                    className={cn(
                                      'flex items-center rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                                      childActive
                                        ? 'bg-sidebar-active text-sidebar-active-foreground'
                                        : 'text-sidebar-foreground hover:bg-accent hover:text-accent-foreground'
                                    )}
                                  >
                                    {t(child.name)}
                                  </Link>
                                </li>
                              );
                            })}
                          </ul>
                        )}
                      </li>
                    );
                  }

                  return (
                    <li key={item.name}>
                      <Link
                        href={href}
                        className={cn(
                          'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                          parentActive
                            ? 'bg-sidebar-active text-sidebar-active-foreground'
                            : 'text-sidebar-foreground hover:bg-accent hover:text-accent-foreground'
                        )}
                      >
                        <Icon className="h-5 w-5 shrink-0" />
                        {!sidebarCollapsed && <span>{t(item.name)}</span>}
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}
      </nav>

      {/* Version */}
      {health?.version && !sidebarCollapsed && (
        <div className="text-sidebar-foreground/60 border-sidebar-border border-t px-4 py-2 text-[10px]">
          version: {health.version}
        </div>
      )}
    </>
  );

  return (
    <>
      {/* Desktop sidebar */}
      <aside
        className={cn(
          'bg-sidebar border-sidebar-border fixed top-0 left-0 z-40 hidden h-screen flex-col border-r transition-all duration-300 md:flex',
          sidebarCollapsed ? 'w-16' : 'w-64'
        )}
      >
        {sidebarContent}
      </aside>

      {/* Mobile sidebar overlay */}
      {sidebarMobileOpen && (
        <div className="fixed inset-0 z-50 flex md:hidden">
          {/* Backdrop */}
          <div
            className="fixed inset-0 bg-black/50"
            onClick={() => setMobileSidebarOpen(false)}
            aria-hidden="true"
          />
          {/* Sidebar panel */}
          <aside className="bg-sidebar border-sidebar-border relative flex h-screen w-64 flex-col border-r">
            {sidebarContent}
          </aside>
        </div>
      )}
    </>
  );
}
