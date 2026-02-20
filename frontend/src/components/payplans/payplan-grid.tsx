'use client';

import React from 'react';
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
          {steps.map((step, i) => (
            <React.Fragment key={step}>
              {i > 0 && <TableHead />}
              <TableHead className="text-right">
                {step}
                {stepMinYearsMap.get(step) != null && (
                  <span className="text-muted-foreground ml-1 text-xs">
                    ({stepMinYearsMap.get(step)}y)
                  </span>
                )}
              </TableHead>
            </React.Fragment>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {grades.map((grade) => (
          <TableRow key={grade}>
            <TableCell className="font-medium">{grade}</TableCell>
            {steps.map((step, i) => {
              const amount = entryMap.get(`${grade}-${step}`);
              const prevAmount = i > 0 ? entryMap.get(`${grade}-${steps[i - 1]}`) : undefined;
              const pctIncrease =
                amount !== undefined && prevAmount !== undefined && prevAmount > 0
                  ? ((amount - prevAmount) / prevAmount) * 100
                  : undefined;
              return (
                <React.Fragment key={step}>
                  {i > 0 && (
                    <TableCell className="px-1 text-center text-[0.65rem] leading-tight text-emerald-600 dark:text-emerald-400">
                      {pctIncrease !== undefined ? `↗${pctIncrease.toFixed(1)}%` : ''}
                    </TableCell>
                  )}
                  <TableCell className="text-right">
                    {amount !== undefined ? formatCurrency(amount) : ''}
                  </TableCell>
                </React.Fragment>
              );
            })}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
