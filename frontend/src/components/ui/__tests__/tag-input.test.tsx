import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TagInput, PropertyTagInput } from '../tag-input';
import { renderWithProviders } from '@/test-utils';

describe('TagInput', () => {
  const onChange = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders existing tags', () => {
    renderWithProviders(<TagInput value={['red', 'blue']} onChange={onChange} />);

    expect(screen.getByText('red')).toBeInTheDocument();
    expect(screen.getByText('blue')).toBeInTheDocument();
  });

  it('shows placeholder when no tags', () => {
    renderWithProviders(<TagInput value={[]} onChange={onChange} placeholder="Add tags" />);
    expect(screen.getByPlaceholderText('Add tags')).toBeInTheDocument();
  });

  it('hides placeholder when tags exist', () => {
    renderWithProviders(<TagInput value={['red']} onChange={onChange} placeholder="Add tags" />);
    expect(screen.getByRole('textbox')).toHaveAttribute('placeholder', '');
  });

  it('adds tag on Enter', async () => {
    const user = userEvent.setup();
    renderWithProviders(<TagInput value={[]} onChange={onChange} />);

    const input = screen.getByRole('textbox');
    await user.type(input, 'green{Enter}');

    expect(onChange).toHaveBeenCalledWith(['green']);
  });

  it('adds tag on comma', async () => {
    const user = userEvent.setup();
    renderWithProviders(<TagInput value={[]} onChange={onChange} />);

    const input = screen.getByRole('textbox');
    await user.type(input, 'green,');

    expect(onChange).toHaveBeenCalledWith(['green']);
  });

  it('trims and lowercases tags', async () => {
    const user = userEvent.setup();
    renderWithProviders(<TagInput value={[]} onChange={onChange} />);

    const input = screen.getByRole('textbox');
    await user.type(input, '  RED  {Enter}');

    expect(onChange).toHaveBeenCalledWith(['red']);
  });

  it('prevents duplicate tags', async () => {
    const user = userEvent.setup();
    renderWithProviders(<TagInput value={['red']} onChange={onChange} />);

    const input = screen.getByRole('textbox');
    await user.type(input, 'red{Enter}');

    expect(onChange).not.toHaveBeenCalled();
  });

  it('removes tag on X button click', async () => {
    const user = userEvent.setup();
    renderWithProviders(<TagInput value={['red', 'blue']} onChange={onChange} />);

    await user.click(screen.getByLabelText('Remove red'));

    expect(onChange).toHaveBeenCalledWith(['blue']);
  });

  it('removes last tag on backspace with empty input', async () => {
    const user = userEvent.setup();
    renderWithProviders(<TagInput value={['red', 'blue']} onChange={onChange} />);

    const input = screen.getByRole('textbox');
    await user.click(input);
    await user.keyboard('{Backspace}');

    expect(onChange).toHaveBeenCalledWith(['red']);
  });

  it('adds tag on blur', async () => {
    const user = userEvent.setup();
    renderWithProviders(<TagInput value={[]} onChange={onChange} />);

    const input = screen.getByRole('textbox');
    await user.type(input, 'green');
    await user.tab(); // triggers blur

    expect(onChange).toHaveBeenCalledWith(['green']);
  });

  it('shows available suggestions', () => {
    renderWithProviders(
      <TagInput value={['red']} onChange={onChange} suggestions={['red', 'blue', 'green']} />
    );

    // red is already selected, should not appear as suggestion button (only Remove button exists)
    const redButtons = screen.getAllByRole('button', { name: /red/i });
    expect(redButtons).toHaveLength(1); // only the "Remove red" button
    expect(redButtons[0]).toHaveAttribute('aria-label', 'Remove red');
    expect(screen.getByRole('button', { name: /blue/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /green/i })).toBeInTheDocument();
  });

  it('adds tag when suggestion clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(<TagInput value={[]} onChange={onChange} suggestions={['blue']} />);

    await user.click(screen.getByRole('button', { name: /blue/i }));

    expect(onChange).toHaveBeenCalledWith(['blue']);
  });

  it('hides remove buttons when disabled', () => {
    renderWithProviders(<TagInput value={['red']} onChange={onChange} disabled />);

    expect(screen.getByText('red')).toBeInTheDocument();
    expect(screen.queryByLabelText('Remove red')).toBeNull();
  });
});

describe('PropertyTagInput', () => {
  const onChange = jest.fn();
  const attrs = [
    { key: 'color', value: 'Red' },
    { key: 'color', value: 'Blue' },
    { key: 'size', value: 'Large' },
  ];

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders selected values as badges', () => {
    renderWithProviders(
      <PropertyTagInput value={{ color: 'Red' }} onChange={onChange} fundingAttributes={attrs} />
    );

    expect(screen.getByText('Red')).toBeInTheDocument();
  });

  it('shows placeholder when no values', () => {
    renderWithProviders(
      <PropertyTagInput value={undefined} onChange={onChange} placeholder="Pick..." />
    );

    expect(screen.getByText('Pick...')).toBeInTheDocument();
  });

  it('adds attribute on suggestion click', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <PropertyTagInput value={undefined} onChange={onChange} fundingAttributes={attrs} />
    );

    await user.click(screen.getByRole('button', { name: /Large/i }));

    expect(onChange).toHaveBeenCalledWith({ size: 'Large' });
  });

  it('replaces attribute with same key', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <PropertyTagInput value={{ color: 'Red' }} onChange={onChange} fundingAttributes={attrs} />
    );

    // Blue has same key 'color' as Red - should replace
    await user.click(screen.getByRole('button', { name: /Blue/i }));

    expect(onChange).toHaveBeenCalledWith({ color: 'Blue' });
  });

  it('removes attribute on X click', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <PropertyTagInput
        value={{ color: 'Red', size: 'Large' }}
        onChange={onChange}
        fundingAttributes={attrs}
      />
    );

    await user.click(screen.getByLabelText('Remove Red'));

    expect(onChange).toHaveBeenCalledWith({ size: 'Large' });
  });

  it('returns undefined when last attribute removed', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <PropertyTagInput value={{ color: 'Red' }} onChange={onChange} fundingAttributes={attrs} />
    );

    await user.click(screen.getByLabelText('Remove Red'));

    expect(onChange).toHaveBeenCalledWith(undefined);
  });

  it('hides suggestions when disabled', () => {
    renderWithProviders(
      <PropertyTagInput value={undefined} onChange={onChange} fundingAttributes={attrs} disabled />
    );

    expect(screen.queryByTestId('property-suggestions')).toBeNull();
  });

  it('does not show already-selected values as suggestions', () => {
    renderWithProviders(
      <PropertyTagInput value={{ color: 'Red' }} onChange={onChange} fundingAttributes={attrs} />
    );

    // Red is selected, should not appear in suggestions
    const suggestions = screen.getByTestId('property-suggestions');
    expect(suggestions).not.toHaveTextContent('Red');
    // Blue and Large should still be available
    expect(screen.getByRole('button', { name: /Blue/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Large/i })).toBeInTheDocument();
  });
});
