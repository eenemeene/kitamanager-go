import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AppHeader } from '../app-header';

const pushMock = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: pushMock, back: jest.fn(), refresh: jest.fn() }),
}));

jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string, params?: Record<string, unknown>) => {
    if (params) return `${key}`;
    return key;
  },
}));

jest.mock('next-themes', () => ({
  useTheme: () => ({ setTheme: jest.fn(), theme: 'light' }),
}));

const mockLogout = jest.fn();
jest.mock('@/stores/auth-store', () => ({
  useAuthStore: () => ({
    user: { name: 'John Doe', email: 'john@test.com' },
    logout: mockLogout,
  }),
}));

jest.mock('@/stores/ui-store', () => ({
  useUiStore: () => ({ sidebarCollapsed: false, toggleMobileSidebar: jest.fn() }),
}));

jest.mock('@/i18n/config', () => ({
  locales: ['en', 'de'],
  localeNames: { en: 'English', de: 'Deutsch' },
}));

describe('AppHeader', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders theme toggle button', () => {
    render(<AppHeader />);

    // Theme toggle has aria-label based on current theme
    const themeButton = screen.getByLabelText('settings.darkMode');
    expect(themeButton).toBeInTheDocument();
  });

  it('renders language selector button', () => {
    render(<AppHeader />);

    const langButton = screen.getByLabelText('settings.language');
    expect(langButton).toBeInTheDocument();
  });

  it('renders user avatar with initials (JD)', () => {
    render(<AppHeader />);

    expect(screen.getByText('JD')).toBeInTheDocument();
  });

  it('renders logout menu item', async () => {
    const user = userEvent.setup();
    render(<AppHeader />);

    // Click the avatar button to open the user dropdown
    const avatarButton = screen.getByText('JD').closest('button')!;
    await user.click(avatarButton);

    await screen.findByText('auth.logout');
    expect(screen.getByText('auth.logout')).toBeInTheDocument();
  });

  it('renders user name and email in user menu', async () => {
    const user = userEvent.setup();
    render(<AppHeader />);

    // Click the avatar button to open the user dropdown
    const avatarButton = screen.getByText('JD').closest('button')!;
    await user.click(avatarButton);

    await screen.findByText('John Doe');
    expect(screen.getByText('John Doe')).toBeInTheDocument();
    expect(screen.getByText('john@test.com')).toBeInTheDocument();
  });

  it('renders Settings menu item enabled and navigates to /settings on click', async () => {
    const user = userEvent.setup();
    render(<AppHeader />);

    const avatarButton = screen.getByText('JD').closest('button')!;
    await user.click(avatarButton);

    const settingsItem = await screen.findByText('nav.settings');
    // The item is NOT disabled. `aria-disabled` is how radix reports disabled state.
    const settingsRow = settingsItem.closest('[role="menuitem"]') as HTMLElement;
    expect(settingsRow).not.toBeNull();
    expect(settingsRow.getAttribute('aria-disabled')).not.toBe('true');

    await user.click(settingsItem);
    expect(pushMock).toHaveBeenCalledWith('/settings');
  });
});
