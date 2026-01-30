'use client';

import { useTranslations } from 'next-intl';
import { Building2, Users, Baby, UserCog } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { useUiStore } from '@/stores/ui-store';
import { useAuthStore } from '@/stores/auth-store';

function StatCard({
  title,
  value,
  icon: Icon,
  loading,
}: {
  title: string;
  value: number | string;
  icon: React.ComponentType<{ className?: string }>;
  loading?: boolean;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        {loading ? (
          <Skeleton className="h-8 w-20" />
        ) : (
          <div className="text-2xl font-bold">{value}</div>
        )}
      </CardContent>
    </Card>
  );
}

export default function DashboardPage() {
  const t = useTranslations();
  const { organizations, organizationsLoading } = useUiStore();
  const { user } = useAuthStore();

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t('dashboard.title')}</h1>
        <p className="text-muted-foreground">
          {t('dashboard.welcome')}
          {user?.name && `, ${user.name}`}
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title={t('dashboard.totalOrganizations')}
          value={organizations.length}
          icon={Building2}
          loading={organizationsLoading}
        />
        <StatCard title={t('dashboard.totalEmployees')} value="-" icon={Users} loading={false} />
        <StatCard title={t('dashboard.totalChildren')} value="-" icon={Baby} loading={false} />
        <StatCard title={t('dashboard.totalUsers')} value="-" icon={UserCog} loading={false} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('dashboard.quickStats')}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">{t('statistics.selectOrgForStats')}</p>
        </CardContent>
      </Card>
    </div>
  );
}
