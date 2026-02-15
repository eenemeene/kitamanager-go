'use client';

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { PayPlanPeriod } from '@/lib/api/types';
import { formatCurrency } from '@/lib/utils/formatting';

function parseGrade(g: string): [number, string] {
  const match = g.match(/^[A-Za-z]*(\d+)(.*)$/);
  return match ? [parseInt(match[1]), match[2]] : [0, g];
}

interface PayPlanGridProps {
  period: PayPlanPeriod;
}

export function PayPlanGrid({ period }: PayPlanGridProps) {
  const entries = period.entries ?? [];

  const grades = Array.from(new Set(entries.map((e) => e.grade))).sort((a, b) => {
    const [numA, suffA] = parseGrade(a);
    const [numB, suffB] = parseGrade(b);
    if (numA !== numB) return numB - numA;
    return suffB.localeCompare(suffA);
  });

  const steps = Array.from(new Set(entries.map((e) => e.step))).sort((a, b) => a - b);

  const entryMap = new Map<string, number>();
  for (const e of entries) {
    entryMap.set(`${e.grade}-${e.step}`, e.monthly_amount);
  }

  const stepMinYearsMap = new Map<number, number>();
  for (const e of entries) {
    if (e.step_min_years != null && !stepMinYearsMap.has(e.step)) {
      stepMinYearsMap.set(e.step, e.step_min_years);
    }
  }

  if (grades.length === 0 || steps.length === 0) {
    return null;
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead />
          {steps.map((step) => (
            <TableHead key={step} className="text-right">
              {step}
              {stepMinYearsMap.get(step) != null && (
                <span className="ml-1 text-xs text-muted-foreground">
                  ({stepMinYearsMap.get(step)}y)
                </span>
              )}
            </TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {grades.map((grade) => (
          <TableRow key={grade}>
            <TableCell className="font-medium">{grade}</TableCell>
            {steps.map((step) => {
              const amount = entryMap.get(`${grade}-${step}`);
              return (
                <TableCell key={step} className="text-right">
                  {amount !== undefined ? formatCurrency(amount) : ''}
                </TableCell>
              );
            })}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
