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
import type { Employee, PayPlan } from '@/lib/api/types';
import { formatDate, calculateAge, formatCurrency } from '@/lib/utils/formatting';
import { getCurrentContract } from '@/lib/utils/contracts';
import { calculateMonthlySalary } from '@/lib/utils/salary';
import { calculateYearsOfService } from '@/lib/utils/step-promotions';

export interface EmployeesTableProps {
  employees: Employee[];
  payPlanMap: Map<number, PayPlan>;
  onViewHistory: (employee: Employee) => void;
  onAddContract: (employee: Employee) => void;
  onEdit: (employee: Employee) => void;
  onDelete: (employee: Employee) => void;
}

export function EmployeesTable({
  employees,
  payPlanMap,
  onViewHistory,
  onAddContract,
  onEdit,
  onDelete,
}: EmployeesTableProps) {
  const t = useTranslations();

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('common.name')}</TableHead>
          <TableHead className="hidden md:table-cell">{t('gender.label')}</TableHead>
          <TableHead className="hidden md:table-cell">{t('employees.birthdate')}</TableHead>
          <TableHead className="hidden md:table-cell">{t('employees.age')}</TableHead>
          <TableHead>{t('employees.staffCategory.label')}</TableHead>
          <TableHead className="hidden lg:table-cell">{t('employees.grade')}</TableHead>
          <TableHead className="hidden lg:table-cell">{t('employees.weeklyHours')}</TableHead>
          <TableHead className="hidden lg:table-cell">{t('employees.salary')}</TableHead>
          <TableHead className="hidden lg:table-cell">{t('employees.yearsOfService')}</TableHead>
          <TableHead className="text-right">{t('common.actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {employees.map((employee) => {
          const currentContract = getCurrentContract(employee.contracts);
          const payPlanForSalary = currentContract?.payplan_id
            ? payPlanMap.get(currentContract.payplan_id)
            : undefined;
          const salary =
            currentContract && payPlanForSalary
              ? calculateMonthlySalary(currentContract, payPlanForSalary)
              : null;
          const yearsOfService = employee.contracts?.length
            ? calculateYearsOfService(employee.contracts)
            : null;
          return (
            <TableRow key={employee.id}>
              <TableCell className="font-medium">
                {employee.first_name} {employee.last_name}
              </TableCell>
              <TableCell className="hidden md:table-cell">
                {t(`gender.${employee.gender}`)}
              </TableCell>
              <TableCell className="hidden md:table-cell">
                {formatDate(employee.birthdate)}
              </TableCell>
              <TableCell className="hidden md:table-cell">
                {calculateAge(employee.birthdate)}
              </TableCell>
              <TableCell>
                {currentContract ? (
                  t(`employees.staffCategory.${currentContract.staff_category}`)
                ) : (
                  <span className="text-muted-foreground">{t('employees.noContract')}</span>
                )}
              </TableCell>
              <TableCell className="hidden lg:table-cell">
                {currentContract ? `${currentContract.grade} / ${currentContract.step}` : '-'}
              </TableCell>
              <TableCell className="hidden lg:table-cell">
                {currentContract?.weekly_hours || '-'}
              </TableCell>
              <TableCell className="hidden lg:table-cell">
                {salary !== null ? formatCurrency(salary) : '-'}
              </TableCell>
              <TableCell className="hidden lg:table-cell">
                {yearsOfService !== null ? yearsOfService.toFixed(1) : '-'}
              </TableCell>
              <TableCell className="text-right">
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => onViewHistory(employee)}
                  title={t('employees.contractHistory')}
                  aria-label={t('employees.contractHistory')}
                >
                  <History className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => onAddContract(employee)}
                  title={t('employees.addContract')}
                  aria-label={t('employees.addContract')}
                >
                  <FileText className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => onEdit(employee)}
                  aria-label={t('common.edit')}
                >
                  <Pencil className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => onDelete(employee)}
                  aria-label={t('common.delete')}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </TableCell>
            </TableRow>
          );
        })}
        {employees.length === 0 && (
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
