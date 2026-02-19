'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { AppSidebar } from '@/components/layout/app-sidebar';
import { AppHeader } from '@/components/layout/app-header';
import { ErrorBoundary } from '@/components/error-boundary';
import { useAuthStore } from '@/stores/auth-store';
import { useUiStore } from '@/stores/ui-store';
import { cn } from '@/lib/utils';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const t = useTranslations('common');
  const { isAuthenticated, checkAuth, loadUser, userLoaded, hasHydrated } = useAuthStore();
  const { sidebarCollapsed, fetchOrganizations } = useUiStore();

  useEffect(() => {
    // Wait for hydration to complete before checking auth
    if (!hasHydrated) return;

    const isValid = checkAuth();
    if (!isValid) {
      router.push('/login');
      return;
    }

    if (!userLoaded) {
      loadUser();
    }

    fetchOrganizations();
  }, [checkAuth, router, loadUser, userLoaded, fetchOrganizations, hasHydrated]);

  // Show loading state while waiting for hydration
  if (!hasHydrated) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="border-primary h-8 w-8 animate-spin rounded-full border-b-2"></div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return null;
  }

  return (
    <div className="bg-background min-h-screen">
      <AppSidebar />
      <AppHeader />
      <main
        className={cn('pt-16 transition-all duration-300', sidebarCollapsed ? 'ml-16' : 'ml-64')}
      >
        <div className="p-6">
          <ErrorBoundary
            title={t('somethingWentWrong')}
            message={t('unexpectedError')}
            retryLabel={t('retry')}
          >
            {children}
          </ErrorBoundary>
        </div>
      </main>
    </div>
  );
}
