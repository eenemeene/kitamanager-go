'use client';

import { useState, useRef } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Upload, Trash2, Eye } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { apiClient, getErrorMessage } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import type { GovernmentFundingBillPeriodListItem } from '@/lib/api/types';
import { useToast } from '@/lib/hooks/use-toast';
import { formatCurrency } from '@/lib/utils/formatting';

function BillComparisonCell({ orgId, billId }: { orgId: number; billId: number }) {
  const { data: comparison, isLoading } = useQuery({
    queryKey: queryKeys.governmentFundingBillPeriods.compare(orgId, billId),
    queryFn: () => apiClient.compareGovernmentFundingBill(orgId, billId),
    retry: false,
  });

  if (isLoading) {
    return (
      <>
        <TableCell className="hidden md:table-cell">
          <Skeleton className="h-4 w-20" />
        </TableCell>
        <TableCell className="hidden md:table-cell">
          <Skeleton className="h-4 w-20" />
        </TableCell>
        <TableCell className="hidden md:table-cell">
          <Skeleton className="h-4 w-20" />
        </TableCell>
      </>
    );
  }

  if (!comparison) {
    return (
      <>
        <TableCell className="text-muted-foreground hidden md:table-cell">&mdash;</TableCell>
        <TableCell className="text-muted-foreground hidden md:table-cell">&mdash;</TableCell>
        <TableCell className="text-muted-foreground hidden md:table-cell">&mdash;</TableCell>
      </>
    );
  }

  return (
    <>
      <TableCell className="hidden md:table-cell">
        {comparison.correction_total ? (
          <span className="text-blue-600">{formatCurrency(comparison.correction_total)}</span>
        ) : (
          '\u2014'
        )}
      </TableCell>
      <TableCell className="hidden md:table-cell">
        {formatCurrency(comparison.calculated_total)}
      </TableCell>
      <TableCell className="hidden md:table-cell">
        <span className={comparison.difference < 0 ? 'text-red-600' : 'text-green-600'}>
          {formatCurrency(comparison.difference)}
        </span>
      </TableCell>
    </>
  );
}

export default function GovernmentFundingBillsPage() {
  const params = useParams();
  const orgId = Number(params.orgId);
  const t = useTranslations('governmentFundingBills');
  const tCommon = useTranslations('common');
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<GovernmentFundingBillPeriodListItem | null>(
    null
  );

  const { data: billPeriods, isLoading } = useQuery({
    queryKey: queryKeys.governmentFundingBillPeriods.all(orgId),
    queryFn: () => apiClient.getGovernmentFundingBillPeriods(orgId, { limit: 100 }),
  });

  const uploadMutation = useMutation({
    mutationFn: async (file: File) => {
      const formData = new FormData();
      formData.append('file', file);
      return apiClient.uploadGovernmentFundingBill(orgId, formData);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.governmentFundingBillPeriods.all(orgId),
      });
      setSelectedFile(null);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
      toast({ title: tCommon('success') });
    },
    onError: (error) => {
      toast({
        title: t('uploadError'),
        description: getErrorMessage(error, t('uploadError')),
        variant: 'destructive',
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => apiClient.deleteGovernmentFundingBillPeriod(orgId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.governmentFundingBillPeriods.all(orgId),
      });
      setDeleteTarget(null);
      toast({ title: t('deleteSuccess') });
    },
    onError: (error) => {
      toast({
        title: tCommon('error'),
        description: getErrorMessage(error, tCommon('error')),
        variant: 'destructive',
      });
    },
  });

  const handleUpload = () => {
    if (selectedFile) {
      uploadMutation.mutate(selectedFile);
    }
  };

  const items = billPeriods?.data ?? [];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>

      {/* Upload Card */}
      <Card>
        <CardHeader>
          <CardTitle>{t('selectFile')}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
            <div className="flex-1">
              <Input
                ref={fileInputRef}
                type="file"
                accept=".xlsx"
                onChange={(e) => setSelectedFile(e.target.files?.[0] ?? null)}
              />
            </div>
            <Button onClick={handleUpload} disabled={!selectedFile || uploadMutation.isPending}>
              <Upload className="mr-2 h-4 w-4" />
              {uploadMutation.isPending ? t('uploading') : t('upload')}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Bill Periods List */}
      <Card>
        <CardHeader>
          <CardTitle>{t('title')}</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <p className="text-muted-foreground py-4 text-center">{tCommon('loading')}</p>
          ) : items.length === 0 ? (
            <p className="text-muted-foreground py-4 text-center">{t('noBills')}</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('billingMonth')}</TableHead>
                  <TableHead>{t('facilityName')}</TableHead>
                  <TableHead className="hidden md:table-cell">{t('facilityTotal')}</TableHead>
                  <TableHead className="hidden md:table-cell">{t('correctionTotal')}</TableHead>
                  <TableHead className="hidden md:table-cell">{t('calculatedTotal')}</TableHead>
                  <TableHead className="hidden md:table-cell">{t('difference')}</TableHead>
                  <TableHead className="hidden md:table-cell">{t('fileName')}</TableHead>
                  <TableHead className="hidden md:table-cell">{t('uploadedAt')}</TableHead>
                  <TableHead className="text-right">{tCommon('actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell>
                      {new Date(item.from).toLocaleDateString('de-DE', {
                        month: 'long',
                        year: 'numeric',
                      })}
                    </TableCell>
                    <TableCell>{item.facility_name}</TableCell>
                    <TableCell className="hidden md:table-cell">
                      {formatCurrency(item.facility_total)}
                    </TableCell>
                    <BillComparisonCell orgId={orgId} billId={item.id} />
                    <TableCell className="hidden text-sm md:table-cell">{item.file_name}</TableCell>
                    <TableCell className="hidden text-sm md:table-cell">
                      {new Date(item.created_at).toLocaleDateString('de-DE')}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" size="icon" asChild>
                          <Link
                            href={`/organizations/${orgId}/government-funding-bills/${item.id}`}
                          >
                            <Eye className="h-4 w-4" />
                          </Link>
                        </Button>
                        <Button variant="ghost" size="icon" onClick={() => setDeleteTarget(item)}>
                          <Trash2 className="text-destructive h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={!!deleteTarget} onOpenChange={() => setDeleteTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('deleteBill')}</AlertDialogTitle>
            <AlertDialogDescription>{t('deleteConfirm')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {tCommon('delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
