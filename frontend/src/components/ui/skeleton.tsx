import { cn } from '@/lib/utils';

function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  // data-slot is a stable hook for tests. e2e/visual-regression.spec.ts has to
  // know when a page has finished loading, and "no skeletons left" is the
  // signal; keying that off the animate-pulse class would tie it to styling.
  return (
    <div
      data-slot="skeleton"
      className={cn('bg-muted animate-pulse rounded-md', className)}
      {...props}
    />
  );
}

export { Skeleton };
