---
paths:
  - "frontend/src/**/*.ts"
  - "frontend/src/**/*.tsx"
---

# Frontend UI conventions

Primary use case: teachers and managers operating the app on **tablets** (768–1024px) and **phones** (375–414px) while standing in a group room. Touch comfort is a first-class requirement, not a polish pass.

## Breakpoints — read this first

| Prefix | Min width | Covers | NOT |
|---|---|---|---|
| (none) | 0 | phones + iPad mini portrait (744px) | — |
| `md:` | 768px | **tablet portrait** — iPad 10/11 (820), iPad Air 11 (820), iPad Pro 11 (834) | NOT desktop |
| `lg:` | 1024px | tablet landscape, iPad Pro 13 / Air 13 portrait (1024), small laptops | — |
| `xl:` | 1280px | desktop | — |

The classic mistake: treating `md:` as "desktop" and shrinking touch targets there. At 768–1023px the user is still tapping with a finger, not clicking with a mouse. **Compact-desktop variants gate on `lg:`/`xl:`, never on `md:`.**

The sidebar is the canonical example: forced into icon-rail at `md:` regardless of stored preference, full sidebar at `lg:+`. See `useIsLgUp()` in `@/hooks/use-media-query`, combined into `effectiveCollapsed` in `app-sidebar.tsx`. New sidebar-adjacent UI should follow the same pattern rather than keying off `sidebarCollapsed` directly.

## Touch targets — 44×44px minimum

The design-system primitives are pre-sized correctly:

| Primitive | Height | Notes |
|---|---|---|
| `Button` (default) | 44px (`h-11`) | |
| `Button size="icon"` | 44×44 (`h-11 w-11`) | use for all icon-only buttons |
| `Button size="lg"` | 48px | primary CTAs / hero actions |
| `Button size="sm"` | 36px | **avoid for primary actions**; dense desktop only |
| `Button size="icon-sm"` | 36×36 | only for very dense cells (e.g. attendance week grid) |
| `Input`, `SelectTrigger` | 44px (`h-11`) | |

Rules:
- Don't override these with `h-9`/`h-10` to match legacy mockups.
- Don't `size="sm"` for "New X"/save/submit/primary-row actions — that's 36px and fails the minimum.
- Don't use a responsive shrink like `h-11 w-11 md:h-10 md:w-10` (see breakpoints).
- For a clickable bare `<button>` (inline editors, linkified text): `min-h-9` + `px-2 py-1` so the hit area clears 36px.
- For calendar day cells / dense grids: `h-11 w-11 md:h-9 md:w-9` — full target on mobile, compact on desktop where mouse precision is available.

## Layout rules

- **Mobile-first stacking**: `flex-col` default, opt into `sm:flex-row`/`md:flex-row` for wider screens.
- **Responsive grids**: `grid-cols-1 md:grid-cols-2`. Never fixed `grid-cols-2`/`grid-cols-3` for page-level layouts.
- **No fixed pixel widths on layout containers.** Icon buttons and stepper value labels (e.g. `min-w-[80px]`) are exceptions.
- **Filter bars**: `flex flex-wrap gap-2` (or `gap-2 md:gap-4`) — controls must wrap on narrow screens.
- **Page headers**: stack title + actions on phone (`flex-col gap-3 sm:flex-row sm:items-center sm:justify-between`); `truncate` long titles; `flex-wrap` action groups.
- **Content padding**: `p-3 md:p-6`.
- **Tables**: hide non-essential columns with `hidden md:table-cell`. The `<Table>` primitive scrolls horizontally so wide tables won't break the page, but forcing horizontal scroll for primary columns is a smell.

## Tables on narrow viewports

Hide non-essential columns at `md:` with `hidden md:table-cell` (sample: `children-table.tsx` hides Gender/Birthdate/Age on phone).

**Action columns must wrap their buttons in a flex container.** Without it, a cell with 3+ icon buttons wraps the icons vertically when the column runs out of width:

```tsx
// CORRECT — buttons stay horizontal, gap is tight
<TableCell className="text-right">
  <div className="flex flex-nowrap items-center justify-end gap-0.5">
    <Button size="icon">...</Button>
    <Button size="icon">...</Button>
  </div>
</TableCell>

// WRONG — buttons wrap to separate rows on tablet
<TableCell className="text-right">
  <Button size="icon">...</Button>
  <Button size="icon">...</Button>
</TableCell>
```

When a row has many actions (>3) and the table shows at `md:`, hide secondary ones with `className="hidden lg:inline-flex"`. Keep only Edit + Delete inline at `md:`.

## Verification

New pages must be verified at **375px** (phone), **820px** (mainstream iPad portrait), and **1280px** (desktop). Use Playwright or the browser DevTools device toolbar.

The E2E suite has `responsive.spec.ts` with `Responsive Layout - Mobile / Tablet / Desktop` describes as a template. The Tablet describe uses `hasTouch: true` and asserts touch-target sizes (≥44px) on the daily-use surfaces.
