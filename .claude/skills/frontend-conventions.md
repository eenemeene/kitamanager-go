# Frontend Development Conventions

## Globs

- `frontend/src/**/*.tsx`
- `frontend/src/**/*.ts`
- `frontend/e2e/**/*.ts`
- `frontend/package.json`

## Tech Stack

- **Framework**: Next.js 16 (App Router) with TypeScript
- **UI Components**: shadcn/ui (Radix UI primitives + Tailwind CSS)
- **State Management**: Zustand (client stores), TanStack React Query (server state)
- **Forms**: React Hook Form + Zod validation
- **API Client**: Axios (singleton `apiClient` in `src/lib/api/client.ts`)
- **i18n**: next-intl (EN + DE required)
- **Charts**: Nivo
- **Drag & Drop**: @dnd-kit
- **Testing**: Jest + React Testing Library (unit), Playwright (E2E)

## Page Patterns

### Simple CRUD Pages (Groups pattern)

For pages with a flat list + create/edit/delete dialogs:

1. Use `useQuery` for data fetching
2. Use `useCrudDialogs` hook for dialog state management
3. Use `useCrudMutations` hook for create/update/delete mutations
4. Use `CrudPageHeader` + `ResourceTable` + `DeleteConfirmDialog` from `@/components/crud`
5. Use `Pagination` component for paginated lists
6. Form validation with `zod` schema + `react-hook-form`

Reference: `src/app/(dashboard)/organizations/[orgId]/groups/page.tsx`

### Complex Pages (Children pattern)

For pages with detail views, nested resources, or multiple tabs:

1. List page with table + link to detail view
2. Detail page with tabs for different sections
3. Nested CRUD operations (e.g., child contracts)

Reference: `src/app/(dashboard)/organizations/[orgId]/children/`

## API Client Conventions

- All API methods live in the singleton `ApiClient` class in `src/lib/api/client.ts`
- Organization-scoped endpoints take `orgId` as first parameter
- Paginated endpoints accept `PaginationParams` and return `PaginatedResponse<T>`
- For "fetch all" methods, use `limit=500` (or `100` for top-level resources)
- Types are defined in `src/lib/api/types.ts`, imported into client

## TypeScript Type Naming

Follow the CLAUDE.md DTO naming convention:

- **Request DTOs**: `{Resource}{Action}Request` (e.g., `SectionCreateRequest`)
- **Response types**: `{Resource}Response` or just the model name (e.g., `Section`)
- **DO NOT** use `Create{Resource}Request` pattern

## i18n Requirements

- **Both EN and DE translations are required** for every new feature
- Translation files: `src/i18n/messages/en.json` and `src/i18n/messages/de.json`
- Use `useTranslations()` hook, access keys like `t('sections.title')`
- Nav entries go under `nav.*` key
- Resource-specific keys go under `{resource}.*` (e.g., `sections.create`)
- Required keys for CRUD resources: `title`, `create`, `edit`, `deleteConfirm`, `deleteSuccess`, `createSuccess`, `updateSuccess`

## Unit Test Conventions

- **Location**: `__tests__/` folder next to the component
- **File naming**: `{component-name}.test.tsx`
- **Framework**: Jest + React Testing Library
- **Global mocks** (in `jest.setup.js`): `next-intl`, `next/navigation`, `window.matchMedia`, `ResizeObserver`
- **Translation mock**: `useTranslations` returns a function that passes through the key string
- **Pattern**: Render component, assert on visible text/structure, test interactions with `fireEvent`/`userEvent`
- Mock API client when needed: `jest.mock('@/lib/api/client')`

## E2E Test Conventions

- **Location**: `frontend/e2e/`
- **File naming**: `{feature}.spec.ts`
- **Locale**: Always use `test.use({ locale: 'en-US' })` for consistent text matching
- **API helpers**: Add helper functions to `e2e/utils/test-helpers.ts` for data setup/teardown
- **Authentication**: Use `login(page)` helper before each test
- **Data setup**: Create test data via API helpers, clean up after tests
- **Selectors**: Prefer `getByRole`, `getByLabel`, `getByText` over CSS selectors
- **Avoid date-dependent assertions** (status values like "Active"/"Upcoming")
- **Unique names**: Use `uniqueName('prefix')` for test data to avoid collisions

## Component Organization

- Shared UI primitives: `src/components/ui/` (shadcn)
- Reusable CRUD components: `src/components/crud/`
- Feature components: `src/components/{feature}/` (e.g., `src/components/sections/`)
- Layout components: `src/components/layout/`
- Pages: `src/app/(dashboard)/organizations/[orgId]/{feature}/page.tsx`
