'use client';

import { useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useAuthStore } from '@/stores/auth-store';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { getErrorMessage } from '@/lib/api/client';
import { loginSchema, type LoginFormData } from '@/lib/schemas';

/**
 * Validate that a path is safe for redirect (prevents open redirect attacks).
 * Only allows relative paths that start with / and don't contain protocol schemes.
 */
function isValidRedirectPath(path: string): boolean {
  // Must start with a single slash
  if (!path.startsWith('/')) return false;
  // Reject protocol-relative URLs (//example.com)
  if (path.startsWith('//')) return false;
  // Reject URLs with protocol schemes
  if (path.includes('://')) return false;
  // Reject paths that could be interpreted as absolute URLs
  if (path.includes('\\')) return false;
  return true;
}

export default function LoginPage() {
  const t = useTranslations();
  const router = useRouter();
  const searchParams = useSearchParams();
  const login = useAuthStore((state) => state.login);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
  });

  const onSubmit = async (data: LoginFormData) => {
    setError(null);
    setIsLoading(true);

    try {
      await login(data);

      // Redirect to the original page or dashboard
      // Validate the 'from' parameter to prevent open redirect attacks
      const from = searchParams.get('from');
      const redirectTo = from && isValidRedirectPath(from) ? from : '/';
      router.push(redirectTo);
    } catch (err) {
      setError(getErrorMessage(err, t('auth.loginError')));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="via-background dark:via-background flex min-h-screen items-center justify-center bg-gradient-to-br from-purple-50 to-purple-100/50 p-4 dark:from-purple-950/30 dark:to-purple-900/20">
      <Card className="w-full max-w-md shadow-lg">
        <CardHeader className="space-y-1">
          <div className="bg-primary mx-auto mb-2 h-1.5 w-12 rounded-full" />
          <CardTitle className="text-2xl font-bold">{t('common.appName')}</CardTitle>
          <CardDescription>{t('auth.loginTitle')}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            {error && (
              <div className="bg-destructive/10 text-destructive rounded-md p-3 text-sm">
                {error}
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="email">{t('auth.email')}</Label>
              <Input
                id="email"
                type="email"
                placeholder="name@example.com"
                aria-invalid={!!errors.email}
                aria-describedby={errors.email ? 'email-error' : undefined}
                {...register('email')}
                disabled={isLoading}
              />
              {errors.email && (
                <p id="email-error" className="text-destructive text-sm">
                  {t('validation.invalidEmail')}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">{t('auth.password')}</Label>
              <Input
                id="password"
                type="password"
                aria-invalid={!!errors.password}
                aria-describedby={errors.password ? 'password-error' : undefined}
                {...register('password')}
                disabled={isLoading}
              />
              {errors.password && (
                <p id="password-error" className="text-destructive text-sm">
                  {t('validation.passwordRequired')}
                </p>
              )}
            </div>

            <Button type="submit" className="w-full" disabled={isLoading}>
              {isLoading ? t('common.loading') : t('auth.loginButton')}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
