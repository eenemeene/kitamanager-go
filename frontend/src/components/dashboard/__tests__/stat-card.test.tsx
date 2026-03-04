import { render, screen } from '@testing-library/react';
import { StatCard } from '../stat-card';
import { Users } from 'lucide-react';

describe('StatCard', () => {
  it('renders title and value', () => {
    render(<StatCard title="Employees" value={42} icon={Users} />);
    expect(screen.getByText('Employees')).toBeInTheDocument();
    expect(screen.getByText('42')).toBeInTheDocument();
  });

  it('renders string value', () => {
    render(<StatCard title="Status" value="Active" icon={Users} />);
    expect(screen.getByText('Active')).toBeInTheDocument();
  });

  it('shows skeleton when loading', () => {
    const { container } = render(<StatCard title="Loading" value={0} icon={Users} loading />);
    expect(container.querySelector('.animate-pulse')).toBeInTheDocument();
    expect(screen.queryByText('0')).not.toBeInTheDocument();
  });

  it('renders description when provided', () => {
    render(<StatCard title="Count" value={10} icon={Users} description="Total active" />);
    expect(screen.getByText('Total active')).toBeInTheDocument();
  });

  it('does not render description when not provided', () => {
    render(<StatCard title="Count" value={10} icon={Users} />);
    const paragraphs = document.querySelectorAll('p.text-muted-foreground');
    expect(paragraphs).toHaveLength(0);
  });

  it('applies valueClassName to value', () => {
    render(<StatCard title="Balance" value="-5%" icon={Users} valueClassName="text-red-500" />);
    const value = screen.getByText('-5%');
    expect(value.className).toContain('text-red-500');
  });
});
