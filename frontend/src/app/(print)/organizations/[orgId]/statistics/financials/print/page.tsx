'use client';

import { useMemo } from 'react';
import { useParams, useSearchParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useIsFetching } from '@tanstack/react-query';
import { Printer } from 'lucide-react';
import { FinancialsReportSection } from '@/components/print-sections/financials-report-section';
import { useUiStore } from '@/stores/ui-store';
import { parseReportMonth, formatReportMonthLong } from '@/lib/utils/report-month';

export default function FinancialsPrintPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const { organizations, fetchOrganizations } = useUiStore();
  const searchParams = useSearchParams();
  const reportMonth = useMemo(() => parseReportMonth(searchParams.get('month')), [searchParams]);

  useQuery({
    queryKey: ['organizations-load'],
    queryFn: async () => {
      if (organizations.length === 0) await fetchOrganizations();
      return null;
    },
  });

  const orgName = organizations.find((o) => o.id === orgId)?.name ?? '';
  const isFetching = useIsFetching();

  return (
    <div
      className="mx-auto max-w-[1100px] p-8"
      data-print-ready={isFetching === 0 ? 'true' : undefined}
    >
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">
            {t('nav.statisticsFinancials')} &middot; {formatReportMonthLong(reportMonth)}
          </h1>
          <p className="text-muted-foreground mt-1 text-sm">
            {orgName} &middot; {new Date().toLocaleDateString()}
          </p>
        </div>
        <button
          className="no-print bg-primary text-primary-foreground hover:bg-primary/90 inline-flex h-10 items-center gap-2 rounded-md px-4 text-sm font-medium"
          onClick={() => window.print()}
        >
          <Printer className="h-4 w-4" />
          {t('common.print')}
        </button>
      </div>

      <FinancialsReportSection orgId={orgId} reportMonth={reportMonth} />
    </div>
  );
}
