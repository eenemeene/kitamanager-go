/**
 * Every form dialog is checked for the accessibility properties its fields are
 * supposed to have: labels that resolve, controls that have a name.
 *
 * These are all rendered open, because a dialog that is shut has no DOM to
 * inspect. Rendering them in one file rather than adding two assertions to each
 * component's own suite keeps the list of what is covered in one place — and
 * makes it obvious when a new dialog is not on it.
 *
 * See `src/test-a11y.ts` for why both assertions are needed and why neither
 * eslint nor axe alone catches the case that prompted them.
 */

import { useForm } from 'react-hook-form';
import { renderWithProviders } from '@/test-utils';
import { expectNoA11yViolations, expectNoOrphanLabels } from '@/test-a11y';

import { ChildCreateDialog } from '@/components/children/child-create-dialog';
import { ChildContractCreateDialog } from '@/components/children/child-contract-create-dialog';
import { PersonFormDialog, type PersonFormData } from '@/components/crud/person-form-dialog';
import { EmployeeContractDialog } from '@/components/employees/employee-contract-dialog';
import { PropertyFormDialog } from '@/components/government-funding-rates/property-form-dialog';
import type { EmployeeContractFormData } from '@/lib/schemas';

jest.mock('next-intl', () => ({
  useLocale: () => 'en',
  useTranslations: () => (key: string) => key,
}));

jest.mock('@/lib/hooks/use-funding-attributes', () => ({
  useFundingAttributes: () => ({
    fundingAttributes: [],
    attributesByKey: {},
    defaultProperties: {},
  }),
}));

const sections = [{ id: 1, name: 'Sonnengruppe' }] as never[];
const noop = () => {};

/** The two dialogs that take react-hook-form handles rather than owning them. */
function PersonFormHarness() {
  const { register, watch, setValue, formState } = useForm<PersonFormData>({
    defaultValues: { first_name: '', last_name: '', gender: 'male', birthdate: '' },
  });
  return (
    <PersonFormDialog
      open
      onOpenChange={noop}
      isEditing={false}
      register={register}
      onSubmit={noop}
      errors={formState.errors}
      watch={watch}
      setValue={setValue}
      isSaving={false}
      translationPrefix="children"
    />
  );
}

function EmployeeContractHarness() {
  const { register, watch, setValue, formState } = useForm<EmployeeContractFormData>();
  return (
    <EmployeeContractDialog
      open
      onOpenChange={noop}
      title="contracts.create"
      register={register}
      onSubmit={noop}
      errors={formState.errors}
      watch={watch}
      setValue={setValue}
      isSaving={false}
      payPlans={[]}
      sections={sections}
    />
  );
}

const dialogs: [name: string, render: () => React.ReactElement][] = [
  [
    'child create',
    () => (
      <ChildCreateDialog
        open
        onOpenChange={noop}
        orgId={1}
        orgState="berlin"
        sections={sections}
        isSaving={false}
        onSubmit={noop}
      />
    ),
  ],
  [
    'child contract create',
    () => (
      <ChildContractCreateDialog
        open
        onOpenChange={noop}
        orgId={1}
        orgState="berlin"
        child={{ id: 1, first_name: 'Mia', last_name: 'Beispiel' } as never}
        sections={sections}
        isSaving={false}
        onSubmit={noop}
      />
    ),
  ],
  ['person form', () => <PersonFormHarness />],
  ['employee contract', () => <EmployeeContractHarness />],
  [
    'funding property',
    () => <PropertyFormDialog open onOpenChange={noop} onSubmit={noop} isSaving={false} />,
  ],
];

describe.each(dialogs)('%s dialog', (_name, renderDialog) => {
  it('has no labels pointing at a missing element', () => {
    const { baseElement } = renderWithProviders(renderDialog());
    expectNoOrphanLabels(baseElement);
  });

  it('has no axe violations', async () => {
    const { baseElement } = renderWithProviders(renderDialog());
    await expectNoA11yViolations(baseElement);
  });
});
