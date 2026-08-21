import { screen } from '@testing-library/react';
import OrganizationNotFound from '../not-found';
import { renderWithProviders } from '@/test-utils';

jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string) => key,
}));

describe('OrganizationNotFound', () => {
  it('names the problem and offers a way out', () => {
    renderWithProviders(<OrganizationNotFound />);

    expect(
      screen.getByRole('heading', { name: 'organizations.notFoundTitle' })
    ).toBeInTheDocument();
    expect(screen.getByText('organizations.notFoundDescription')).toBeInTheDocument();

    // The escape hatch matters as much as the message: this replaces a page that
    // looked like it was working, so the reader arrives here without having
    // chosen to and needs somewhere to go.
    const back = screen.getByRole('link', { name: 'organizations.notFoundAction' });
    expect(back).toHaveAttribute('href', '/organizations');
  });
});
