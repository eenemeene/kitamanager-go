import React from 'react';
import { render, screen } from '@testing-library/react';
import { ContractTimeline } from '../contract-timeline';
import type { BaseContract } from '../timeline-utils';

jest.mock('next-intl', () => ({
  useTranslations: () => {
    const t = (key: string, params?: Record<string, unknown>) => {
      if (params && 'days' in params) return `${key} (${params.days})`;
      return key;
    };
    t.has = () => false;
    return t;
  },
}));

const adjacentContracts: BaseContract[] = [
  { id: 2, from: '2024-07-01T00:00:00Z', to: '2024-12-31T00:00:00Z' },
  { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-06-30T00:00:00Z' },
];

const gapContracts: BaseContract[] = [
  { id: 2, from: '2024-07-01T00:00:00Z', to: '2024-12-31T00:00:00Z' },
  { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-03-31T00:00:00Z' },
];

const renderContent = (contract: BaseContract) => <span>Contract {contract.id}</span>;
const onBoundaryChange = jest.fn().mockResolvedValue(undefined);

describe('ContractTimeline', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders empty state when no contracts', () => {
    render(
      <ContractTimeline
        contracts={[]}
        renderSegmentContent={renderContent}
        onBoundaryChange={onBoundaryChange}
      />
    );
    expect(screen.getByTestId('timeline-empty')).toBeInTheDocument();
  });

  it('renders correct number of segments', () => {
    render(
      <ContractTimeline
        contracts={adjacentContracts}
        renderSegmentContent={renderContent}
        onBoundaryChange={onBoundaryChange}
      />
    );
    expect(screen.getAllByTestId('timeline-segment')).toHaveLength(2);
  });

  it('renders boundary handle between adjacent contracts', () => {
    render(
      <ContractTimeline
        contracts={adjacentContracts}
        renderSegmentContent={renderContent}
        onBoundaryChange={onBoundaryChange}
      />
    );
    expect(screen.getByTestId('boundary-handle')).toBeInTheDocument();
  });

  it('renders gap indicator for non-adjacent contracts', () => {
    render(
      <ContractTimeline
        contracts={gapContracts}
        renderSegmentContent={renderContent}
        onBoundaryChange={onBoundaryChange}
      />
    );
    expect(screen.getByTestId('timeline-gap')).toBeInTheDocument();
    expect(screen.queryByTestId('boundary-handle')).not.toBeInTheDocument();
  });

  it('calls renderSegmentContent for each contract', () => {
    render(
      <ContractTimeline
        contracts={adjacentContracts}
        renderSegmentContent={renderContent}
        onBoundaryChange={onBoundaryChange}
      />
    );
    expect(screen.getByText('Contract 1')).toBeInTheDocument();
    expect(screen.getByText('Contract 2')).toBeInTheDocument();
  });

  it('applies opacity when isUpdating is true', () => {
    render(
      <ContractTimeline
        contracts={adjacentContracts}
        renderSegmentContent={renderContent}
        onBoundaryChange={onBoundaryChange}
        isUpdating
      />
    );
    const timeline = screen.getByTestId('contract-timeline');
    expect(timeline.className).toContain('opacity-60');
  });

  it('renders single contract without boundary or gap', () => {
    const single: BaseContract[] = [
      { id: 1, from: '2024-01-01T00:00:00Z', to: '2024-12-31T00:00:00Z' },
    ];
    render(
      <ContractTimeline
        contracts={single}
        renderSegmentContent={renderContent}
        onBoundaryChange={onBoundaryChange}
      />
    );
    expect(screen.getAllByTestId('timeline-segment')).toHaveLength(1);
    expect(screen.queryByTestId('boundary-handle')).not.toBeInTheDocument();
    expect(screen.queryByTestId('timeline-gap')).not.toBeInTheDocument();
  });
});
