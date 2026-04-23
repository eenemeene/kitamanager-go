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
