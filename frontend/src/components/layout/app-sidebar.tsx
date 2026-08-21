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
  ScrollText,
  CalendarCheck,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  type LucideIcon,
} from 'lucide-react';
import * as DialogPrimitive from '@radix-ui/react-dialog';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { useUiStore } from '@/stores/ui-store';
import { useCurrentRole, hasMinimumRole, type EffectiveRole } from '@/hooks/use-current-role';
import { useFrontendVersion } from '@/hooks/use-frontend-version';
import { useIsLgUp, useIsMdUp } from '@/hooks/use-media-query';
import { apiClient } from '@/lib/api/client';
import { OrgSelector } from './org-selector';

/** DOM id of the mobile drawer, so the header's trigger can `aria-controls` it. */
export const MOBILE_SIDEBAR_ID = 'mobile-sidebar';

/** Ties a submenu's `<ul>` to the chevron that expands it. */
const submenuId = (name: string) => `submenu-${name.replace(/[^a-zA-Z0-9]+/g, '-')}`;

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
    minRole: 'manager',
  },
];

const orgNavigationGroups: NavGroup[] = [
  {
    label: 'nav.groupDailyOperations',
    minRole: 'member',
    items: [
      { name: 'nav.dashboard', href: '/dashboard', icon: LayoutDashboard, minRole: 'member' },
      { name: 'nav.attendance', href: '/attendance', icon: CalendarCheck, minRole: 'staff' },
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
    minRole: 'manager',
    items: [
      {
        name: 'nav.governmentFundingBills',
        href: '/government-funding-bills',
        icon: Landmark,
        minRole: 'manager',
      },
      { name: 'nav.budgetItems', href: '/budget-items', icon: Wallet, minRole: 'manager' },
      {
        name: 'nav.statistics',
        href: '/statistics',
        icon: BarChart3,
        minRole: 'manager',
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
        minRole: 'manager',
      },
    ],
  },
  {
    label: 'nav.groupSettings',
    minRole: 'admin',
    items: [
      { name: 'nav.payPlans', href: '/payplans', icon: Settings, minRole: 'admin' },
      { name: 'nav.users', href: '/users', icon: Users, minRole: 'admin' },
      { name: 'nav.auditLog', href: '/audit-logs', icon: ScrollText, minRole: 'admin' },
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
  const isLgUp = useIsLgUp();
  const isMdUp = useIsMdUp();
  // At md (tablet portrait) the rail is always visually collapsed — the user
  // preference only applies at lg+ where there's room for the expanded version.
  const desktopCollapsed = sidebarCollapsed || !isLgUp;
  const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set());

  const { data: health } = useQuery({
    queryKey: ['health'],
    queryFn: () => apiClient.getHealth(),
    staleTime: Infinity,
    retry: false,
  });

  // The frontend ships as its own image and may be at a different version
  // than the API. Read it from the meta tag the root layout injects (same
  // source the report-pdf colophon uses, so the two never disagree).
  const webVersion = useFrontendVersion();

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

  const renderSidebar = (collapsed: boolean) => (
    <>
      {/* Header */}
      <div className="flex h-16 items-center justify-between border-b px-4">
        {!collapsed && (
          <Link href="/" className="text-xl font-bold">
            {t('common.appName')}
          </Link>
        )}
        <Button
          variant="ghost"
          size="icon"
          onClick={toggleSidebar}
          aria-label={t('common.toggleSidebar')}
          className={cn('hidden lg:inline-flex', sidebarCollapsed && 'mx-auto')}
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
                    aria-label={t(item.name)}
                    title={collapsed ? t(item.name) : undefined}
                    className={cn(
                      'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                      active
                        ? 'bg-sidebar-active text-sidebar-active-foreground'
                        : 'text-sidebar-foreground hover:bg-accent hover:text-accent-foreground'
                    )}
                  >
                    <Icon className="h-5 w-5 shrink-0" />
                    {!collapsed && <span>{t(item.name)}</span>}
                  </Link>
                </li>
              );
            })}
          </ul>
        )}

        {/* Organization Selector */}
        {!collapsed && (
          <div className="mt-6 px-3">
            <OrgSelector />
          </div>
        )}

        {/* Organization-scoped navigation grouped by section */}
        {selectedOrganizationId &&
          filteredOrgGroups.map((group) => (
            <div key={group.label} className="mt-4">
              {!collapsed && (
                <div className="text-sidebar-foreground/70 px-3 pb-1 text-[11px] font-semibold tracking-wider uppercase">
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

                  if (hasChildren && !collapsed) {
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
                          {/* Was a bare, unnamed button: screen readers announced
                              "button" and nothing more, with no indication of
                              whether the submenu was open, on a ~24px target.
                              Compact only at lg+, where there is a mouse. */}
                          <button
                            type="button"
                            onClick={() => toggleExpanded(item.name)}
                            aria-label={t('common.toggleSubmenu', { name: t(item.name) })}
                            aria-expanded={isExpanded}
                            aria-controls={submenuId(item.name)}
                            className="text-sidebar-foreground hover:bg-accent hover:text-accent-foreground mr-1 inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-md lg:h-9 lg:w-9"
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
                          <ul id={submenuId(item.name)} className="mt-1 ml-6 space-y-1">
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
                        aria-label={t(item.name)}
                        title={collapsed ? t(item.name) : undefined}
                        className={cn(
                          'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                          parentActive
                            ? 'bg-sidebar-active text-sidebar-active-foreground'
                            : 'text-sidebar-foreground hover:bg-accent hover:text-accent-foreground'
                        )}
                      >
                        <Icon className="h-5 w-5 shrink-0" />
                        {!collapsed && <span>{t(item.name)}</span>}
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}
      </nav>

      {/* Versions — API + Web shown separately because the two ship as
          independent images and can drift between releases. Each row is
          conditionally rendered: if either side fails to populate, the
          other still appears. */}
      {!collapsed && (health?.version || webVersion) && (
        <div
          data-visual-mask="version"
          className="text-sidebar-foreground/70 border-sidebar-border space-y-0.5 border-t px-4 py-2 text-[10px]"
        >
          {health?.version && <div>API: {health.version}</div>}
          {webVersion && <div>Web: {webVersion}</div>}
        </div>
      )}
    </>
  );

  // Radix owns the drawer's open state, which is what makes it a real dialog:
  // focus trap, Escape, scroll lock, `aria-modal`, and focus returned to the
  // hamburger on close. The hand-rolled version was a plain div — a keyboard
  // user tabbed straight through it into the page behind, and a screen reader
  // was never told a menu had opened. Since below md this drawer is the only
  // route to every other page, that was not a corner case.
  //
  // It has to close when the viewport reaches md: Radix keeps the trap and the
  // body's `pointer-events: none` alive for as long as it is open, so hiding it
  // with `md:hidden` alone would leave the whole app inert after a rotation.
  useEffect(() => {
    if (isMdUp && sidebarMobileOpen) {
      setMobileSidebarOpen(false);
    }
  }, [isMdUp, sidebarMobileOpen, setMobileSidebarOpen]);

  return (
    <>
      {/* Docked sidebar — icon rail at md (tablet portrait), expanded only at
          lg+ when the user preference says so. */}
      <aside
        className={cn(
          'bg-sidebar border-sidebar-border fixed top-0 left-0 z-40 hidden h-screen flex-col border-r transition-all duration-300 md:flex md:w-16',
          sidebarCollapsed ? 'lg:w-16' : 'lg:w-64'
        )}
      >
        {renderSidebar(desktopCollapsed)}
      </aside>

      {/* Mobile drawer */}
      <DialogPrimitive.Root open={sidebarMobileOpen} onOpenChange={setMobileSidebarOpen}>
        <DialogPrimitive.Portal>
          <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/50 md:hidden" />
          <DialogPrimitive.Content
            id={MOBILE_SIDEBAR_ID}
            // No description: the panel is a list of links and needs no prose.
            // Passing undefined explicitly stops Radix warning about it.
            aria-describedby={undefined}
            className="bg-sidebar border-sidebar-border fixed top-0 left-0 z-50 flex h-[100dvh] w-64 flex-col border-r focus:outline-none md:hidden"
            // Close on the tap that navigates, rather than reacting to the path
            // afterwards. The old effect keyed on `pathname` shut the drawer for
            // any navigation, including ones the user did not ask for: landing on
            // `/` renders this whole shell and only then redirects to the org
            // dashboard once the organization list arrives, so a menu opened in
            // that second was torn down by a redirect. It also ran on mount.
            //
            // Delegating to the panel keeps it to one handler, so a link added
            // later cannot forget to close the drawer behind itself.
            onClick={(event) => {
              if ((event.target as HTMLElement).closest('a')) {
                setMobileSidebarOpen(false);
              }
            }}
          >
            {/* Radix requires a title for the dialog's accessible name. It is
                visually redundant beside the app name in the panel header, so
                it is exposed to assistive tech only. */}
            <DialogPrimitive.Title className="sr-only">
              {t('common.mainNavigation')}
            </DialogPrimitive.Title>
            {renderSidebar(false)}
          </DialogPrimitive.Content>
        </DialogPrimitive.Portal>
      </DialogPrimitive.Root>
    </>
  );
}
