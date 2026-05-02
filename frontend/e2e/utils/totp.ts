import * as OTPAuth from 'otpauth';

/**
 * generateTotp returns the current 6-digit code for the given base32
 * secret. Matches the backend's pquerna/otp defaults (SHA1, 6 digits,
 * 30-second window) so a code produced here is accepted by the same
 * verifier Google Authenticator would hit.
 *
 * Used in Playwright tests as the stand-in for the human reading
 * their authenticator app. Real e2e means real codes — never "000000".
 */
export function generateTotp(secret: string): string {
  const totp = new OTPAuth.TOTP({
    algorithm: 'SHA1',
    digits: 6,
    period: 30,
    secret: OTPAuth.Secret.fromBase32(secret),
  });
  return totp.generate();
}

/**
 * extractTotpSecret reads the base32 secret out of an otpauth:// URI
 * of the form `otpauth://totp/Issuer:Account?secret=JBSWY...&...`.
 * Used by tests that pull the secret off the enrolment response.
 */
export function extractTotpSecret(otpauthUri: string): string {
  const match = otpauthUri.match(/[?&]secret=([^&]+)/i);
  if (!match) {
    throw new Error(`no secret= param in ${otpauthUri}`);
  }
  return decodeURIComponent(match[1]);
}

/**
 * waitForNextTotpStep blocks until the wall-clock TOTP window
 * advances by one 30-second step. Required between operations that
 * bump the backend's `last_used_step` (TOTP activation, login verify,
 * step-up verify) and a subsequent operation that needs the SAME
 * secret to verify again — without it the second call would generate
 * the same per-window code, which the backend correctly rejects as a
 * replay. Closes the regression introduced by audit fix A-M-1
 * (activation now bumps last_used_step) for every E2E that activates
 * a factor and then immediately tries to verify the next code.
 *
 * Computed wait — instead of a flat `waitForTimeout(31000)` — keeps
 * CI fast: average ≈ 15 s, worst case ≈ 31 s. The 1-second cushion
 * past the boundary protects against clock skew on slow runners.
 *
 * The proper longer-term fix is a `SEED_TEST_DATA`-gated reset
 * endpoint that nukes `last_used_step` on demand; that work is
 * tracked in memory `project_e2e_totp_sleep_debt.md`.
 */
export async function waitForNextTotpStep(): Promise<void> {
  const periodMs = 30_000;
  const ms = periodMs - (Date.now() % periodMs) + 1000;
  await new Promise((resolve) => setTimeout(resolve, ms));
}
