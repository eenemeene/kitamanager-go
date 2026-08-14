import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import LoginPage from '../page';
import { useAuthStore } from '@/stores/auth-store';

// Mock the auth store
jest.mock('@/stores/auth-store', () => ({
  useAuthStore: jest.fn(),
}));

// Mock searchParams
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: jest.fn(),
  }),
  useSearchParams: () => ({
    get: jest.fn(() => null),
  }),
}));

describe('LoginPage', () => {
  const mockLogin = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    (useAuthStore as unknown as jest.Mock).mockImplementation((selector) => {
      const state = {
        login: mockLogin,
        token: null,
      };
      return selector ? selector(state) : state;
    });
  });

  it('renders login form', () => {
    render(<LoginPage />);

    expect(screen.getByLabelText('auth.email')).toBeInTheDocument();
    expect(screen.getByLabelText('auth.password')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'auth.loginButton' })).toBeInTheDocument();
  });

  it('renders app name and login title', () => {
    render(<LoginPage />);

    expect(screen.getByText('common.appName')).toBeInTheDocument();
    expect(screen.getByText('auth.loginTitle')).toBeInTheDocument();
  });

  it('does not call login with invalid email', async () => {
    render(<LoginPage />);

    const emailInput = screen.getByLabelText('auth.email');
    const passwordInput = screen.getByLabelText('auth.password');
    const submitButton = screen.getByRole('button', { name: 'auth.loginButton' });

    // Type invalid email (no @ symbol)
    await userEvent.type(emailInput, 'invalid-email');
    await userEvent.type(passwordInput, 'password123');
    await userEvent.click(submitButton);

    // Wait a bit for any potential form submission
    await waitFor(() => {
      // Login should not have been called with invalid email
      expect(mockLogin).not.toHaveBeenCalled();
    });
  });

  it('shows validation error for empty password', async () => {
    render(<LoginPage />);

    const emailInput = screen.getByLabelText('auth.email');
    const submitButton = screen.getByRole('button', { name: 'auth.loginButton' });

    await userEvent.type(emailInput, 'test@example.com');
    await userEvent.click(submitButton);

    await waitFor(
      () => {
        expect(screen.getByText('validation.passwordRequired')).toBeInTheDocument();
      },
      { timeout: 3000 }
    );
  });

  it('calls login on valid form submission', async () => {
    mockLogin.mockResolvedValue({ status: 'authenticated', expires_in: 604800 });
    (useAuthStore as unknown as jest.Mock).mockImplementation((selector) => {
      const state = {
        login: mockLogin,
        token: 'mock-token',
      };
      return selector ? selector(state) : state;
    });

    render(<LoginPage />);

    const emailInput = screen.getByLabelText('auth.email');
    const passwordInput = screen.getByLabelText('auth.password');
    const submitButton = screen.getByRole('button', { name: 'auth.loginButton' });

    await userEvent.type(emailInput, 'test@example.com');
    await userEvent.type(passwordInput, 'password123');
    await userEvent.click(submitButton);

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith({
        email: 'test@example.com',
        password: 'password123',
      });
    });
  });

  it('displays error message on login failure', async () => {
    mockLogin.mockRejectedValue({
      response: {
        data: {
          // A problem document, as the API sends: the UI reads `detail`/`code`.
          status: 401,
          code: 'unauthorized',
          detail: 'Invalid credentials',
        },
      },
    });

    render(<LoginPage />);

    const emailInput = screen.getByLabelText('auth.email');
    const passwordInput = screen.getByLabelText('auth.password');
    const submitButton = screen.getByRole('button', { name: 'auth.loginButton' });

    await userEvent.type(emailInput, 'test@example.com');
    await userEvent.type(passwordInput, 'wrongpassword');
    await userEvent.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText('Invalid credentials')).toBeInTheDocument();
    });
  });

  it('disables inputs while loading', async () => {
    // Make login hang
    mockLogin.mockImplementation(() => new Promise(() => {}));

    render(<LoginPage />);

    const emailInput = screen.getByLabelText('auth.email');
    const passwordInput = screen.getByLabelText('auth.password');
    const submitButton = screen.getByRole('button', { name: 'auth.loginButton' });

    await userEvent.type(emailInput, 'test@example.com');
    await userEvent.type(passwordInput, 'password123');
    await userEvent.click(submitButton);

    await waitFor(() => {
      expect(emailInput).toBeDisabled();
      expect(passwordInput).toBeDisabled();
      expect(submitButton).toBeDisabled();
    });
  });
});

