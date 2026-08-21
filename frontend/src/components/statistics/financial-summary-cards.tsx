'use client';

import { useTranslations } from 'next-intl';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { HeaderWithTooltip } from '@/components/ui/header-with-tooltip';
import { useFormatters } from '@/hooks/use-formatters';

interface FinancialSummaryCardsProps {
  totalIncome: number;
  totalExpenses: number;
  balance: number;
}

export function FinancialSummaryCards({
  totalIncome,
  totalExpenses,
  balance,
}: FinancialSummaryCardsProps) {
  const t = useTranslations();

  const fmt = useFormatters();
  return (
    <div className="grid gap-4 md:grid-cols-3">
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-muted-foreground text-sm font-medium">
            <HeaderWithTooltip
              label={t('statistics.totalIncome')}
              tooltip={t('statistics.totalIncomeTooltip')}
            />
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div data-visual-mask="currency" className="text-success text-2xl font-bold">
            {fmt.currency(totalIncome)}
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-muted-foreground text-sm font-medium">
            <HeaderWithTooltip
              label={t('statistics.totalExpenses')}
              tooltip={t('statistics.totalExpensesTooltip')}
            />
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div data-visual-mask="currency" className="text-destructive text-2xl font-bold">
            {fmt.currency(totalExpenses)}
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-muted-foreground text-sm font-medium">
            <HeaderWithTooltip
              label={t('statistics.balance')}
              tooltip={t('statistics.balanceTooltip')}
            />
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div
            data-visual-mask="currency"
            className={`text-2xl font-bold ${balance >= 0 ? 'text-info' : 'text-destructive'}`}
          >
            {fmt.currency(balance)}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
