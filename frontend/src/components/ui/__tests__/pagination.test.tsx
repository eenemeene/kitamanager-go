import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Pagination } from '../pagination';
import { renderWithProviders } from '@/test-utils';

jest.mock('next-intl', () => ({
  useTranslations: () => (key: string, params?: Record<string, unknown>) => {
    if (params) return `${key}(${JSON.stringify(params)})`;
    return key;
  },
}));

describe('Pagination', () => {
  const onPageChange = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders nothing when totalPages <= 1', () => {
    const { container } = renderWithProviders(
      <Pagination page={1} totalPages={1} total={5} limit={10} onPageChange={onPageChange} />
    );
    expect(container.querySelector('nav')).toBeNull();
  });

  it('renders navigation buttons', () => {
    renderWithProviders(
      <Pagination page={2} totalPages={5} total={50} limit={10} onPageChange={onPageChange} />
    );

    expect(screen.getByLabelText('pagination.firstPage')).toBeInTheDocument();
    expect(screen.getByLabelText('pagination.previousPage')).toBeInTheDocument();
    expect(screen.getByLabelText('pagination.nextPage')).toBeInTheDocument();
    expect(screen.getByLabelText('pagination.lastPage')).toBeInTheDocument();
  });

  it('disables first/previous on page 1', () => {
    renderWithProviders(
      <Pagination page={1} totalPages={3} total={30} limit={10} onPageChange={onPageChange} />
    );

    expect(screen.getByLabelText('pagination.firstPage')).toBeDisabled();
    expect(screen.getByLabelText('pagination.previousPage')).toBeDisabled();
    expect(screen.getByLabelText('pagination.nextPage')).toBeEnabled();
    expect(screen.getByLabelText('pagination.lastPage')).toBeEnabled();
  });

  it('disables next/last on last page', () => {
    renderWithProviders(
      <Pagination page={3} totalPages={3} total={30} limit={10} onPageChange={onPageChange} />
    );

    expect(screen.getByLabelText('pagination.firstPage')).toBeEnabled();
    expect(screen.getByLabelText('pagination.previousPage')).toBeEnabled();
    expect(screen.getByLabelText('pagination.nextPage')).toBeDisabled();
    expect(screen.getByLabelText('pagination.lastPage')).toBeDisabled();
  });

  it('calls onPageChange(1) when first page clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Pagination page={3} totalPages={5} total={50} limit={10} onPageChange={onPageChange} />
    );

    await user.click(screen.getByLabelText('pagination.firstPage'));
    expect(onPageChange).toHaveBeenCalledWith(1);
  });

  it('calls onPageChange(page-1) when previous clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Pagination page={3} totalPages={5} total={50} limit={10} onPageChange={onPageChange} />
    );

    await user.click(screen.getByLabelText('pagination.previousPage'));
    expect(onPageChange).toHaveBeenCalledWith(2);
  });

  it('calls onPageChange(page+1) when next clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Pagination page={3} totalPages={5} total={50} limit={10} onPageChange={onPageChange} />
    );

    await user.click(screen.getByLabelText('pagination.nextPage'));
    expect(onPageChange).toHaveBeenCalledWith(4);
  });

  it('calls onPageChange(totalPages) when last page clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Pagination page={3} totalPages={5} total={50} limit={10} onPageChange={onPageChange} />
    );

    await user.click(screen.getByLabelText('pagination.lastPage'));
    expect(onPageChange).toHaveBeenCalledWith(5);
  });

  it('disables all buttons when loading', () => {
    renderWithProviders(
      <Pagination
        page={2}
        totalPages={3}
        total={30}
        limit={10}
        onPageChange={onPageChange}
        isLoading
      />
    );

    expect(screen.getByLabelText('pagination.firstPage')).toBeDisabled();
    expect(screen.getByLabelText('pagination.previousPage')).toBeDisabled();
    expect(screen.getByLabelText('pagination.nextPage')).toBeDisabled();
    expect(screen.getByLabelText('pagination.lastPage')).toBeDisabled();
  });
});
