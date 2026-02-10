'use client';

import { useMemo, useState } from 'react';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useQuery } from '@tanstack/react-query';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { apiClient } from '@/lib/api/client';
import type { Section, SectionCreateRequest, SectionUpdateRequest } from '@/lib/api/types';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useCrudMutations } from '@/lib/hooks/use-crud-mutations';
import { useCrudDialogs } from '@/lib/hooks/use-crud-dialogs';
import { CrudPageHeader, ResourceTable, DeleteConfirmDialog, Column } from '@/components/crud';
import { Pagination } from '@/components/ui/pagination';
import { SectionKanbanBoard } from '@/components/sections/section-kanban-board';

const sectionSchema = z.object({
  name: z.string().min(1).max(255),
});

type SectionFormData = z.infer<typeof sectionSchema>;

const defaultValues: SectionFormData = {
  name: '',
};

export default function SectionsPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations();
  const [page, setPage] = useState(1);

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<SectionFormData>({
    resolver: zodResolver(sectionSchema),
    defaultValues,
  });

  const { data: paginatedData, isLoading } = useQuery({
    queryKey: ['sections', orgId, page],
    queryFn: () => apiClient.getSections(orgId, { page }),
    enabled: !!orgId,
  });

  const sections = paginatedData?.data;

  const dialogs = useCrudDialogs<Section, SectionFormData>({
    reset,
    itemToFormData: (section) => ({ name: section.name }),
    defaultValues,
  });

  const mutations = useCrudMutations<Section, SectionCreateRequest, SectionUpdateRequest>({
    resourceName: 'sections',
    queryKey: ['sections', orgId],
    createFn: (data) => apiClient.createSection(orgId, data),
    updateFn: (id, data) => apiClient.updateSection(orgId, id, data),
    deleteFn: (id) => apiClient.deleteSection(orgId, id),
    onSuccess: dialogs.closeDialog,
    onDeleteSuccess: dialogs.closeDeleteDialog,
  });

  const onSubmit = (data: SectionFormData) => {
    if (dialogs.editingItem) {
      mutations.updateMutation.mutate({ id: dialogs.editingItem.id, data });
    } else {
      mutations.createMutation.mutate(data);
    }
  };

  const columns = useMemo<Column<Section>[]>(
    () => [
      { key: 'id', header: 'common.id', render: (section) => section.id },
      {
        key: 'name',
        header: 'common.name',
        render: (section) => (
          <div className="flex items-center gap-2">
            <span className="font-medium">{section.name}</span>
            {section.is_default && (
              <Badge variant="secondary" className="text-xs">
                {t('sections.defaultSection')}
              </Badge>
            )}
          </div>
        ),
      },
    ],
    [t]
  );

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold tracking-tight">{t('sections.title')}</h1>

      <Tabs defaultValue="board">
        <TabsList>
          <TabsTrigger value="board">{t('sections.board')}</TabsTrigger>
          <TabsTrigger value="manage">{t('sections.manage')}</TabsTrigger>
        </TabsList>

        <TabsContent value="board" className="mt-4">
          <SectionKanbanBoard orgId={orgId} />
        </TabsContent>

        <TabsContent value="manage" className="mt-4 space-y-6">
          <CrudPageHeader
            title="sections.manage"
            onNew={dialogs.handleCreate}
            newButtonText="sections.newSection"
          />

          <Card>
            <CardHeader>
              <CardTitle>{t('sections.title')}</CardTitle>
            </CardHeader>
            <CardContent>
              <ResourceTable
                items={sections}
                columns={columns}
                getItemKey={(section) => section.id}
                isLoading={isLoading}
                onEdit={dialogs.handleEdit}
                onDelete={dialogs.handleDelete}
              />
              {paginatedData && (
                <Pagination
                  page={paginatedData.page}
                  totalPages={paginatedData.total_pages}
                  total={paginatedData.total}
                  limit={paginatedData.limit}
                  onPageChange={setPage}
                  isLoading={isLoading}
                />
              )}
            </CardContent>
          </Card>

          <Dialog open={dialogs.isDialogOpen} onOpenChange={dialogs.setIsDialogOpen}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>
                  {dialogs.isEditing ? t('sections.edit') : t('sections.create')}
                </DialogTitle>
              </DialogHeader>
              <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="name">{t('common.name')}</Label>
                  <Input id="name" {...register('name')} />
                  {errors.name && (
                    <p className="text-sm text-destructive">{t('validation.nameRequired')}</p>
                  )}
                </div>

                <DialogFooter>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => dialogs.setIsDialogOpen(false)}
                  >
                    {t('common.cancel')}
                  </Button>
                  <Button type="submit" disabled={mutations.isMutating}>
                    {t('common.save')}
                  </Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>

          <DeleteConfirmDialog
            open={dialogs.isDeleteDialogOpen}
            onOpenChange={dialogs.setIsDeleteDialogOpen}
            onConfirm={() =>
              dialogs.deletingItem && mutations.deleteMutation.mutate(dialogs.deletingItem.id)
            }
            isLoading={mutations.deleteMutation.isPending}
            resourceName="sections"
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}
