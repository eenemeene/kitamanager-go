import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ContractTimeline } from '../contract-timeline';
import type { BaseContract } from '../timeline-utils';

jest.mock('next-intl', () => ({
  useTranslations: () => {
    const t = (key: string) => key;
    t.has = () => false;
    return t;
  },
}));

// Stand in for the date-picker handle so the payload contract can be asserted
// without driving a calendar popover. Each mocked handle reports the boundary
// move it was given.
jest.mock('../boundary-handle', () => ({
  BoundaryHandle: ({
    upperContract,
    lowerContract,
    onBoundaryChange,
  }: {
    upperContract: { id: number };
    lowerContract: { id: number };
    onBoundaryChange: (newTo: string, newFrom: string) => void;
  }) => (
    <button
      data-testid={`boundary-${lowerContract.id}-${upperContract.id}`}
      onClick={() => onBoundaryChange('2024-09-30', '2024-10-01')}
    />
  ),
}));

// Three contracts, newest first. The middle one HAS a `to` because a third
// contract follows it — that is the case that regressed.
const threeContracts: BaseContract[] = [
  { id: 3, from: '2025-01-01T00:00:00Z', to: null },
  { id: 2, from: '2024-07-01T00:00:00Z', to: '2024-12-31T00:00:00Z' },
  { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-06-30T00:00:00Z' },
];

const renderContent = (c: BaseContract) => <span>Contract {c.id}</span>;

describe('ContractTimeline boundary payload', () => {
  // Regression guard for two bugs fixed together:
  //  - omitting `to` for the upper contract cleared it, so a contract with a
  //    successor became ongoing and collided with it (409)
  //  - a dates-only entry also stripped funding properties server-side
  // Both are avoided by sending the full date pair for both contracts.
  it('sends from AND to for both contracts, preserving the upper contract to', async () => {
    const onBoundaryChange = jest.fn().mockResolvedValue(undefined);
    render(
      <ContractTimeline
        contracts={threeContracts}
        renderSegmentContent={renderContent}
        onBoundaryChange={onBoundaryChange}
      />
    );

    // Move the OLDER boundary: between contract 1 (lower) and contract 2 (upper).
    await userEvent.click(screen.getByTestId('boundary-1-2'));

    expect(onBoundaryChange).toHaveBeenCalledTimes(1);
    const updates = onBoundaryChange.mock.calls[0][0];
    expect(updates).toHaveLength(2);

    const lower = updates.find((u: { id: number }) => u.id === 1);
    const upper = updates.find((u: { id: number }) => u.id === 2);

    // The lower contract keeps its own start and takes the new end.
    expect(lower).toEqual({ id: 1, from: '2024-01-01T00:00:00Z', to: '2024-09-30T00:00:00Z' });

    // The upper contract takes the new start and MUST retain its existing end,
    // otherwise the batch endpoint clears it and it collides with contract 3.
    expect(upper).toEqual({ id: 2, from: '2024-10-01T00:00:00Z', to: '2024-12-31T00:00:00Z' });
    expect(upper.to).toBeDefined();
  });

  it('leaves to undefined when the upper contract is genuinely ongoing', async () => {
    const onBoundaryChange = jest.fn().mockResolvedValue(undefined);
    render(
      <ContractTimeline
        contracts={threeContracts}
        renderSegmentContent={renderContent}
        onBoundaryChange={onBoundaryChange}
      />
    );

    // The newest boundary: contract 2 (lower) and contract 3 (upper, to = null).
    await userEvent.click(screen.getByTestId('boundary-2-3'));

    const updates = onBoundaryChange.mock.calls[0][0];
    const upper = updates.find((u: { id: number }) => u.id === 3);
    expect(upper.from).toBe('2024-10-01T00:00:00Z');
    expect(upper.to).toBeUndefined();
  });
});
