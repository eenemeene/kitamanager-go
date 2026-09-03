'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { Building2, ChevronsUpDown } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useUiStore } from '@/stores/ui-store';
import { useAuthStore } from '@/stores/auth-store';

export function OrgSelector() {
  const t = useTranslations();
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const {
    organizations,
    organizationsLoading,
    selectedOrganizationId,
    setSelectedOrganization,
    fetchOrganizations,
    getSelectedOrganization,
  } = useUiStore();

  useEffect(() => {
    if (isAuthenticated && organizations.length === 0) {
      fetchOrganizations();
    }
  }, [isAuthenticated, organizations.length, fetchOrganizations]);

  const selectedOrg = getSelectedOrganization();

  const handleSelect = (orgId: number) => {
    setSelectedOrganization(orgId);
    // Navigate to the organization's default page
    router.push(`/organizations/${orgId}/dashboard`);
  };

  if (organizationsLoading) {
    return (
      <Button variant="outline" className="w-full justify-start" disabled>
        <Building2 className="mr-2 h-4 w-4" />
        {t('common.loading')}
      </Button>
    );
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" className="w-full justify-between" data-testid="org-selector">
          {/*
            min-w-0 on both, or the truncate below never fires: a flex item
            defaults to min-width:auto and will not shrink under its content, so
            a long organisation name overflowed the button instead of
            ellipsising. Only visible once a name is longer than the sidebar.
          */}
          <span className="flex min-w-0 items-center">
            <Building2 className="mr-2 h-4 w-4 shrink-0" />
            <span className="min-w-0 truncate">
              {selectedOrg ? selectedOrg.name : t('organizations.selectOrg')}
            </span>
          </span>
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-56">
        <DropdownMenuLabel>{t('organizations.selectOrg')}</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {organizations.map((org) => (
          <DropdownMenuItem
            key={org.id}
            onClick={() => handleSelect(org.id)}
            className={org.id === selectedOrganizationId ? 'bg-accent' : ''}
          >
            <Building2 className="mr-2 h-4 w-4" />
            {org.name}
          </DropdownMenuItem>
        ))}
        {organizations.length === 0 && (
          <DropdownMenuItem disabled>{t('common.noResults')}</DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
