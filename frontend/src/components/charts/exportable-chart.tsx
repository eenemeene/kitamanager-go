'use client';

import { useRef, useCallback, type ReactNode } from 'react';
import { Download } from 'lucide-react';
import { cn } from '@/lib/utils';

interface ExportableChartProps {
  children: ReactNode;
  filename: string;
  className?: string;
}

/**
 * Wraps a Nivo chart and adds an SVG export button in the top-right corner.
 * Pass `className` for the chart height, e.g. `className="h-[350px]"`.
 */
export function ExportableChart({ children, filename, className }: ExportableChartProps) {
  const ref = useRef<HTMLDivElement>(null);

  const handleExport = useCallback(() => {
    const svg = ref.current?.querySelector('svg');
    if (!svg) return;

    // Clone the SVG so we can modify it without affecting the chart
    const clone = svg.cloneNode(true) as SVGSVGElement;

    // Resolve CSS variables to computed values for standalone SVG
    const computedStyle = getComputedStyle(document.documentElement);
    const walker = document.createTreeWalker(clone, NodeFilter.SHOW_ELEMENT);
    let node: Node | null = walker.currentNode;
    while (node) {
      if (node instanceof SVGElement || node instanceof HTMLElement) {
        const el = node as SVGElement;
        // Resolve inline style CSS variables
        const style = el.getAttribute('style');
        if (style && style.includes('var(')) {
          el.setAttribute(
            'style',
            style.replace(/hsl\(var\(--([^)]+)\)\)/g, (_, varName) => {
              const value = computedStyle.getPropertyValue(`--${varName}`).trim();
              return value ? `hsl(${value})` : '#000';
            })
          );
        }
        // Resolve fill/stroke attributes
        for (const attr of ['fill', 'stroke']) {
          const val = el.getAttribute(attr);
          if (val && val.includes('var(')) {
            el.setAttribute(
              attr,
              val.replace(/hsl\(var\(--([^)]+)\)\)/g, (_, varName) => {
                const value = computedStyle.getPropertyValue(`--${varName}`).trim();
                return value ? `hsl(${value})` : '#000';
              })
            );
          }
        }
      }
      node = walker.nextNode();
    }

    // Add white background
    const bg = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
    bg.setAttribute('width', '100%');
    bg.setAttribute('height', '100%');
    bg.setAttribute('fill', 'white');
    clone.insertBefore(bg, clone.firstChild);

    // Set explicit dimensions if missing
    if (!clone.getAttribute('width')) {
      clone.setAttribute('width', String(svg.clientWidth));
      clone.setAttribute('height', String(svg.clientHeight));
    }

    const serializer = new XMLSerializer();
    const svgStr = serializer.serializeToString(clone);
    const blob = new Blob([svgStr], { type: 'image/svg+xml;charset=utf-8' });
    const url = URL.createObjectURL(blob);

    const a = document.createElement('a');
    a.href = url;
    a.download = `${filename}.svg`;
    a.click();
    URL.revokeObjectURL(url);
  }, [filename]);

  return (
    // Masked for visual regression, at the wrapper rather than per test.
    //
    // Every chart here redraws from data derived from today, and SVG carries
    // sub-pixel anti-aliasing jitter between runs on top of that, so no chart
    // can be part of a pixel comparison. The tests used to mask
    // `[role="application"]`, which is nivo's own attribute -- it is absent at
    // the mobile breakpoint, so the entire staffing chart was compared there and
    // failed on data that legitimately differs. One attribute on the shared
    // wrapper covers all 35 charts at every breakpoint.
    <div ref={ref} data-visual-mask="chart" className={cn('relative', className)}>
      {children}
      <button
        onClick={handleExport}
        className="bg-background/80 hover:bg-muted absolute top-1 right-1 z-20 rounded-md border p-1.5 opacity-0 transition-opacity hover:opacity-100 [div:hover>&]:opacity-60"
        title="Export SVG"
      >
        <Download className="text-muted-foreground h-3.5 w-3.5" />
      </button>
    </div>
  );
}
