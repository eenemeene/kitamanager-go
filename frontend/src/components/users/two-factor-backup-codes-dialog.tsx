'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { Copy, Download } from 'lucide-react';

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import { useToast } from '@/lib/hooks/use-toast';
import { copyToClipboard } from '@/lib/utils/clipboard';
import { downloadAsText } from '@/lib/utils/download';
import type { BackupCodesPayload } from '@/lib/api/types';

interface Props {
  // `null` means the dialog is closed. When the parent assigns a
  // payload the dialog opens; on close the parent clears it and the
  // codes are dropped from memory — they never exist anywhere else.
  payload: BackupCodesPayload | null;
  onClose: () => void;
}

// Outer wrapper: forces a fresh inner-component mount per payload so
// `acknowledged` resets cleanly when a regenerate-after-save delivers
// a new set. Using `key` is React's canonical pattern for "re-run
// state init when this thing changes" and sidesteps the setState-in-
// effect lint rule.
export function TwoFactorBackupCodesDialog({ payload, onClose }: Props) {
  return <BackupCodesBody key={payloadKey(payload)} payload={payload} onClose={onClose} />;
}

function payloadKey(p: BackupCodesPayload | null): string {
  if (!p) return 'closed';
  return `${p.factor_id}:${p.codes.length}:${p.codes[0] ?? ''}`;
}

function BackupCodesBody({ payload, onClose }: Props) {
  const t = useTranslations('settings.twoFactor.backupCodesDialog');
  const { toast } = useToast();
  const [acknowledged, setAcknowledged] = useState(false);

  const open = payload !== null;
  const codes = payload?.codes ?? [];
  const joined = codes.join('\n');

  const handleCopy = async () => {
    const ok = await copyToClipboard(joined);
    if (ok) {
      toast({ title: t('copied') });
    } else {
      toast({ title: t('copyFailed'), variant: 'destructive' });
    }
  };

  const handleDownload = () => {
    downloadAsText(joined, t('filename'));
  };

  const handleConfirm = () => {
    setAcknowledged(false);
    onClose();
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        // Ignore outside-click / ESC closes until the user has
        // confirmed they saved the codes. The "Done" button is the
        // one allowed exit — otherwise users lose their codes
        // without noticing.
        if (next === false && !acknowledged) {
          return;
        }
        if (next === false) onClose();
      }}
    >
      <DialogContent
        data-testid="backup-codes-dialog"
        onEscapeKeyDown={(e) => !acknowledged && e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>{t('title')}</DialogTitle>
          <DialogDescription>{t('description')}</DialogDescription>
        </DialogHeader>
        <div
          data-testid="backup-codes-list"
          className="bg-muted/50 grid grid-cols-1 gap-1 rounded-md p-3 font-mono text-sm break-all md:grid-cols-2"
        >
          {codes.map((c, i) => (
            <div key={i} data-testid={`backup-code-${i}`}>
              {c}
            </div>
          ))}
        </div>
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant="outline" onClick={handleCopy}>
            <Copy className="mr-2 h-4 w-4" aria-hidden />
            {t('copy')}
          </Button>
          <Button type="button" variant="outline" onClick={handleDownload}>
            <Download className="mr-2 h-4 w-4" aria-hidden />
            {t('download')}
          </Button>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox
            id="backup-codes-ack"
            checked={acknowledged}
            onCheckedChange={(v) => setAcknowledged(v === true)}
          />
          <Label htmlFor="backup-codes-ack" className="cursor-pointer">
            {t('acknowledge')}
          </Label>
        </div>
        <DialogFooter>
          <Button type="button" onClick={handleConfirm} disabled={!acknowledged}>
            {t('confirm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
