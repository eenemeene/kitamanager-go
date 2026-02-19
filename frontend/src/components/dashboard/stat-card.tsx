import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';

export function StatCard({
  title,
  value,
  icon: Icon,
  loading,
  description,
  valueClassName,
}: {
  title: string;
  value: number | string;
  icon: React.ComponentType<{ className?: string }>;
  loading?: boolean;
  description?: string;
  valueClassName?: string;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <Icon className="text-muted-foreground h-4 w-4" />
      </CardHeader>
      <CardContent>
        {loading ? (
          <Skeleton className="h-8 w-20" />
        ) : (
          <>
            <div className={`text-2xl font-bold ${valueClassName ?? ''}`}>{value}</div>
            {description && <p className="text-muted-foreground text-xs">{description}</p>}
          </>
        )}
      </CardContent>
    </Card>
  );
}
