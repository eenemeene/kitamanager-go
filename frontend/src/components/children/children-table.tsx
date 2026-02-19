'use client';

import { useTranslations } from 'next-intl';
import { Pencil, Trash2, FileText, History } from 'lucide-react';
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
import type { Child, ChildFundingResponse, ContractProperties } from '@/lib/api/types';
import {
  formatDate,
  calculateAge,
  formatCurrency,
  formatFte,
  propertiesToValues,
} from '@/lib/utils/formatting';
import { getCurrentContract } from '@/lib/utils/contracts';

export interface ChildrenTableProps {
  items: Child[];
  fundingByChildId: Map<number, ChildFundingResponse>;
  weeklyHoursBasis?: number;
  onViewHistory: (child: Child) => void;
  onAddContract: (child: Child) => void;
  onEdit: (child: Child) => void;
  onDelete: (child: Child) => void;
}

export function ChildrenTable({
  items,
  fundingByChildId,
  weeklyHoursBasis,
  onViewHistory,
  onAddContract,
  onEdit,
  onDelete,
}: ChildrenTableProps) {
  const t = useTranslations();

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('common.name')}</TableHead>
          <TableHead>{t('gender.label')}</TableHead>
          <TableHead>{t('children.birthdate')}</TableHead>
          <TableHead>{t('children.age')}</TableHead>
          <TableHead>{t('sections.title')}</TableHead>
          <TableHead>{t('children.properties')}</TableHead>
          <TableHead className="text-right">{t('children.funding')}</TableHead>
          <TableHead className="text-right">
            {t('children.requirement')}
            {weeklyHoursBasis ? ` (${weeklyHoursBasis}h)` : ''}
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
              <TableCell>{t(`gender.${child.gender}`)}</TableCell>
              <TableCell>{formatDate(child.birthdate)}</TableCell>
              <TableCell>{calculateAge(child.birthdate)}</TableCell>
              <TableCell>
                {currentContract?.section_name && <span>{currentContract.section_name}</span>}
              </TableCell>
              <TableCell>
                {currentContract?.properties &&
                Object.keys(currentContract.properties).length > 0 ? (
                  <div className="flex flex-wrap gap-1">
                    {propertiesToValues(currentContract.properties as ContractProperties)
                      .slice(0, 3)
                      .map((value) => (
                        <Badge key={value} variant="outline" className="text-xs">
                          {value}
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
              <TableCell className="text-right">
                {(() => {
                  const funding = fundingByChildId.get(child.id);
                  if (!funding || funding.funding === 0) {
                    return <span className="text-muted-foreground text-sm">-</span>;
                  }
                  return <span className="font-medium">{formatCurrency(funding.funding)}</span>;
                })()}
              </TableCell>
              <TableCell className="text-right">
                {(() => {
                  const funding = fundingByChildId.get(child.id);
                  if (!funding || funding.requirement === 0) {
                    return <span className="text-muted-foreground text-sm">-</span>;
                  }
                  return <span className="font-medium">{formatFte(funding.requirement)}</span>;
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
            <TableCell colSpan={9} className="text-muted-foreground text-center">
              {t('common.noResults')}
            </TableCell>
          </TableRow>
        )}
      </TableBody>
    </Table>
  );
}
