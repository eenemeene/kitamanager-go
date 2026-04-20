'use client';

import { useTranslations } from 'next-intl';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { formatCurrency } from '@/lib/utils/formatting';

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

  return (
    <div className="grid gap-4 md:grid-cols-3">
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-muted-foreground text-sm font-medium">
            {t('statistics.totalIncome')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-success text-2xl font-bold">{formatCurrency(totalIncome)}</div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-muted-foreground text-sm font-medium">
            {t('statistics.totalExpenses')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-destructive text-2xl font-bold">{formatCurrency(totalExpenses)}</div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-muted-foreground text-sm font-medium">
            {t('statistics.balance')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className={`text-2xl font-bold ${balance >= 0 ? 'text-info' : 'text-destructive'}`}>
            {formatCurrency(balance)}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
