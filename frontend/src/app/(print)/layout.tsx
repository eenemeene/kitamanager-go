'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/stores/auth-store';
import './print.css';

export default function PrintLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { isAuthenticated, checkAuth, hasHydrated } = useAuthStore();

  useEffect(() => {
    if (!hasHydrated) return;

    const isValid = checkAuth();
    if (!isValid) {
      router.push('/login');
    }
  }, [checkAuth, router, hasHydrated]);

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

  return <>{children}</>;
}
