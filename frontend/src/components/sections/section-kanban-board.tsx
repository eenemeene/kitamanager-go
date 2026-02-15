'use client';

import { useMemo, useState } from 'react';
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  useSensor,
  useSensors,
  type DragStartEvent,
  type DragEndEvent,
} from '@dnd-kit/core';
import { useQuery } from '@tanstack/react-query';
import { differenceInMonths, parseISO } from 'date-fns';
import { useTranslations } from 'next-intl';
import { GripVertical } from 'lucide-react';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { useToast } from '@/lib/hooks/use-toast';
import { useMoveContractMutation } from '@/lib/hooks/use-move-contract-mutation';
import type { Child, Employee } from '@/lib/api/types';
import { getActiveContract } from '@/lib/utils/contracts';
import { Skeleton } from '@/components/ui/skeleton';
import { SectionColumn } from './section-column';
import { ChildCard } from './child-card';
import { EmployeeCard } from './employee-card';

interface SectionKanbanBoardProps {
  orgId: number;
}

type ActiveItem = { type: 'child'; item: Child } | { type: 'employee'; item: Employee };

/** Get the section_id from the active contract. */
function getContractSectionId(
  contracts?: { from: string; to?: string | null; section_id: number }[]
): number | null {
  const active = getActiveContract(contracts);
  return active?.section_id ?? null;
}

