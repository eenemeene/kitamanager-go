/**
 * Thin WebAuthn helpers. The go-webauthn server emits
 * PublicKeyCredentialCreationOptionsJSON and
 * PublicKeyCredentialRequestOptionsJSON — both carry challenge /
 * user.id / allowCredentials[].id as base64url strings. The browser
 * WebAuthn API expects ArrayBuffers there. These helpers do the
 * conversion in a single place so each call site stays short.
 *
 * We deliberately avoid pulling in @simplewebauthn/browser: we need
 * exactly two operations (start registration + start authentication)
 * and the decoding itself is trivial.
 */

// base64urlToBuffer decodes a base64url (RFC 4648 §5) string into an
// ArrayBuffer. Accepts strings with or without padding; handles URL-
// safe `-` / `_` characters that WebAuthn JSON uses.
export function base64urlToBuffer(b64url: string): ArrayBuffer {
  const padded = b64url.replace(/-/g, '+').replace(/_/g, '/');
  const padding = '='.repeat((4 - (padded.length % 4)) % 4);
  const binary = atob(padded + padding);
  const buf = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) buf[i] = binary.charCodeAt(i);
  return buf.buffer;
}

// bufferToBase64url encodes an ArrayBuffer (or ArrayBufferView) into
// a base64url string without padding — the shape WebAuthn JSON uses.
export function bufferToBase64url(buf: ArrayBuffer | ArrayBufferView): string {
  const view =
    buf instanceof ArrayBuffer
      ? new Uint8Array(buf)
      : new Uint8Array(buf.buffer, buf.byteOffset, buf.byteLength);
  let binary = '';
  for (let i = 0; i < view.length; i++) binary += String.fromCharCode(view[i]);
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

// Shape of PublicKeyCredentialCreationOptionsJSON — we only decode
// the fields the browser needs as ArrayBuffers.
interface CreationOptionsJSON {
  challenge: string;
  user: { id: string; name: string; displayName: string };
  rp: { id: string; name: string };
  pubKeyCredParams: PublicKeyCredentialParameters[];
  timeout?: number;
  excludeCredentials?: Array<{
    id: string;
    type: 'public-key';
    transports?: AuthenticatorTransport[];
  }>;
  authenticatorSelection?: AuthenticatorSelectionCriteria;
  attestation?: AttestationConveyancePreference;
  extensions?: Record<string, unknown>;
}

interface RequestOptionsJSON {
  challenge: string;
  rpId?: string;
  timeout?: number;
  allowCredentials?: Array<{
    id: string;
    type: 'public-key';
    transports?: AuthenticatorTransport[];
  }>;
  userVerification?: UserVerificationRequirement;
  extensions?: Record<string, unknown>;
}

// decodeCreationOptions turns the server JSON into the object
// `navigator.credentials.create({publicKey: ...})` expects — namely
// with ArrayBuffers in place of base64url strings.
//
// go-webauthn emits the options nested under a top-level `publicKey`
// key to match the browser shape; unwrap if present so callers can
// pass the server response directly without having to know the shape.
export function decodeCreationOptions(
  opts: CreationOptionsJSON | { publicKey: CreationOptionsJSON }
): PublicKeyCredentialCreationOptions {
  const inner = 'publicKey' in opts ? opts.publicKey : opts;
  return {
    ...inner,
    challenge: base64urlToBuffer(inner.challenge),
    user: {
      ...inner.user,
      id: base64urlToBuffer(inner.user.id),
    },
    excludeCredentials: inner.excludeCredentials?.map((c) => ({
      ...c,
      id: base64urlToBuffer(c.id),
    })),
  };
}

// decodeRequestOptions turns the server JSON into the object
// `navigator.credentials.get({publicKey: ...})` expects. Unwraps the
// optional top-level `publicKey` wrapper for the same reason as
// decodeCreationOptions.
export function decodeRequestOptions(
  opts: RequestOptionsJSON | { publicKey: RequestOptionsJSON }
): PublicKeyCredentialRequestOptions {
  const inner = 'publicKey' in opts ? opts.publicKey : opts;
  return {
    ...inner,
    challenge: base64urlToBuffer(inner.challenge),
    allowCredentials: inner.allowCredentials?.map((c) => ({
      ...c,
      id: base64urlToBuffer(c.id),
    })),
  };
}

// encodeRegistrationResponse serialises the PublicKeyCredential
// returned by `navigator.credentials.create()` into the JSON shape
// the go-webauthn library expects on the wire.
export function encodeRegistrationResponse(cred: PublicKeyCredential): Record<string, unknown> {
  const response = cred.response as AuthenticatorAttestationResponse;
  return {
    id: cred.id,
    rawId: bufferToBase64url(cred.rawId),
    type: cred.type,
    response: {
      attestationObject: bufferToBase64url(response.attestationObject),
      clientDataJSON: bufferToBase64url(response.clientDataJSON),
      // transports come from the authenticator via
      // getTransports() — optional but helpful for allowCredentials[]
      // on subsequent logins.
      transports: response.getTransports ? response.getTransports() : undefined,
    },
    clientExtensionResults: cred.getClientExtensionResults(),
  };
}

// encodeAssertionResponse serialises the PublicKeyCredential returned
// by `navigator.credentials.get()` for wire transport.
export function encodeAssertionResponse(cred: PublicKeyCredential): Record<string, unknown> {
  const response = cred.response as AuthenticatorAssertionResponse;
  return {
    id: cred.id,
    rawId: bufferToBase64url(cred.rawId),
    type: cred.type,
    response: {
      authenticatorData: bufferToBase64url(response.authenticatorData),
      clientDataJSON: bufferToBase64url(response.clientDataJSON),
      signature: bufferToBase64url(response.signature),
      userHandle: response.userHandle ? bufferToBase64url(response.userHandle) : undefined,
    },
    clientExtensionResults: cred.getClientExtensionResults(),
  };
}

/**
 * isWebAuthnSupported returns true iff the browser exposes the
 * PublicKeyCredential API. Component code uses this to hide the
 * "Add security key" button on unsupported browsers.
 */
export function isWebAuthnSupported(): boolean {
  return typeof window !== 'undefined' && typeof window.PublicKeyCredential !== 'undefined';
}
