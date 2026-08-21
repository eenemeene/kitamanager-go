import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SearchInput } from '../search-input';
import { renderWithProviders } from '@/test-utils';

jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string) => key,
}));

describe('SearchInput', () => {
  const onChange = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders input with placeholder', () => {
    renderWithProviders(
      <SearchInput id="test-search" value="" onChange={onChange} placeholder="Search..." />
    );

    expect(screen.getByPlaceholderText('Search...')).toBeInTheDocument();
  });

  it('renders with default placeholder from translations', () => {
    renderWithProviders(<SearchInput id="test-search" value="" onChange={onChange} />);

    expect(screen.getByPlaceholderText('common.search')).toBeInTheDocument();
  });

  it('displays current value', () => {
    renderWithProviders(<SearchInput id="test-search" value="hello" onChange={onChange} />);

    expect(screen.getByDisplayValue('hello')).toBeInTheDocument();
  });

  it('calls onChange when user types', async () => {
    const user = userEvent.setup();
    renderWithProviders(<SearchInput id="test-search" value="" onChange={onChange} />);

    const input = screen.getByRole('textbox');
    await user.type(input, 'a');

    expect(onChange).toHaveBeenCalledWith('a');
  });

  it('has accessible label', () => {
    renderWithProviders(
      <SearchInput id="test-search" value="" onChange={onChange} placeholder="Find children" />
    );

    expect(screen.getByLabelText('Find children')).toBeInTheDocument();
  });
});
