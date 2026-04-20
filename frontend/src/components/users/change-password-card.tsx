'use client';

import { useTranslations } from 'next-intl';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useToast } from '@/lib/hooks/use-toast';
import { apiClient } from '@/lib/api/client';
import { changePasswordSchema, type ChangePasswordFormData } from '@/lib/schemas';

export function ChangePasswordCard() {
  const t = useTranslations();
  const { toast } = useToast();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isValid },
  } = useForm<ChangePasswordFormData>({
    resolver: zodResolver(changePasswordSchema),
    mode: 'onChange',
    defaultValues: { current_password: '', new_password: '', confirm_password: '' },
  });

  const mutation = useMutation({
    mutationFn: (data: ChangePasswordFormData) =>
      apiClient.changePassword(data.current_password, data.new_password),
    onSuccess: () => {
      toast({ title: t('settings.password.successToast') });
      reset();
    },
    onError: () => {
      // Intentionally generic — never surface the backend error message
      // for password-change to avoid leaking whether the *current* password
      // was wrong vs. rate-limited vs. anything else. Defense-in-depth in
      // case a future handler regression returns more detail than it should.
      toast({
        title: t('settings.password.errorGeneric'),
        variant: 'destructive',
      });
    },
  });

  const onSubmit = (data: ChangePasswordFormData) => {
    mutation.mutate(data);
  };

  const currentError = errors.current_password?.message;
  const newError = errors.new_password?.message;
  const confirmError = errors.confirm_password?.message;

  // zod schema stores i18n keys in `message`; translate them here. If the key
  // isn't one of ours, fall back to showing the raw string unchanged.
  const translateError = (msg: string | undefined) => {
    if (!msg) return undefined;
    if (msg.startsWith('settings.password.validation.')) return t(msg);
    // zod default messages for min/max etc. — translate to our generic tooShort
    if (msg.toLowerCase().includes('at least')) return t('settings.password.validation.tooShort');
    return msg;
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('settings.password.title')}</CardTitle>
        <CardDescription>{t('settings.password.description')}</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
          <div className="space-y-1">
            <Label htmlFor="current_password">{t('settings.password.currentLabel')}</Label>
            <Input
              id="current_password"
              type="password"
              autoComplete="current-password"
              aria-invalid={!!currentError}
              {...register('current_password')}
            />
            {currentError && (
              <p role="alert" className="text-destructive text-sm">
                {translateError(currentError) ?? t('settings.password.validation.currentRequired')}
              </p>
            )}
          </div>

          <div className="space-y-1">
            <Label htmlFor="new_password">{t('settings.password.newLabel')}</Label>
            <Input
              id="new_password"
              type="password"
              autoComplete="new-password"
              aria-invalid={!!newError}
              {...register('new_password')}
            />
            {newError && (
              <p role="alert" className="text-destructive text-sm">
                {translateError(newError)}
              </p>
            )}
          </div>

          <div className="space-y-1">
            <Label htmlFor="confirm_password">{t('settings.password.confirmLabel')}</Label>
            <Input
              id="confirm_password"
              type="password"
              autoComplete="new-password"
              aria-invalid={!!confirmError}
              {...register('confirm_password')}
            />
            {confirmError && (
              <p role="alert" className="text-destructive text-sm">
                {translateError(confirmError)}
              </p>
            )}
          </div>

          <Button type="submit" disabled={!isValid || mutation.isPending}>
            {t('settings.password.submit')}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
