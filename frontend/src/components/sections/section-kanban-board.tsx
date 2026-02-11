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
import { useToast } from '@/lib/hooks/use-toast';
import type { Child } from '@/lib/api/types';
import { Skeleton } from '@/components/ui/skeleton';
import { SectionColumn } from './section-column';
import { ChildCard } from './child-card';

interface SectionKanbanBoardProps {
  orgId: number;
}

export function SectionKanbanBoard({ orgId }: SectionKanbanBoardProps) {
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const [activeChild, setActiveChild] = useState<Child | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 8 },
    })
  );

  const { data: sectionsData, isLoading: sectionsLoading } = useQuery({
    queryKey: ['sections', orgId],
    queryFn: () => apiClient.getSections(orgId, { limit: 100 }),
    enabled: !!orgId,
  });

  const { data: children, isLoading: childrenLoading } = useQuery({
    queryKey: ['children-all', orgId],
    queryFn: () => apiClient.getChildrenAll(orgId),
    enabled: !!orgId,
  });

  const allSections = useMemo(() => sectionsData?.data ?? [], [sectionsData]);
  const defaultSection = useMemo(() => allSections.find((s) => s.is_default), [allSections]);
  const sections = useMemo(() => allSections.filter((s) => !s.is_default), [allSections]);
  const isLoading = sectionsLoading || childrenLoading;

  const childrenBySection = useMemo(() => {
    const map = new Map<string, Child[]>();
    map.set('unassigned', []);
    for (const section of sections) {
      map.set(String(section.id), []);
    }
    for (const child of children ?? []) {
      const sectionId = child.section_id ?? null;
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

  const moveMutation = useMutation({
    mutationFn: ({ childId, sectionId }: { childId: number; sectionId: number | null }) =>
      apiClient.updateChild(orgId, childId, { section_id: sectionId }),
    onMutate: async ({ childId, sectionId }) => {
      await queryClient.cancelQueries({ queryKey: ['children-all', orgId] });
      const previous = queryClient.getQueryData<Child[]>(['children-all', orgId]);
      queryClient.setQueryData<Child[]>(['children-all', orgId], (old) =>
        old?.map((c) => (c.id === childId ? { ...c, section_id: sectionId } : c))
      );
      return { previous };
    },
    onSuccess: () => {
      toast({ title: t('sections.movedSuccess') });
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(['children-all', orgId], context.previous);
      }
      toast({ title: t('sections.movedFailed'), variant: 'destructive' });
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['children-all', orgId] });
      queryClient.invalidateQueries({ queryKey: ['children', orgId] });
    },
  });

  function handleDragStart(event: DragStartEvent) {
    const child = event.active.data.current?.child as Child | undefined;
    if (child) setActiveChild(child);
  }

  function handleDragEnd(event: DragEndEvent) {
    setActiveChild(null);
    const { active, over } = event;
    if (!over) return;

    const child = active.data.current?.child as Child | undefined;
    if (!child) return;

    const targetColumnId = String(over.id);
    const newSectionId =
      targetColumnId === 'unassigned' ? (defaultSection?.id ?? null) : Number(targetColumnId);
    const currentSectionId = child.section_id ?? null;

    if (newSectionId === currentSectionId) return;

    moveMutation.mutate({ childId: child.id, sectionId: newSectionId });
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
          />
          {sections.map((section) => (
            <SectionColumn
              key={section.id}
              id={String(section.id)}
              title={section.name}
              items={childrenBySection.get(String(section.id)) ?? []}
              isDefault={section.is_default}
            />
          ))}
        </div>
        <DragOverlay>{activeChild ? <ChildCard child={activeChild} /> : null}</DragOverlay>
      </DndContext>
    </div>
  );
}
