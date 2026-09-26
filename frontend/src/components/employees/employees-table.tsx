'use client';

import { useTranslations } from 'next-intl';
import { Pencil, Trash2, FileText, History } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { RowActionsMenu } from '@/components/crud/row-actions-menu';
import { HeaderWithTooltip } from '@/components/ui/header-with-tooltip';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { TooltipProvider } from '@/components/ui/tooltip';
import type { Employee, PayPlanDetail } from '@/lib/api/types';
import { calculateAge } from '@/lib/utils/formatting';
import { getCurrentContract, toUTCDate } from '@/lib/utils/contracts';
import { calculateMonthlySalary } from '@/lib/utils/salary';
import { calculateYearsOfService } from '@/lib/utils/step-promotions';
import { useFormatters } from '@/hooks/use-formatters';

export interface EmployeesTableProps {
  employees: Employee[];
  payPlanMap: Map<number, PayPlanDetail>;
  /**
   * The date the roster is filtered to, "YYYY-MM-DD".
   *
   * Staff category, grade/step, weekly hours, salary and years of service all
   * come from a contract, and which contract that is depends on the date. The
   * page sends this to the API as `active_on`; resolving the columns against
   * today instead would price a March roster with September's Entgelttabelle.
   */
  asOf: string;
  onViewHistory: (employee: Employee) => void;
  onAddContract: (employee: Employee) => void;
  onEdit: (employee: Employee) => void;
  onDelete: (employee: Employee) => void;
}

export function EmployeesTable({
  employees,
  payPlanMap,
  asOf,
  onViewHistory,
  onAddContract,
  onEdit,
  onDelete,
}: EmployeesTableProps) {
  const t = useTranslations();

  const fmt = useFormatters();
  return (
    <TooltipProvider>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('common.name')}</TableHead>
            <TableHead className="hidden md:table-cell">{t('gender.label')}</TableHead>
            <TableHead className="hidden md:table-cell">{t('employees.birthdate')}</TableHead>
            <TableHead className="hidden md:table-cell">{t('employees.age')}</TableHead>
            <TableHead>
              <HeaderWithTooltip
                label={t('employees.staffCategory.label')}
                tooltip={t('employees.staffCategoryTooltip')}
              />
            </TableHead>
            <TableHead className="hidden lg:table-cell">
              <HeaderWithTooltip
                label={t('employees.grade')}
                tooltip={t('employees.gradeTooltip')}
              />
            </TableHead>
            <TableHead className="hidden lg:table-cell">
              <HeaderWithTooltip
                label={t('employees.weeklyHours')}
                tooltip={t('employees.weeklyHoursTooltip')}
              />
            </TableHead>
            <TableHead className="hidden lg:table-cell">
              <HeaderWithTooltip
                label={t('employees.salary')}
                tooltip={t('employees.salaryTooltip')}
              />
            </TableHead>
            <TableHead className="hidden lg:table-cell">
              <HeaderWithTooltip
                label={t('employees.yearsOfService')}
                tooltip={t('employees.yearsOfServiceTooltip')}
              />
            </TableHead>
            <TableHead className="text-right">{t('common.actions')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {employees.map((employee) => {
            const currentContract = getCurrentContract(employee.contracts, asOf);
            const payPlanForSalary = currentContract?.payplan_id
              ? payPlanMap.get(currentContract.payplan_id)
              : undefined;
            const salary =
              currentContract && payPlanForSalary
                ? calculateMonthlySalary(currentContract, payPlanForSalary, asOf)
                : null;
            // The UTC-midnight frame the helper documents, which is also what
            // `toUTCDate` answers in — see its `asOf` parameter.
            const yearsOfService = employee.contracts?.length
              ? calculateYearsOfService(employee.contracts, new Date(toUTCDate(asOf)))
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
                  {fmt.date(employee.birthdate)}
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
                  {salary !== null ? fmt.currency(salary) : '-'}
                </TableCell>
                <TableCell className="hidden lg:table-cell">
                  {yearsOfService !== null
                    ? fmt.number(yearsOfService, {
                        minimumFractionDigits: 1,
                        maximumFractionDigits: 1,
                      })
                    : '-'}
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex flex-nowrap items-center justify-end gap-0.5">
                    {/* See children-table: contract history and Add contract are
                        hidden below sm and have no other route into them. */}
                    <RowActionsMenu
                      className="sm:hidden"
                      label={t('common.actions')}
                      actions={[
                        {
                          key: 'history',
                          label: t('employees.contractHistory'),
                          icon: History,
                          onSelect: () => onViewHistory(employee),
                        },
                        {
                          key: 'add-contract',
                          label: t('employees.addContract'),
                          icon: FileText,
                          onSelect: () => onAddContract(employee),
                        },
                      ]}
                    />
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onViewHistory(employee)}
                      title={t('employees.contractHistory')}
                      aria-label={t('employees.contractHistory')}
                      className="hidden sm:inline-flex"
                    >
                      <History className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onAddContract(employee)}
                      title={t('employees.addContract')}
                      aria-label={t('employees.addContract')}
                      className="hidden sm:inline-flex"
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
                  </div>
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
    </TooltipProvider>
  );
}
