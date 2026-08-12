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
// contract follows it — that is the case that regressed twice.
const threeContracts: BaseContract[] = [
  { id: 3, version: 7, from: '2025-01-01T00:00:00Z', to: null },
  { id: 2, version: 4, from: '2024-07-01T00:00:00Z', to: '2024-12-31T00:00:00Z' },
  { id: 1, version: 2, from: '2024-01-01T00:00:00Z', to: '2024-06-30T00:00:00Z' },
];

const renderContent = (c: BaseContract) => <span>Contract {c.id}</span>;

describe('ContractTimeline boundary payload', () => {
  // This file used to guard a four-date payload: the client computed `from` and
  // `to` for BOTH contracts, because the batch endpoint it posted to cleared `to`
  // whenever an entry omitted it. Getting that wrong caused two funding bugs —
  // the neighbour's end date was wiped (a 409 for every child with three or more
  // contracts) and, in another version, its care type and supplements went too.
  //
  // The payload is now a seam: one date, two ids, two versions. The guard is
  // therefore inverted — what matters is that no end date is sent at all, which
  // is what makes clearing the neighbour's structurally impossible rather than
  // merely avoided.
  it('sends one seam date with both contract ids and versions', async () => {
    const onBoundaryChange = jest.fn().mockResolvedValue(undefined);
    render(
      <ContractTimeline
        contracts={threeContracts}
        renderSegmentContent={renderContent}
        onBoundaryChange={onBoundaryChange}
      />
    );

    // Move the OLDER boundary: between contract 1 (earlier) and contract 2.
    await userEvent.click(screen.getByTestId('boundary-1-2'));

    expect(onBoundaryChange).toHaveBeenCalledTimes(1);
    const move = onBoundaryChange.mock.calls[0][0];

    // The timeline is sorted newest-first, so the "lower" contract is the earlier
    // one. Naming them explicitly is what lets the server derive both sides.
    expect(move).toEqual({
      earlier_id: 1,
      later_id: 2,
      at: '2024-10-01T00:00:00Z',
      earlier_version: 2,
      later_version: 4,
    });
  });

  it('never sends an end date, so a neighbour cannot be cleared', async () => {
    const onBoundaryChange = jest.fn().mockResolvedValue(undefined);
    render(
      <ContractTimeline
        contracts={threeContracts}
        renderSegmentContent={renderContent}
        onBoundaryChange={onBoundaryChange}
      />
    );

    await userEvent.click(screen.getByTestId('boundary-2-3'));

    const move = onBoundaryChange.mock.calls[0][0];
    expect(move).not.toHaveProperty('to');
    expect(move).not.toHaveProperty('updates');
    // The still-ongoing contract 3 is the later side; its open end is simply not
    // part of the request, so it cannot be lost.
    expect(move).toEqual({
      earlier_id: 2,
      later_id: 3,
      at: '2024-10-01T00:00:00Z',
      earlier_version: 4,
      later_version: 7,
    });
  });
});
