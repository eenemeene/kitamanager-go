import type { CDPSession, Page, BrowserContext } from '@playwright/test';

/**
 * WebAuthn virtual-authenticator helpers for Playwright. Chromium's
 * CDP domain `WebAuthn` lets us attach an in-memory authenticator
 * that generates real keys, signs real attestations, and handles
 * both registration and authentication ceremonies transparently to
 * the page. We don't need to fake the wire protocol at any layer —
 * the browser does the crypto, the server verifies it for real.
 *
 * Playwright hasn't shipped a first-class wrapper (feature request
 * microsoft/playwright#7276 is still open), so we drive CDP directly.
 */

export interface VirtualAuthenticator {
  authenticatorId: string;
  cdp: CDPSession;
}

/**
 * Attach an internal-transport virtual authenticator to the current
 * browser context. `userVerified` controls whether the authenticator
 * auto-confirms the UV gesture (Touch ID / PIN prompt); set false in
 * tests that need to exercise the NotAllowedError path.
 *
 * Returns a handle the test uses to later remove the authenticator or
 * inspect its credentials. Call removeVirtualAuthenticator in
 * afterEach so state doesn't bleed across tests.
 */
export async function addVirtualAuthenticator(
  context: BrowserContext,
  page: Page,
  opts: {
    transport?: 'internal' | 'usb';
    hasUserVerification?: boolean;
    isUserVerified?: boolean;
  } = {}
): Promise<VirtualAuthenticator> {
  const cdp = await context.newCDPSession(page);
  await cdp.send('WebAuthn.enable');
  const result = (await cdp.send('WebAuthn.addVirtualAuthenticator', {
    options: {
      protocol: 'ctap2',
      transport: opts.transport ?? 'internal',
      hasResidentKey: false,
      hasUserVerification: opts.hasUserVerification ?? true,
      isUserVerified: opts.isUserVerified ?? true,
      automaticPresenceSimulation: true,
    },
  })) as { authenticatorId: string };
  return { authenticatorId: result.authenticatorId, cdp };
}

/**
 * Removes the virtual authenticator attached earlier. Safe to call
 * even if the session has already gone away (Playwright destroys
 * CDP sessions on page close).
 */
export async function removeVirtualAuthenticator(auth: VirtualAuthenticator): Promise<void> {
  try {
    await auth.cdp.send('WebAuthn.removeVirtualAuthenticator', {
      authenticatorId: auth.authenticatorId,
    });
  } catch {
    // Session already closed — ignore.
  }
}

/**
 * Switches the virtual authenticator's UV gesture from auto-confirm
 * to auto-deny. Useful for testing the NotAllowedError code path on
 * the login ceremony.
 */
export async function setUserVerified(
  auth: VirtualAuthenticator,
  verified: boolean
): Promise<void> {
  await auth.cdp.send('WebAuthn.setUserVerified', {
    authenticatorId: auth.authenticatorId,
    isUserVerified: verified,
  });
}
