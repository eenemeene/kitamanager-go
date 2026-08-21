'use client';

import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { useTheme } from 'next-themes';
import { Moon, Sun, LogOut, User, Globe, Menu } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import { useAuthStore } from '@/stores/auth-store';
import { useUiStore } from '@/stores/ui-store';
import { cn } from '@/lib/utils';
import { locales, localeNames, type Locale } from '@/i18n/config';
import { MOBILE_SIDEBAR_ID } from './app-sidebar';

export function AppHeader() {
  const t = useTranslations();
  const router = useRouter();
  const { setTheme, theme } = useTheme();
  const { user, logout } = useAuthStore();
  const { sidebarCollapsed, sidebarMobileOpen, toggleMobileSidebar } = useUiStore();

  const handleLogout = async () => {
    // Awaited, because the navigation races the request otherwise. `logout()`
    // is what makes the server clear the session cookie, and the proxy decides
    // whether /login is reachable by looking at `csrf_token` -- so navigating
    // first meant the proxy could still see a logged-in user and bounce the
    // request straight back to the dashboard. Awaiting also guarantees the
    // cache and the stored organization are gone before the login screen, and
    // therefore the next account, appears.
    await logout();
    router.push('/login');
  };

  const handleLocaleChange = (locale: Locale) => {
    document.cookie = `locale=${locale}; path=/; max-age=31536000; SameSite=Strict`;
    router.refresh();
  };

  const userInitials = user?.name
    ? user.name
        .split(' ')
        .map((n) => n[0])
        .join('')
        .toUpperCase()
        .slice(0, 2)
    : user?.email?.slice(0, 2).toUpperCase() || 'U';

  return (
    <header
      className={cn(
        'bg-background fixed top-0 right-0 z-30 flex h-16 items-center justify-between gap-4 border-b px-3 transition-all duration-300 md:justify-end md:px-6',
        'left-0 md:left-16',
        !sidebarCollapsed && 'lg:left-64'
      )}
    >
      {/* Mobile hamburger. Below md this is the only route to every other page,
          so it reports its state: `aria-expanded` says whether the drawer is
          open and `aria-controls` names the panel it opens. The label comes
          from the catalogue — it used to be the English literal "Menu", which
          is what a German screen-reader user heard. */}
      <Button
        variant="ghost"
        size="icon"
        onClick={toggleMobileSidebar}
        className="md:hidden"
        aria-label={t('common.openMenu')}
        aria-expanded={sidebarMobileOpen}
        aria-controls={MOBILE_SIDEBAR_ID}
      >
        <Menu className="h-5 w-5" />
      </Button>

      {/* Theme Toggle */}
      <Button
        variant="ghost"
        size="icon"
        onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
        aria-label={theme === 'dark' ? t('settings.lightMode') : t('settings.darkMode')}
      >
        <Sun className="h-5 w-5 scale-100 rotate-0 transition-all dark:scale-0 dark:-rotate-90" />
        <Moon className="absolute h-5 w-5 scale-0 rotate-90 transition-all dark:scale-100 dark:rotate-0" />
      </Button>

      {/* Language Selector */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon" aria-label={t('settings.language')}>
            <Globe className="h-5 w-5" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel>{t('settings.language')}</DropdownMenuLabel>
          <DropdownMenuSeparator />
          {locales.map((locale) => (
            <DropdownMenuItem key={locale} onClick={() => handleLocaleChange(locale)}>
              {localeNames[locale]}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>

      {/* User Menu */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            className="relative rounded-full"
            aria-label={t('common.userMenu')}
          >
            <Avatar className="h-10 w-10">
              <AvatarFallback className="bg-primary text-primary-foreground">
                {userInitials}
              </AvatarFallback>
            </Avatar>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-56">
          <DropdownMenuLabel className="font-normal">
            <div className="flex flex-col space-y-1">
              <p className="text-sm leading-none font-medium">
                {user?.name || t('common.unknown')}
              </p>
              <p className="text-muted-foreground text-xs leading-none">{user?.email}</p>
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => router.push('/settings')}>
            <User className="mr-2 h-4 w-4" />
            {t('nav.settings')}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={handleLogout}>
            <LogOut className="mr-2 h-4 w-4" />
            {t('auth.logout')}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </header>
  );
}
