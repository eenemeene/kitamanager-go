import { Info } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';

export function StatCard({
  title,
  value,
  icon: Icon,
  loading,
  description,
  valueClassName,
  tooltip,
}: {
  title: string;
  value: number | string;
  icon: React.ComponentType<{ className?: string }>;
  loading?: boolean;
  description?: string;
  valueClassName?: string;
  tooltip?: string;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <div className="flex items-center gap-1.5">
          <CardTitle className="text-sm font-medium">{title}</CardTitle>
          {tooltip && (
            <Tooltip delayDuration={0}>
              <TooltipTrigger
                className="text-muted-foreground hover:text-foreground cursor-help"
                aria-label={tooltip}
              >
                <Info className="h-3.5 w-3.5" />
              </TooltipTrigger>
              <TooltipContent className="max-w-xs">{tooltip}</TooltipContent>
            </Tooltip>
          )}
        </div>
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
