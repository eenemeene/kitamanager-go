'use client';

import { useMemo } from 'react';
import { Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { GovernmentFundingPeriod, GovernmentFundingProperty } from '@/lib/api/types';
import { formatAgeRange, formatCurrency, formatFte } from '@/lib/utils/formatting';

interface PropertiesGroupedByKeyProps {
  period: GovernmentFundingPeriod;
  onDeleteProperty: (property: GovernmentFundingProperty) => void;
  t: (key: string) => string;
}

export function PropertiesGroupedByKey({
  period,
  onDeleteProperty,
  t,
}: PropertiesGroupedByKeyProps) {
  // Group by key, then by value within each key
  const groups = useMemo(() => {
    const keyMap = new Map<string, Map<string, GovernmentFundingProperty[]>>();
    for (const prop of period.properties ?? []) {
      let valueMap = keyMap.get(prop.key);
      if (!valueMap) {
        valueMap = new Map<string, GovernmentFundingProperty[]>();
        keyMap.set(prop.key, valueMap);
      }
      const list = valueMap.get(prop.value);
      if (list) {
        list.push(prop);
      } else {
        valueMap.set(prop.value, [prop]);
      }
    }
    return Array.from(keyMap.entries()).map(
      ([key, valueMap]) =>
        [key, Array.from(valueMap.entries())] as [string, [string, GovernmentFundingProperty[]][]]
    );
  }, [period.properties]);

  if (!period.properties?.length) {
    return (
      <p className="text-muted-foreground py-4 text-center">
        {t('governmentFundings.noPropertiesDefined')}
      </p>
    );
  }

  return (
    <div className="space-y-6">
      {groups.map(([key, valueGroups]) => (
        <div key={key}>
          <h4 className="text-sm font-semibold">{key}</h4>
          <div className="mt-2 space-y-3 pl-4">
            {valueGroups.map(([value, properties]) => (
              <div key={value}>
                <p className="text-muted-foreground mb-1 text-sm">
                  {properties[0]?.label || value}
                </p>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('governmentFundings.ageRange')}</TableHead>
                      <TableHead>{t('governmentFundings.payment')}</TableHead>
                      <TableHead>{t('governmentFundings.requirementFte')}</TableHead>
                      <TableHead className="text-right">{t('common.actions')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {properties.map((property) => (
                      <TableRow key={property.id}>
                        <TableCell>
                          {formatAgeRange(property.min_age, property.max_age, t('common.years'))}
                        </TableCell>
                        <TableCell>{formatCurrency(property.payment)}</TableCell>
                        <TableCell>{formatFte(property.requirement)}</TableCell>
                        <TableCell className="text-right">
                          <Button
                            size="icon"
                            variant="ghost"
                            onClick={() => onDeleteProperty(property)}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}
