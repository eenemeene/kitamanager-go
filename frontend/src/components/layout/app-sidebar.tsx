'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useTranslations } from 'next-intl';
import {
  LayoutDashboard,
  LayoutGrid,
  Building2,
  Users,
  Baby,
  BarChart3,
  Landmark,
  Wallet,
  Settings,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  type LucideIcon,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { useUiStore } from '@/stores/ui-store';
import { OrgSelector } from './org-selector';

interface NavChild {
  name: string;
  href: string;
  exact?: boolean;
}

interface NavItem {
  name: string;
  href: string;
  icon: LucideIcon;
  requiresOrg?: boolean;
  children?: NavChild[];
}

const navigation: NavItem[] = [
  { name: 'nav.organizations', href: '/organizations', icon: Building2, requiresOrg: false },
  {
    name: 'nav.governmentFundings',
    href: '/government-fundings',
    icon: Landmark,
    requiresOrg: false,
  },
];

const orgNavigation: NavItem[] = [
  { name: 'nav.dashboard', href: '/dashboard', icon: LayoutDashboard },
  {
    name: 'nav.employees',
    href: '/employees',
    icon: Users,
    children: [{ name: 'nav.payPlans', href: '/payplans' }],
  },
  { name: 'nav.children', href: '/children', icon: Baby },
  { name: 'nav.sections', href: '/sections', icon: LayoutGrid },
  {
    name: 'nav.statistics',
    href: '/statistics',
    icon: BarChart3,
    children: [
      { name: 'nav.statisticsOverview', href: '/statistics', exact: true },
      { name: 'nav.statisticsFinancials', href: '/statistics/financials' },
      { name: 'nav.statisticsStaffing', href: '/statistics/staffing' },
      { name: 'nav.statisticsChildren', href: '/statistics/children' },
      { name: 'nav.statisticsOccupancy', href: '/statistics/occupancy' },
      { name: 'nav.statisticsBudget', href: '/statistics/budget' },
    ],
  },
  { name: 'nav.budgetItems', href: '/budget-items', icon: Wallet },
  {
    name: 'nav.admin',
    href: '/users',
    icon: Settings,
    children: [
      { name: 'nav.users', href: '/users' },
      { name: 'nav.groups', href: '/groups' },
    ],
  },
];

export function AppSidebar() {
  const t = useTranslations();
  const pathname = usePathname();
  const { sidebarCollapsed, toggleSidebar, selectedOrganizationId } = useUiStore();
  const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set());

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
    for (const item of orgNavigation) {
      if (item.children && isAnyChildActive(item)) {
        setExpandedItems((prev) => {
          if (prev.has(item.name)) return prev;
          const next = new Set(prev);
          next.add(item.name);
          return next;
        });
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pathname, selectedOrganizationId]);

  return (
    <aside
      className={cn(
        'bg-background fixed top-0 left-0 z-40 flex h-screen flex-col border-r transition-all duration-300',
        sidebarCollapsed ? 'w-16' : 'w-64'
      )}
    >
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
          className={cn(sidebarCollapsed && 'mx-auto')}
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
        <ul className="space-y-1">
          {navigation.map((item) => {
            const Icon = item.icon;
            const active = isActive(item.href);
            return (
              <li key={item.name}>
                <Link
                  href={item.href}
                  className={cn(
                    'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                    active
                      ? 'bg-primary text-primary-foreground'
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                  )}
                >
                  <Icon className="h-5 w-5 shrink-0" />
                  {!sidebarCollapsed && <span>{t(item.name)}</span>}
                </Link>
              </li>
            );
          })}
        </ul>

        {/* Organization Selector */}
        {!sidebarCollapsed && (
          <div className="mt-6 px-3">
            <OrgSelector />
          </div>
        )}

        {/* Organization-scoped navigation */}
        {selectedOrganizationId && (
          <ul className="mt-4 space-y-1">
            {orgNavigation.map((item) => {
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
                    {/* Parent item: clicking toggles expand, but also navigates */}
                    <div className="flex items-center">
                      <Link
                        href={href}
                        className={cn(
                          'flex flex-1 items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                          anyChildActive
                            ? 'bg-primary/10 text-foreground'
                            : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                        )}
                      >
                        <Icon className="h-5 w-5 shrink-0" />
                        <span className="flex-1">{t(item.name)}</span>
                      </Link>
                      <button
                        onClick={() => toggleExpanded(item.name)}
                        className="text-muted-foreground hover:bg-muted hover:text-foreground mr-1 rounded-md p-1"
                      >
                        <ChevronDown
                          className={cn('h-4 w-4 transition-transform', isExpanded && 'rotate-180')}
                        />
                      </button>
                    </div>
                    {/* Children */}
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
                                    ? 'bg-primary text-primary-foreground'
                                    : 'text-muted-foreground hover:bg-muted hover:text-foreground'
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

              // Regular item (no children) or collapsed sidebar
              return (
                <li key={item.name}>
                  <Link
                    href={href}
                    className={cn(
                      'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                      parentActive
                        ? 'bg-primary text-primary-foreground'
                        : 'text-muted-foreground hover:bg-muted hover:text-foreground'
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
      </nav>
    </aside>
  );
}
