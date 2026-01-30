'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useTranslations } from 'next-intl';
import {
  LayoutDashboard,
  Building2,
  Users,
  Baby,
  UserCog,
  UsersRound,
  BarChart3,
  Landmark,
  CircleDollarSign,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { useUiStore } from '@/stores/ui-store';
import { OrgSelector } from './org-selector';

const navigation = [
  { name: 'nav.dashboard', href: '/', icon: LayoutDashboard, requiresOrg: false },
  { name: 'nav.organizations', href: '/organizations', icon: Building2, requiresOrg: false },
  {
    name: 'nav.governmentFundings',
    href: '/government-fundings',
    icon: Landmark,
    requiresOrg: false,
  },
];

const orgNavigation = [
  { name: 'nav.users', href: '/users', icon: UserCog },
  { name: 'nav.groups', href: '/groups', icon: UsersRound },
  { name: 'nav.employees', href: '/employees', icon: Users },
  { name: 'nav.children', href: '/children', icon: Baby },
  { name: 'nav.statistics', href: '/statistics', icon: BarChart3 },
  { name: 'nav.payPlans', href: '/payplans', icon: CircleDollarSign },
];

export function AppSidebar() {
  const t = useTranslations();
  const pathname = usePathname();
  const { sidebarCollapsed, toggleSidebar, selectedOrganizationId } = useUiStore();

  const isActive = (href: string) => {
    if (href === '/') return pathname === '/';
    return pathname.startsWith(href);
  };

  const getOrgHref = (path: string) => {
    if (!selectedOrganizationId) return '#';
    return `/organizations/${selectedOrganizationId}${path}`;
  };

  return (
    <aside
      className={cn(
        'fixed left-0 top-0 z-40 flex h-screen flex-col border-r bg-background transition-all duration-300',
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
              const active = pathname.includes(
                `/organizations/${selectedOrganizationId}${item.href}`
              );
              return (
                <li key={item.name}>
                  <Link
                    href={href}
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
        )}
      </nav>
    </aside>
  );
}