export function SectionKanbanBoard({ orgId }: SectionKanbanBoardProps) {
  const t = useTranslations();
  const { toast } = useToast();
  const [activeItem, setActiveItem] = useState<ActiveItem | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 8 },
    })
  );

  const { data: sectionsData, isLoading: sectionsLoading } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId, { limit: 100 }),
    enabled: !!orgId,
  });

  const { data: children, isLoading: childrenLoading } = useQuery({
    queryKey: queryKeys.children.allUnpaginated(orgId),
    queryFn: () => apiClient.getChildrenAll(orgId),
    enabled: !!orgId,
  });

  const { data: allEmployees, isLoading: employeesLoading } = useQuery({
    queryKey: queryKeys.employees.allUnpaginated(orgId),
    queryFn: () => apiClient.getEmployeesAll(orgId),
    enabled: !!orgId,
  });

  const pedagogicalEmployees = useMemo(() => {
    if (!allEmployees) return [];
    // Backend already filters by active_on — every employee here has an active contract.
    return allEmployees.filter((e) => {
      const c = getActiveContract(e.contracts);
      return c && c.staff_category !== 'non_pedagogical';
    });
  }, [allEmployees]);

  const allSections = useMemo(() => sectionsData?.data ?? [], [sectionsData]);
  const sections = allSections;
  const isLoading = sectionsLoading || childrenLoading || employeesLoading;

  const childrenBySection = useMemo(() => {
    const map = new Map<string, Child[]>();
    for (const section of sections) {
      map.set(String(section.id), []);
    }
    for (const child of children ?? []) {
      const sectionId = getContractSectionId(child.contracts);
      if (sectionId) {
        const key = String(sectionId);
        const list = map.get(key);
        if (list) {
          list.push(child);
        }
      }
    }
    return map;
  }, [sections, children]);

  const employeesBySection = useMemo(() => {
    const map = new Map<string, Employee[]>();
    for (const section of sections) {
      map.set(String(section.id), []);
    }
    for (const emp of pedagogicalEmployees) {
      const sectionId = getContractSectionId(emp.contracts);
      if (sectionId) {
        const key = String(sectionId);
        const list = map.get(key);
        if (list) {
          list.push(emp);
        }
      }
    }
    return map;
  }, [sections, pedagogicalEmployees]);

  const moveChildMutation = useMoveContractMutation<Child>({
    orgId,
    updateFn: (childId, contractId, sectionId) =>
      apiClient.updateChildContract(orgId, childId, contractId, { section_id: sectionId }),
    allUnpaginatedKey: queryKeys.children.allUnpaginated(orgId),
    invalidateKeys: (childId) => [
      queryKeys.children.all(orgId),
      queryKeys.children.contracts(orgId, childId),
      queryKeys.children.detail(orgId, childId),
    ],
    successMessage: 'sections.movedSuccess',
    errorMessage: 'sections.movedFailed',
  });

  const moveEmployeeMutation = useMoveContractMutation<Employee>({
    orgId,
    updateFn: (employeeId, contractId, sectionId) =>
      apiClient.updateEmployeeContract(orgId, employeeId, contractId, { section_id: sectionId }),
    allUnpaginatedKey: queryKeys.employees.allUnpaginated(orgId),
    invalidateKeys: (employeeId) => [
      queryKeys.employees.all(orgId),
      queryKeys.employees.contracts(orgId, employeeId),
      queryKeys.employees.detail(orgId, employeeId),
    ],
    successMessage: 'sections.employeeMovedSuccess',
    errorMessage: 'sections.employeeMovedFailed',
  });

  function handleDragStart(event: DragStartEvent) {
    const data = event.active.data.current;
    if (data?.type === 'employee') {
      setActiveItem({ type: 'employee', item: data.employee as Employee });
    } else if (data?.child) {
      setActiveItem({ type: 'child', item: data.child as Child });
    }
  }

  function handleDragEnd(event: DragEndEvent) {
    const currentItem = activeItem;
    setActiveItem(null);
    const { over } = event;
    if (!over || !currentItem) return;

    const targetColumnId = String(over.id);
    const newSectionId = Number(targetColumnId);

    if (currentItem.type === 'child') {
      const child = currentItem.item;
      const activeContract = getActiveContract(child.contracts);
      if (!activeContract) return; // no active contract to update
      if (newSectionId === activeContract.section_id) return;

      // Warn if child's age is outside target section's age range
      const targetSection = allSections.find((s) => s.id === newSectionId);
      if (targetSection && child.birthdate) {
        const ageMonths = differenceInMonths(new Date(), parseISO(child.birthdate));
        const minAge = targetSection.min_age_months;
        const maxAge = targetSection.max_age_months;
        const outsideRange =
          (minAge != null && ageMonths < minAge) || (maxAge != null && ageMonths >= maxAge);
        if (outsideRange) {
          toast({
            title: t('sections.ageMismatchWarning'),
            description: t('sections.ageMismatchDescription'),
            variant: 'destructive',
          });
        }
      }

      moveChildMutation.mutate({
        entityId: child.id,
        contractId: activeContract.id,
        sectionId: newSectionId,
      });
    } else {
      const employee = currentItem.item;
      const activeContract = getActiveContract(employee.contracts);
      if (!activeContract) return; // no active contract to update
      if (newSectionId === activeContract.section_id) return;
      moveEmployeeMutation.mutate({
        entityId: employee.id,
        contractId: activeContract.id,
        sectionId: newSectionId,
      });
    }
  }

  if (isLoading) {
    return (
      <div className="flex gap-4 overflow-x-auto p-4">
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-96 w-72 shrink-0" />
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <p className="flex items-center gap-2 text-sm text-muted-foreground">
        <GripVertical className="h-4 w-4" />
        {t('sections.dragHint')}
      </p>
      <DndContext sensors={sensors} onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
        <div className="flex gap-4 overflow-x-auto pb-4">
          {sections.map((section) => (
            <SectionColumn
              key={section.id}
              id={String(section.id)}
              title={section.name}
              items={childrenBySection.get(String(section.id)) ?? []}
              employees={employeesBySection.get(String(section.id)) ?? []}
              isDefault={section.is_default}
              minAgeMonths={section.min_age_months}
              maxAgeMonths={section.max_age_months}
            />
          ))}
        </div>
        <DragOverlay>
          {activeItem?.type === 'child' ? (
            <ChildCard child={activeItem.item} />
          ) : activeItem?.type === 'employee' ? (
            <EmployeeCard employee={activeItem.item} />
          ) : null}
        </DragOverlay>
      </DndContext>
    </div>
  );
}
