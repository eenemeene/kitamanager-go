'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { Plus } from 'lucide-react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { apiClient } from '@/lib/api/client';
import { formatMonthRange } from '@/lib/utils/formatting';
import type { Section, SectionCreateRequest, SectionUpdateRequest } from '@/lib/api/types';
import { useCrudPage } from '@/lib/hooks/use-crud-page';
import { ResourceTable, DeleteConfirmDialog, CrudFormDialog, Column } from '@/components/crud';
import { Pagination } from '@/components/ui/pagination';
import { SectionKanbanBoard } from '@/components/sections/section-kanban-board';
import { sectionSchema, type SectionFormData } from '@/lib/schemas';

const defaultValues: SectionFormData = {
  name: '',
  min_age_months: null,
  max_age_months: null,
};

export default function SectionsPage() {
  const t = useTranslations();
  const crud = useCrudPage<Section, SectionFormData, SectionCreateRequest, SectionUpdateRequest>({
    resourceName: 'sections',
    schema: sectionSchema,
    defaultValues,
    itemToFormData: (section) => ({
      name: section.name,
      min_age_months: section.min_age_months ?? null,
      max_age_months: section.max_age_months ?? null,
    }),
    listFn: (orgId, params) => apiClient.getSections(orgId, params),
    createFn: (orgId, data) => apiClient.createSection(orgId, data),
    updateFn: (orgId, id, data) => apiClient.updateSection(orgId, id, data),
    deleteFn: (orgId, id) => apiClient.deleteSection(orgId, id),
  });

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
      {
        key: 'ageRange',
        header: 'sections.ageRange',
        render: (section) => {
          const range = formatMonthRange(section.min_age_months, section.max_age_months);
          return range ? (
            <span className="text-muted-foreground">
              {range} {t('sections.months')}
            </span>
          ) : (
            <span className="text-muted-foreground">—</span>
          );
        },
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
          <SectionKanbanBoard orgId={crud.orgId} />
        </TabsContent>

        <TabsContent value="manage" className="mt-4 space-y-6">
          <div className="flex justify-end">
            <Button onClick={crud.dialogs.handleCreate}>
              <Plus className="mr-2 h-4 w-4" />
              {t('sections.newSection')}
            </Button>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>{t('sections.title')}</CardTitle>
            </CardHeader>
            <CardContent>
              <ResourceTable
                items={crud.items}
                columns={columns}
                getItemKey={(section) => section.id}
                isLoading={crud.isLoading}
                onEdit={crud.dialogs.handleEdit}
                onDelete={crud.dialogs.handleDelete}
              />
              {crud.paginatedData && (
                <Pagination
                  page={crud.paginatedData.page}
                  totalPages={crud.paginatedData.total_pages}
                  total={crud.paginatedData.total}
                  limit={crud.paginatedData.limit}
                  onPageChange={crud.setPage}
                  isLoading={crud.isLoading}
                />
              )}
            </CardContent>
          </Card>

          <CrudFormDialog
            open={crud.dialogs.isDialogOpen}
            onOpenChange={crud.dialogs.setIsDialogOpen}
            isEditing={crud.dialogs.isEditing}
            translationPrefix="sections"
            onSubmit={crud.handleSubmit(crud.onSubmit)}
            isSaving={crud.mutations.isMutating}
          >
            <div className="space-y-2">
              <Label htmlFor="name">{t('common.name')}</Label>
              <Input id="name" {...crud.register('name')} />
              {crud.errors.name && (
                <p className="text-sm text-destructive">{t('validation.nameRequired')}</p>
              )}
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="min_age_months">
                  {t('sections.minAgeMonths')}{' '}
                  <span className="text-muted-foreground">({t('sections.optional')})</span>
                </Label>
                <Input
                  id="min_age_months"
                  type="number"
                  min={0}
                  {...crud.register('min_age_months', {
                    setValueAs: (v: unknown) =>
                      v === '' || v === null || v === undefined ? null : Number(v),
                  })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="max_age_months">
                  {t('sections.maxAgeMonths')}{' '}
                  <span className="text-muted-foreground">({t('sections.optional')})</span>
                </Label>
                <Input
                  id="max_age_months"
                  type="number"
                  min={0}
                  {...crud.register('max_age_months', {
                    setValueAs: (v: unknown) =>
                      v === '' || v === null || v === undefined ? null : Number(v),
                  })}
                />
                {crud.errors.max_age_months && (
                  <p className="text-sm text-destructive">{t('sections.ageRangeError')}</p>
                )}
              </div>
            </div>
          </CrudFormDialog>

          <DeleteConfirmDialog
            open={crud.dialogs.isDeleteDialogOpen}
            onOpenChange={crud.dialogs.setIsDeleteDialogOpen}
            onConfirm={() =>
              crud.dialogs.deletingItem &&
              crud.mutations.deleteMutation.mutate(crud.dialogs.deletingItem.id)
            }
            isLoading={crud.mutations.deleteMutation.isPending}
            resourceName="sections"
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}
