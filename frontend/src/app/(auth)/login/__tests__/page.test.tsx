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
    mockLogin.mockResolvedValue(undefined);
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
          message: 'Invalid credentials',
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
