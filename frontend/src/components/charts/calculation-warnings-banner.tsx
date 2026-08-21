'use client';

import { AlertTriangle } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import type { CalculationWarning } from '@/lib/api/types';

/**
 * CalculationWarningsBanner surfaces the per-row data-quality warnings
 * the backend's calculator emits when it has to skip a salary line.
 * Without this banner the user sees "lower than expected gross salary"
 * with no signal as to why — the silent under-count F1 was designed
 * to fix on the backend is only half-fixed until the UI shows them.
 *
 * We group by `code` so a 12-month forecast that skipped one
 * misconfigured contract surfaces as one row, not twelve. (The
 * backend already de-dupes per (code, contract_id), so we shouldn't
 * see runaway rows here even with grouping.)
 */
export function CalculationWarningsBanner({
  warnings,
}: {
  warnings: CalculationWarning[] | undefined | null;
}) {
  const t = useTranslations('statistics.warnings');

  if (!warnings || warnings.length === 0) {
    return null;
  }

  // Group by code so all "missing pay plan" rows render under one
  // heading. Order is "most occurrences first" so the user sees the
  // most actionable category at the top.
  const byCode = new Map<string, CalculationWarning[]>();
  for (const w of warnings) {
    const list = byCode.get(w.code) ?? [];
    list.push(w);
    byCode.set(w.code, list);
  }
  const codes = Array.from(byCode.keys()).sort(
    (a, b) => (byCode.get(b)?.length ?? 0) - (byCode.get(a)?.length ?? 0)
  );

  return (
    <Alert variant="destructive" data-testid="calculation-warnings-banner">
      <AlertTriangle className="h-4 w-4" />
      <AlertTitle>{t('title', { count: warnings.length })}</AlertTitle>
      <AlertDescription>
        <ul className="mt-2 list-disc space-y-1 pl-4">
          {codes.map((code) => {
            const items = byCode.get(code) ?? [];
            // Each item carries identifying metadata; render a compact
            // human-readable line per item rather than the raw backend
            // message (which is English-only and not user-localized).
            return items.map((w, i) => (
              <li key={`${code}-${w.contract_id ?? 'x'}-${i}`}>{renderWarning(t, w)}</li>
            ));
          })}
        </ul>
      </AlertDescription>
    </Alert>
  );
}

// renderWarning picks the most informative i18n string for the given
// warning and inserts the metadata. Falls back to the raw backend
// message if the code is unknown — covers backend evolution where a
// new code lands before the frontend is updated.
function renderWarning(t: ReturnType<typeof useTranslations>, w: CalculationWarning): string {
  const empPart = w.employee_id ? ` (employee #${w.employee_id})` : '';
  const datePart = w.date ? ` — ${w.date}` : '';
  switch (w.code) {
    case 'missing_pay_plan':
      return t('missingPayPlan', { payPlan: w.payplan_id ?? '?' }) + empPart + datePart;
    case 'no_pay_plan_period':
      return t('noPayPlanPeriod', { payPlan: w.payplan_id ?? '?' }) + empPart + datePart;
    case 'no_pay_plan_entry':
      return (
        t('noPayPlanEntry', {
          grade: w.grade ?? '?',
          step: w.step ?? '?',
        }) +
        empPart +
        datePart
      );
    case 'unusable_pay_plan_period':
      return t('unusablePayPlanPeriod', { payPlan: w.payplan_id ?? '?' }) + empPart + datePart;
    default:
      return w.message + empPart + datePart;
  }
}
