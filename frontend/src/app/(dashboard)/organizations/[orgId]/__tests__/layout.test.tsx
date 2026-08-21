import { screen, waitFor } from '@testing-library/react';
import OrganizationScopedLayout from '../layout';
import { apiClient } from '@/lib/api/client';
import { useUiStore } from '@/stores/ui-store';
import { renderWithProviders } from '@/test-utils';

// Records the call rather than throwing. The real notFound() throws a sentinel
// the framework catches at the not-found boundary; reproducing that here would
// only test React's error handling, and what matters is whether the guard fired.
const notFound = jest.fn();
let params: Record<string, unknown> = { orgId: '1' };

jest.mock('next/navigation', () => ({
  useParams: () => params,
  notFound: () => notFound(),
}));

jest.mock('@/lib/api/client', () => ({
  apiClient: { getOrganization: jest.fn() },
}));

const getOrganization = apiClient.getOrganization as jest.Mock;

/** An axios-shaped rejection carrying a problem document. */
function problem(status: number) {
  return { response: { data: { status, title: 't', code: 'c', type: 'u' } } };
}

function renderLayout() {
  renderWithProviders(
    <OrganizationScopedLayout>
      <p>org content</p>
    </OrganizationScopedLayout>
  );
}

describe('OrganizationScopedLayout', () => {
  beforeEach(() => {
    notFound.mockClear();
    params = { orgId: '1' };
    getOrganization.mockReset();
  });

  it('renders the page for an organization that exists', async () => {
    getOrganization.mockResolvedValue({ id: 1, name: 'Kita Sonnenschein' });
    renderLayout();

    expect(await screen.findByText('org content')).toBeInTheDocument();
    expect(notFound).not.toHaveBeenCalled();
  });

  it('renders the page while the check is still in flight', async () => {
    // A valid organization -- nearly every load -- must not wait on this. The
    // children go out immediately and the guard catches up.
    getOrganization.mockReturnValue(new Promise(() => {}));
    renderLayout();

    expect(await screen.findByText('org content')).toBeInTheDocument();
    expect(notFound).not.toHaveBeenCalled();
  });

  it('is not found when the organization does not exist', async () => {
    getOrganization.mockRejectedValue(problem(404));
    renderLayout();

    await waitFor(() => expect(notFound).toHaveBeenCalled());
  });

  it('is not found when the id is not a number, without asking the server', async () => {
    params = { orgId: 'abc' };
    renderLayout();

    expect(notFound).toHaveBeenCalled();
    expect(getOrganization).not.toHaveBeenCalled();
  });

  it('leaves a server failure to the page, rather than claiming the org is gone', async () => {
    // The rule this guards: a 500, a timeout, or an offline laptop must not tell
    // somebody their Kita has been deleted.
    getOrganization.mockRejectedValue(problem(500));
    renderLayout();

    expect(await screen.findByText('org content')).toBeInTheDocument();
    await waitFor(() => expect(getOrganization).toHaveBeenCalled());
    expect(notFound).not.toHaveBeenCalled();
  });

  /**
   * The store has to follow the URL, or the sidebar, the organization selector
   * and -- worst -- the role that gates the whole UI all describe a different
   * organization from the one on screen.
   */
  describe('keeping the store on the URL organization', () => {
    beforeEach(() => {
      useUiStore.setState({ selectedOrganizationId: null, organizations: [] });
    });

    it('adopts the organization from the route', async () => {
      // The deep-link case: a bookmark or shared URL for an organization other
      // than the one left selected in localStorage.
      useUiStore.setState({ selectedOrganizationId: 3 });
      params = { orgId: '7' };
      getOrganization.mockResolvedValue({ id: 7, name: 'Kita Sonnenschein' });

      renderLayout();

      await waitFor(() => expect(useUiStore.getState().selectedOrganizationId).toBe(7));
    });

    it('leaves a matching selection alone', async () => {
      useUiStore.setState({ selectedOrganizationId: 1 });
      params = { orgId: '1' };
      getOrganization.mockResolvedValue({ id: 1, name: 'Kita Sonnenschein' });

      renderLayout();

      await waitFor(() => expect(screen.getByText('org content')).toBeInTheDocument());
      expect(useUiStore.getState().selectedOrganizationId).toBe(1);
    });

    it('does not clear a good selection for an unaddressable segment', async () => {
      // /organizations/abc/... is on its way to a 404. Wiping the stored
      // organization on the way would leave the user with no selection at all
      // once they navigated back.
      useUiStore.setState({ selectedOrganizationId: 3 });
      params = { orgId: 'abc' };

      renderLayout();

      await waitFor(() => expect(notFound).toHaveBeenCalled());
      expect(useUiStore.getState().selectedOrganizationId).toBe(3);
    });

    it('adopts the route organization before the existence check resolves', async () => {
      // The sidebar renders alongside these children, so waiting for the check
      // would leave it pointing at the wrong organization in the meantime.
      useUiStore.setState({ selectedOrganizationId: 3 });
      params = { orgId: '9' };
      getOrganization.mockReturnValue(new Promise(() => {}));

      renderLayout();

      await waitFor(() => expect(useUiStore.getState().selectedOrganizationId).toBe(9));
    });
  });
});
