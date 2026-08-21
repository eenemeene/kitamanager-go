'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { GenderSelect } from '@/components/ui/gender-select';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

import { apiClient } from '@/lib/api/client';
import { queryKeys } from '@/lib/api/queryKeys';
import { useToast } from '@/lib/hooks/use-toast';
import { showErrorToast } from '@/lib/utils/show-error-toast';
import { formatDateForApi } from '@/lib/utils/formatting';
import type { Gender, UnmatchedBillChild } from '@/lib/api/types';

export interface AddChildFromBillDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orgId: number;
  /** Bill row driving the prefill. Null when the dialog isn't tied to a row. */
  billChild: UnmatchedBillChild | null;
}

/**
 * AddChildFromBillDialog turns an unmatched ISBJ bill row into a real
 * KitaManager child + initial contract + voucher assignment in one
 * submit. Pre-fills name, birthdate (first day of the first billing
 * month), and contract start (same date) from the bill data so the
 * user only needs to confirm gender + section.
 *
 * The birthdate is a placeholder — the bill format only carries
 * MM.YY. The warning banner makes that explicit so the user knows to
 * adjust against the Kita-Gutschein paperwork.
 */
export function AddChildFromBillDialog({
  open,
  onOpenChange,
  orgId,
  billChild,
}: AddChildFromBillDialogProps) {
  const t = useTranslations();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  // Birthdate placeholder = first day of the bill's first-seen month.
  // FirstSeenBillFrom is already the 1st of the month in the API response,
  // so we can use it directly for both fields.
  //
  // State initializers read directly from the billChild prop. The
  // parent passes a stable `key` (the voucher_number) so this component
  // remounts whenever the user picks a different bill row — the
  // initializers fire fresh, no useEffect-resetting needed (which the
  // react-hooks/set-state-in-effect rule rightly forbids).
  const placeholderBirthdate = billChild?.first_seen_bill_from ?? '';

  const [firstName, setFirstName] = useState(billChild?.first_name ?? '');
  const [lastName, setLastName] = useState(billChild?.last_name ?? '');
  const [gender, setGender] = useState<Gender>('male');
  const [birthdate, setBirthdate] = useState(placeholderBirthdate);
  const [contractFrom, setContractFrom] = useState(placeholderBirthdate);
  const [sectionId, setSectionId] = useState<number>(0);

  const { data: sectionsResp } = useQuery({
    queryKey: queryKeys.sections.list(orgId),
    queryFn: () => apiClient.getSections(orgId),
    enabled: open,
  });
  const sections = sectionsResp?.data ?? [];

  // Three-call chain: createChild → createChildContract → assignChildVoucher.
  // If voucher assign fails (e.g. cross-org 409), the child + contract
  // remain — the user can fix the voucher via the Vouchers dialog. We
  // surface the specific failure so they know what step blocked.
  const submitMutation = useMutation({
    mutationFn: async () => {
      if (!billChild) throw new Error('no bill child');
      const newChild = await apiClient.createChild(orgId, {
        first_name: firstName,
        last_name: lastName,
        gender,
        birthdate,
      });
      // `from` is a Go `time.Time`, which only unmarshals RFC3339 -- and a
      // `<input type="date">` yields a bare "YYYY-MM-DD". Sending it raw 400'd
      // every time: the child above was created, the contract was refused, and
      // the user was left with an error toast and a child with no contract and
      // no voucher. `formatDateForApi` is the conversion every other contract
      // call site already uses.
      const from = formatDateForApi(contractFrom);
      if (!from) throw new Error('invalid contract start date');
      await apiClient.createChildContract(orgId, newChild.id, {
        from,
        section_id: sectionId,
        properties: {},
      });
      await apiClient.assignChildVoucher(orgId, newChild.id, billChild.voucher_number);
      return newChild;
    },
    onSuccess: () => {
      toast({ title: t('addChildFromBill.success') });
      queryClient.invalidateQueries({
        queryKey: queryKeys.governmentFundingBillPeriods.unmatchedChildren(orgId),
      });
      queryClient.invalidateQueries({ queryKey: queryKeys.children.all(orgId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.children.withoutVouchers(orgId) });
      onOpenChange(false);
    },
    onError: (error) => {
      showErrorToast(t('common.error'), error, t('addChildFromBill.error'));
    },
  });

  const canSubmit =
    firstName.trim() !== '' &&
    lastName.trim() !== '' &&
    birthdate !== '' &&
    contractFrom !== '' &&
    sectionId > 0 &&
    !submitMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('addChildFromBill.title')}</DialogTitle>
          {billChild && (
            <DialogDescription>
              {t('addChildFromBill.subtitle', { voucher: billChild.voucher_number })}
            </DialogDescription>
          )}
        </DialogHeader>

        <div className="space-y-4">
          <Alert className="border-warning/50 text-warning [&>svg]:text-warning">
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription>{t('addChildFromBill.birthdateWarning')}</AlertDescription>
          </Alert>

          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="afb_first_name">{t('children.firstName')}</Label>
              <Input
                id="afb_first_name"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="afb_last_name">{t('children.lastName')}</Label>
              <Input
                id="afb_last_name"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="afb_gender">{t('gender.label')}</Label>
            <GenderSelect value={gender} onValueChange={setGender} />
          </div>

          <div className="space-y-2">
            <Label htmlFor="afb_birthdate">{t('children.birthdate')}</Label>
            <Input
              id="afb_birthdate"
              type="date"
              value={birthdate}
              onChange={(e) => setBirthdate(e.target.value)}
            />
            <p className="text-muted-foreground text-xs">
              {t('addChildFromBill.birthdateHint', { billDate: billChild?.bill_birth_date ?? '' })}
            </p>
          </div>

          <div className="space-y-4 border-t pt-4">
            <h4 className="text-sm font-medium">{t('children.initialContract')}</h4>

            <div className="space-y-2">
              <Label htmlFor="afb_contract_from">{t('contracts.startDate')}</Label>
              <Input
                id="afb_contract_from"
                type="date"
                value={contractFrom}
                onChange={(e) => setContractFrom(e.target.value)}
              />
              <p className="text-muted-foreground text-xs">
                {t('addChildFromBill.contractFromHint')}
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="afb_section">{t('sections.title')}</Label>
              <Select
                value={sectionId > 0 ? sectionId.toString() : ''}
                onValueChange={(v) => setSectionId(Number(v))}
              >
                <SelectTrigger id="afb_section" aria-label={t('sections.title')}>
                  <SelectValue placeholder={t('sections.selectSection')} />
                </SelectTrigger>
                <SelectContent>
                  {sections.map((s) => (
                    <SelectItem key={s.id} value={s.id.toString()}>
                      {s.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button disabled={!canSubmit} onClick={() => submitMutation.mutate()}>
            {t('addChildFromBill.createAction')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
