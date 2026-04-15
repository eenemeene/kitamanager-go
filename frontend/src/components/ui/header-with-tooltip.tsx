import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';

export function HeaderWithTooltip({ label, tooltip }: { label: string; tooltip: string }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="cursor-help border-b border-dotted border-current">{label}</span>
      </TooltipTrigger>
      <TooltipContent className="max-w-xs">
        <p>{tooltip}</p>
      </TooltipContent>
    </Tooltip>
  );
}
