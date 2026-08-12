'use client';

import { useTranslations } from 'next-intl';
import {
  Pencil,
  Trash2,
  FileText,
  History,
  Receipt,
  Ticket,
  ArrowDown,
  ArrowUp,
  AlertTriangle,
  CalendarCheck,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { HeaderWithTooltip } from '@/components/ui/header-with-tooltip';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { Badge } from '@/components/ui/badge';
import { SchoolEnrollmentBadge } from '@/components/children/school-enrollment-badge';
import type {
  Child,
  ChildContract,
  ChildFundingResponse,
  ChildBillingSummaryEntry,
  ContractProperties,
} from '@/lib/api/types';
import { formatDate, calculateAge, formatCurrency, formatFte } from '@/lib/utils/formatting';
import { propertiesToLabelKeys } from '@/lib/utils/contract-properties';
import { getCurrentContract, toUTCDate } from '@/lib/utils/contracts';
import { classifySchoolEnrollment } from '@/lib/utils/school-enrollment';

export interface ChildrenTableProps {
  items: Child[];
  fundingByChildId: Map<number, ChildFundingResponse>;
  billingSummaryByChildId: Map<number, ChildBillingSummaryEntry>;
  weeklyHoursBasis?: number;
  orgState?: string;
  onViewHistory: (child: Child) => void;
  onViewBilling: (child: Child) => void;
  onAddContract: (child: Child) => void;
  onEdit: (child: Child) => void;
  onDelete: (child: Child) => void;
  onManageVouchers: (child: Child) => void;
  onAdjustContractEnd?: (
    child: Child,
    contract: Pick<ChildContract, 'id' | 'version'>,
    newTo: string
  ) => void;
  isAdjustingContractEnd?: boolean;
}

export function ChildrenTable({
  items,
  fundingByChildId,
  billingSummaryByChildId,
  weeklyHoursBasis,
  orgState,
  onViewHistory,
  onViewBilling,
  onAddContract,
  onEdit,
  onDelete,
  onManageVouchers,
  onAdjustContractEnd,
  isAdjustingContractEnd,
}: ChildrenTableProps) {
  const t = useTranslations();
  const tLabels = useTranslations('fundingLabels');

  return (
    <TooltipProvider>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('common.name')}</TableHead>
            <TableHead className="hidden lg:table-cell">{t('gender.label')}</TableHead>
            <TableHead className="hidden lg:table-cell">{t('children.birthdate')}</TableHead>
            <TableHead className="hidden md:table-cell">{t('children.age')}</TableHead>
            <TableHead>{t('sections.title')}</TableHead>
            <TableHead className="hidden lg:table-cell">{t('children.properties')}</TableHead>
            <TableHead className="hidden text-right lg:table-cell">
              <HeaderWithTooltip
                label={t('children.funding')}
                tooltip={t('children.fundingTooltip')}
              />
            </TableHead>
            <TableHead className="hidden text-right lg:table-cell">
              <HeaderWithTooltip
                label={`${t('children.requirement')}${weeklyHoursBasis ? ` (${weeklyHoursBasis}h)` : ''}`}
                tooltip={t('children.requirementTooltip')}
              />
            </TableHead>
            <TableHead className="hidden text-right lg:table-cell">
              <HeaderWithTooltip
                label={t('children.billingDifference')}
                tooltip={t('children.billingDifferenceTooltip')}
              />
            </TableHead>
            <TableHead className="text-right">{t('common.actions')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((child) => {
            const currentContract = getCurrentContract(child.contracts);
            const enrollment = orgState
              ? classifySchoolEnrollment(child.birthdate, orgState)
              : null;
            const contractOverrun =
              enrollment && currentContract?.to
                ? toUTCDate(currentContract.to) > toUTCDate(enrollment.mussContractEnd)
                : false;
            return (
              <TableRow key={child.id}>
                <TableCell className="font-medium">
                  {child.first_name} {child.last_name}
                </TableCell>
                <TableCell className="hidden lg:table-cell">
                  {t(`gender.${child.gender}`)}
                </TableCell>
                <TableCell className="hidden lg:table-cell">
                  {formatDate(child.birthdate)}
                </TableCell>
                <TableCell className="hidden md:table-cell">
                  <div className="flex flex-col gap-1">
                    <span>{calculateAge(child.birthdate)}</span>
                    <SchoolEnrollmentBadge birthdate={child.birthdate} state={orgState} />
                  </div>
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    {currentContract?.section_name && <span>{currentContract.section_name}</span>}
                    {contractOverrun && enrollment && (
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <AlertTriangle
                            className="text-warning h-4 w-4 shrink-0"
                            aria-label={t('children.contractRunsPastSchoolStart', {
                              date: formatDate(enrollment.mussContractEnd),
                            })}
                          />
                        </TooltipTrigger>
                        <TooltipContent className="max-w-xs">
                          <p>
                            {t('children.contractRunsPastSchoolStart', {
                              date: formatDate(enrollment.mussContractEnd),
                            })}
                          </p>
                        </TooltipContent>
                      </Tooltip>
                    )}
                    {contractOverrun && enrollment && currentContract && onAdjustContractEnd && (
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="shrink-0"
                            disabled={isAdjustingContractEnd}
                            onClick={() =>
                              onAdjustContractEnd(
                                child,
                                currentContract,
                                enrollment.mussContractEnd
                              )
                            }
                            aria-label={t('children.adjustContractEnd', {
                              date: formatDate(enrollment.mussContractEnd),
                            })}
                          >
                            <CalendarCheck className="h-4 w-4" />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent className="max-w-xs">
                          <p>
                            {t('children.adjustContractEnd', {
                              date: formatDate(enrollment.mussContractEnd),
                            })}
                          </p>
                        </TooltipContent>
                      </Tooltip>
                    )}
                  </div>
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
                    const diff = billing.total_difference ?? 0;
                    const contractMonths = billing.contract_months ?? 0;
                    const coverage =
                      contractMonths > 0
                        ? `${billing.bill_count}/${contractMonths}`
                        : `${billing.bill_count}`;
                    return (
                      <div className="flex flex-col items-end gap-0.5">
                        <span
                          className={`inline-flex items-center gap-1 font-medium ${diff < 0 ? 'text-destructive' : diff > 0 ? 'text-success' : ''}`}
                        >
                          {diff < 0 ? (
                            <ArrowDown className="h-3 w-3" aria-hidden="true" />
                          ) : diff > 0 ? (
                            <ArrowUp className="h-3 w-3" aria-hidden="true" />
                          ) : null}
                          <span className="sr-only">
                            {diff < 0
                              ? t('children.billingShortfall')
                              : diff > 0
                                ? t('children.billingOverage')
                                : ''}
                          </span>
                          {formatCurrency(diff)}
                        </span>
                        <span className="text-muted-foreground text-xs">{coverage}</span>
                      </div>
                    );
                  })()}
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex flex-nowrap items-center justify-end gap-0.5">
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onViewHistory(child)}
                      title={t('children.contractHistory')}
                      aria-label={t('children.contractHistory')}
                      className="hidden lg:inline-flex"
                    >
                      <History className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onViewBilling(child)}
                      title={t('children.billingHistory')}
                      aria-label={t('children.billingHistory')}
                      className="hidden lg:inline-flex"
                    >
                      <Receipt className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onAddContract(child)}
                      title={t('children.addContract')}
                      aria-label={t('children.addContract')}
                      className="hidden lg:inline-flex"
                    >
                      <FileText className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onManageVouchers(child)}
                      title={t('vouchers.dialogTitle')}
                      aria-label={t('vouchers.dialogTitle')}
                      className="hidden lg:inline-flex"
                    >
                      <Ticket className="h-4 w-4" />
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
                  </div>
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
    </TooltipProvider>
  );
}
