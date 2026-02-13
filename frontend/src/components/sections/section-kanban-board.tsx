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
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { GripVertical } from 'lucide-react';
import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { useToast } from '@/lib/hooks/use-toast';
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

/** Get the section_id from the active contract, or null if none. */
function getContractSectionId(contracts?: { from: string; to?: string | null; section_id?: number | null }[]): number | null {
  const active = getActiveContract(contracts);
  return active?.section_id ?? null;
}

export function SectionKanbanBoard({ orgId }: SectionKanbanBoardProps) {
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();
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
  const defaultSection = useMemo(() => allSections.find((s) => s.is_default), [allSections]);
  const sections = useMemo(() => allSections.filter((s) => !s.is_default), [allSections]);
  const isLoading = sectionsLoading || childrenLoading || employeesLoading;

  const childrenBySection = useMemo(() => {
    const map = new Map<string, Child[]>();
    map.set('unassigned', []);
    for (const section of sections) {
      map.set(String(section.id), []);
    }
    for (const child of children ?? []) {
      const sectionId = getContractSectionId(child.contracts);
      const isUnassigned = !sectionId || (defaultSection && sectionId === defaultSection.id);
      const key = isUnassigned ? 'unassigned' : String(sectionId);
      const list = map.get(key);
      if (list) {
        list.push(child);
      } else {
        map.get('unassigned')!.push(child);
      }
    }
    return map;
  }, [sections, defaultSection, children]);

  const employeesBySection = useMemo(() => {
    const map = new Map<string, Employee[]>();
    map.set('unassigned', []);
    for (const section of sections) {
      map.set(String(section.id), []);
    }
    for (const emp of pedagogicalEmployees) {
      const sectionId = getContractSectionId(emp.contracts);
      const isUnassigned = !sectionId || (defaultSection && sectionId === defaultSection.id);
      const key = isUnassigned ? 'unassigned' : String(sectionId);
      const list = map.get(key);
      if (list) {
        list.push(emp);
      } else {
        map.get('unassigned')!.push(emp);
      }
    }
    return map;
  }, [sections, defaultSection, pedagogicalEmployees]);

  const moveChildMutation = useMutation({
    mutationFn: ({ childId, contractId, sectionId }: { childId: number; contractId: number; sectionId: number | null }) =>
      apiClient.updateChildContract(orgId, childId, contractId, { section_id: sectionId }),
    onMutate: async ({ childId, contractId, sectionId }) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.children.allUnpaginated(orgId) });
      const previous = queryClient.getQueryData<Child[]>(queryKeys.children.allUnpaginated(orgId));
      queryClient.setQueryData<Child[]>(queryKeys.children.allUnpaginated(orgId), (old) =>
        old?.map((c) =>
          c.id === childId
            ? {
                ...c,
                contracts: c.contracts?.map((ct) =>
                  ct.id === contractId ? { ...ct, section_id: sectionId } : ct
                ),
              }
            : c
        )
      );
      return { previous };
    },
    onSuccess: () => {
      toast({ title: t('sections.movedSuccess') });
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(queryKeys.children.allUnpaginated(orgId), context.previous);
      }
      toast({ title: t('sections.movedFailed'), variant: 'destructive' });
    },
    onSettled: (_data, _error, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.children.allUnpaginated(orgId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.children.all(orgId) });
      queryClient.invalidateQueries({
        queryKey: queryKeys.children.contracts(orgId, variables.childId),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.children.detail(orgId, variables.childId),
      });
    },
  });

  const moveEmployeeMutation = useMutation({
    mutationFn: ({ employeeId, contractId, sectionId }: { employeeId: number; contractId: number; sectionId: number | null }) =>
      apiClient.updateEmployeeContract(orgId, employeeId, contractId, { section_id: sectionId }),
    onMutate: async ({ employeeId, contractId, sectionId }) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.employees.allUnpaginated(orgId) });
      const previous = queryClient.getQueryData<Employee[]>(
        queryKeys.employees.allUnpaginated(orgId)
      );
      queryClient.setQueryData<Employee[]>(queryKeys.employees.allUnpaginated(orgId), (old) =>
        old?.map((e) =>
          e.id === employeeId
            ? {
                ...e,
                contracts: e.contracts?.map((ct) =>
                  ct.id === contractId ? { ...ct, section_id: sectionId } : ct
                ),
              }
            : e
        )
      );
      return { previous };
    },
    onSuccess: () => {
      toast({ title: t('sections.employeeMovedSuccess') });
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(queryKeys.employees.allUnpaginated(orgId), context.previous);
      }
      toast({ title: t('sections.employeeMovedFailed'), variant: 'destructive' });
    },
    onSettled: (_data, _error, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.employees.allUnpaginated(orgId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.employees.all(orgId) });
      queryClient.invalidateQueries({
        queryKey: queryKeys.employees.contracts(orgId, variables.employeeId),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.employees.detail(orgId, variables.employeeId),
      });
    },
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
    const newSectionId =
      targetColumnId === 'unassigned' ? (defaultSection?.id ?? null) : Number(targetColumnId);

    if (currentItem.type === 'child') {
      const child = currentItem.item;
      const activeContract = getActiveContract(child.contracts);
      if (!activeContract) return; // no active contract to update
      const currentSectionId = activeContract.section_id ?? null;
      if (newSectionId === currentSectionId) return;

      // Warn if child's age is outside target section's age range
      const targetSection = allSections.find((s) => s.id === newSectionId);
      if (targetSection && child.birthdate) {
        const ageMonths = Math.floor(
          (Date.now() - new Date(child.birthdate).getTime()) / (1000 * 60 * 60 * 24 * 30.44)
        );
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

      moveChildMutation.mutate({ childId: child.id, contractId: activeContract.id, sectionId: newSectionId });
    } else {
      const employee = currentItem.item;
      const activeContract = getActiveContract(employee.contracts);
      if (!activeContract) return; // no active contract to update
      const currentSectionId = activeContract.section_id ?? null;
      if (newSectionId === currentSectionId) return;
      moveEmployeeMutation.mutate({ employeeId: employee.id, contractId: activeContract.id, sectionId: newSectionId });
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
          <SectionColumn
            id="unassigned"
            title={t('sections.unassigned')}
            items={childrenBySection.get('unassigned') ?? []}
            employees={employeesBySection.get('unassigned') ?? []}
          />
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
