'use client';

import { useTranslations } from 'next-intl';
import { Pencil, Trash2, FileText, History, Receipt } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import type {
  Child,
  ChildFundingResponse,
  ChildBillingSummaryEntry,
  ContractProperties,
} from '@/lib/api/types';
import { formatDate, calculateAge, formatCurrency, formatFte } from '@/lib/utils/formatting';
import { propertiesToLabelKeys } from '@/lib/utils/contract-properties';
import { getCurrentContract } from '@/lib/utils/contracts';

export interface ChildrenTableProps {
  items: Child[];
  fundingByChildId: Map<number, ChildFundingResponse>;
  billingSummaryByChildId: Map<number, ChildBillingSummaryEntry>;
  weeklyHoursBasis?: number;
  onViewHistory: (child: Child) => void;
  onViewBilling: (child: Child) => void;
  onAddContract: (child: Child) => void;
  onEdit: (child: Child) => void;
  onDelete: (child: Child) => void;
}

export function ChildrenTable({
  items,
  fundingByChildId,
  billingSummaryByChildId,
  weeklyHoursBasis,
  onViewHistory,
  onViewBilling,
  onAddContract,
  onEdit,
  onDelete,
}: ChildrenTableProps) {
  const t = useTranslations();
  const tLabels = useTranslations('fundingLabels');

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('common.name')}</TableHead>
          <TableHead className="hidden md:table-cell">{t('gender.label')}</TableHead>
          <TableHead className="hidden md:table-cell">{t('children.birthdate')}</TableHead>
          <TableHead className="hidden md:table-cell">{t('children.age')}</TableHead>
          <TableHead>{t('sections.title')}</TableHead>
          <TableHead className="hidden lg:table-cell">{t('children.properties')}</TableHead>
          <TableHead className="hidden text-right lg:table-cell">{t('children.funding')}</TableHead>
          <TableHead className="hidden text-right lg:table-cell">
            {t('children.requirement')}
            {weeklyHoursBasis ? ` (${weeklyHoursBasis}h)` : ''}
          </TableHead>
          <TableHead className="hidden text-right lg:table-cell">
            {t('children.billingDifference')}
          </TableHead>
          <TableHead className="text-right">{t('common.actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((child) => {
          const currentContract = getCurrentContract(child.contracts);
          return (
            <TableRow key={child.id}>
              <TableCell className="font-medium">
                {child.first_name} {child.last_name}
              </TableCell>
              <TableCell className="hidden md:table-cell">{t(`gender.${child.gender}`)}</TableCell>
              <TableCell className="hidden md:table-cell">{formatDate(child.birthdate)}</TableCell>
              <TableCell className="hidden md:table-cell">
                {calculateAge(child.birthdate)}
              </TableCell>
              <TableCell>
                {currentContract?.section_name && <span>{currentContract.section_name}</span>}
              </TableCell>
              <TableCell className="hidden lg:table-cell">
                {currentContract?.properties &&
                Object.keys(currentContract.properties).length > 0 ? (
                  <div className="flex flex-wrap gap-1">
                    {propertiesToLabelKeys(currentContract.properties as ContractProperties)
                      .slice(0, 3)
                      .map((labelKey) => (
                        <Badge key={labelKey} variant="outline" className="text-xs">
                          {tLabels.has(labelKey) ? tLabels(labelKey) : labelKey.split('--').pop()}
                        </Badge>
                      ))}
                    {Object.keys(currentContract.properties).length > 3 && (
                      <Badge variant="outline" className="text-xs">
                        +{Object.keys(currentContract.properties).length - 3}
                      </Badge>
                    )}
                  </div>
                ) : (
                  <span className="text-muted-foreground text-sm">
                    {t('contracts.noProperties')}
                  </span>
                )}
              </TableCell>
              <TableCell className="hidden text-right lg:table-cell">
                {(() => {
                  const funding = fundingByChildId.get(child.id);
                  if (!funding || funding.funding === 0) {
                    return <span className="text-muted-foreground text-sm">-</span>;
                  }
                  return <span className="font-medium">{formatCurrency(funding.funding)}</span>;
                })()}
              </TableCell>
              <TableCell className="hidden text-right lg:table-cell">
                {(() => {
                  const funding = fundingByChildId.get(child.id);
                  if (!funding || funding.requirement === 0) {
                    return <span className="text-muted-foreground text-sm">-</span>;
                  }
                  return <span className="font-medium">{formatFte(funding.requirement)}</span>;
                })()}
              </TableCell>
              <TableCell className="hidden text-right lg:table-cell">
                {(() => {
                  const billing = billingSummaryByChildId.get(child.id);
                  if (!billing || billing.bill_count === 0) {
                    return <span className="text-muted-foreground text-sm">-</span>;
                  }
                  const diff = billing.total_difference;
                  return (
                    <span
                      className={`font-medium ${diff < 0 ? 'text-red-600' : diff > 0 ? 'text-green-600' : ''}`}
                    >
                      {formatCurrency(diff)}
                    </span>
                  );
                })()}
              </TableCell>
              <TableCell className="text-right">
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => onViewHistory(child)}
                  title={t('children.contractHistory')}
                  aria-label={t('children.contractHistory')}
                >
                  <History className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => onViewBilling(child)}
                  title={t('children.billingHistory')}
                  aria-label={t('children.billingHistory')}
                >
                  <Receipt className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => onAddContract(child)}
                  title={t('children.addContract')}
                  aria-label={t('children.addContract')}
                >
                  <FileText className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => onEdit(child)}
                  aria-label={t('common.edit')}
                >
                  <Pencil className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => onDelete(child)}
                  aria-label={t('common.delete')}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </TableCell>
            </TableRow>
          );
        })}
        {items.length === 0 && (
          <TableRow>
            <TableCell colSpan={99} className="text-muted-foreground text-center">
              {t('common.noResults')}
            </TableCell>
          </TableRow>
        )}
      </TableBody>
    </Table>
  );
}