// Page-level state machine tests. The login page goes from
//   password  →  (authenticated: navigate)
//              →  (mfa_required: swap in MfaVerifyForm)
//              →  (password error: stay, show banner)
// and from mfa_required back to password via the MfaVerifyForm's
// onRestart callback (user clicks Back, or gets 429).
describe('LoginPage — state machine', () => {
  const mockLogin = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    (useAuthStore as unknown as jest.Mock).mockImplementation((selector) => {
      const state = { login: mockLogin, hydrateAfterAuth: jest.fn() };
      return selector ? selector(state) : state;
    });
  });

  it('stays in password state on authenticated response (and does not show MFA form)', async () => {
    mockLogin.mockResolvedValue({ status: 'authenticated', expires_in: 1 });
    render(<LoginPage />);
    await userEvent.type(screen.getByLabelText('auth.email'), 'a@example.com');
    await userEvent.type(screen.getByLabelText('auth.password'), 'pw-123456');
    await userEvent.click(screen.getByRole('button', { name: 'auth.loginButton' }));
    await waitFor(() => expect(mockLogin).toHaveBeenCalled());
    // No MFA form should appear — navigation is handled by router.push.
    expect(screen.queryByTestId('mfa-verify-form')).not.toBeInTheDocument();
  });

  it('swaps in MFA form on mfa_required response', async () => {
    mockLogin.mockResolvedValue({
      status: 'mfa_required',
      pending_token: 'pending-abc',
      expires_at: 'never',
      factors: [{ id: 42, type: 'totp', label: 'iPhone' }],
    });
    render(<LoginPage />);
    await userEvent.type(screen.getByLabelText('auth.email'), 'a@example.com');
    await userEvent.type(screen.getByLabelText('auth.password'), 'pw-123456');
    await userEvent.click(screen.getByRole('button', { name: 'auth.loginButton' }));
    await waitFor(() => {
      expect(screen.getByTestId('mfa-verify-form')).toBeInTheDocument();
    });
    // Password inputs are gone — we transitioned state.
    expect(screen.queryByLabelText('auth.email')).not.toBeInTheDocument();
  });

  it('reverts to password state when MFA form Back button is clicked', async () => {
    mockLogin.mockResolvedValue({
      status: 'mfa_required',
      pending_token: 'pending-abc',
      expires_at: 'never',
      factors: [{ id: 42, type: 'totp' }],
    });
    render(<LoginPage />);
    await userEvent.type(screen.getByLabelText('auth.email'), 'a@example.com');
    await userEvent.type(screen.getByLabelText('auth.password'), 'pw-123456');
    await userEvent.click(screen.getByRole('button', { name: 'auth.loginButton' }));
    await waitFor(() => expect(screen.getByTestId('mfa-verify-form')).toBeInTheDocument());
    // jest.setup.js's next-intl mock returns bare keys (no namespace),
    // so the button inside MfaVerifyForm renders as just "back".
    await userEvent.click(screen.getByRole('button', { name: 'back' }));
    expect(screen.getByLabelText('auth.email')).toBeInTheDocument();
    expect(screen.queryByTestId('mfa-verify-form')).not.toBeInTheDocument();
  });

  it('shows banner on password error in password state', async () => {
    mockLogin.mockRejectedValue({
      response: { data: { status: 401, code: 'unauthorized', detail: 'Invalid credentials' } },
    });
    render(<LoginPage />);
    await userEvent.type(screen.getByLabelText('auth.email'), 'a@example.com');
    await userEvent.type(screen.getByLabelText('auth.password'), 'wrongpw');
    await userEvent.click(screen.getByRole('button', { name: 'auth.loginButton' }));
    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Invalid credentials');
    });
    // Still in password state.
    expect(screen.getByLabelText('auth.email')).toBeInTheDocument();
  });
});
