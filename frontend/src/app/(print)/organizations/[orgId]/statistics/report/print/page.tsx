'use client';

import { useMemo } from 'react';
import { useParams, useSearchParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery, useIsFetching } from '@tanstack/react-query';
import { Printer } from 'lucide-react';
import { ChildrenReportSection } from '@/components/print-sections/children-report-section';
import { OccupancyReportSection } from '@/components/print-sections/occupancy-report-section';
import { StaffingReportSection } from '@/components/print-sections/staffing-report-section';
import { FinancialsReportSection } from '@/components/print-sections/financials-report-section';
import { ReportCover } from '@/components/print-sections/report-cover';
import { useUiStore } from '@/stores/ui-store';
import { parseReportMonth } from '@/lib/utils/report-month';

/**
 * Combined report page: cover + all four section components on one
 * scrollable page. End users hit File → Print to get a PDF straight
 * from the browser; the report-pdf CLI tool also navigates here as a
 * single Playwright session (no per-section loop, no merge step).
 *
 * data-print-ready waits for every TanStack query on the page to
 * settle (`useIsFetching() === 0`). With six section-component
 * queries firing in parallel plus the cover's lookups (deduplicated
 * by query key), the page typically becomes ready in 5–8s.
 *
 * Page-break behaviour is driven by inline @media print CSS at the
 * top of the render — Chromium honours `break-before: page` for
 * both the browser print dialog and Playwright's page.PDF().
 */
export default function CombinedReportPrintPage() {
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
      {/* Print CSS lives inline because it's tightly coupled to this
          page's structure: each `.report-section` becomes a fresh page
          in the printed PDF, and elements marked break-inside-avoid
          (charts, tables, summary cards) stay on a single page. */}
      <style>{`
        @media print {
          @page { size: A4 landscape; margin: 10mm; }
          body { print-color-adjust: exact; -webkit-print-color-adjust: exact; }
          .report-section { break-before: page; }
          .report-section:first-of-type { break-before: auto; }
          .break-inside-avoid { break-inside: avoid; }
          .no-print { display: none !important; }
        }
      `}</style>

      <div className="no-print mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t('report.title')}</h1>
          <p className="text-muted-foreground mt-1 text-sm">{orgName}</p>
        </div>
        <button
          className="bg-primary text-primary-foreground hover:bg-primary/90 inline-flex h-10 items-center gap-2 rounded-md px-4 text-sm font-medium"
          onClick={() => window.print()}
        >
          <Printer className="h-4 w-4" />
          {t('report.printAction')}
        </button>
      </div>

      <ReportCover orgId={orgId} orgName={orgName} reportMonth={reportMonth} />
      <ChildrenReportSection orgId={orgId} reportMonth={reportMonth} />
      <OccupancyReportSection orgId={orgId} reportMonth={reportMonth} />
      <StaffingReportSection orgId={orgId} reportMonth={reportMonth} />
      <FinancialsReportSection orgId={orgId} reportMonth={reportMonth} />
    </div>
  );
}
