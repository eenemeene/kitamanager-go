import {
  base64urlToBuffer,
  bufferToBase64url,
  decodeCreationOptions,
  decodeRequestOptions,
  encodeRegistrationResponse,
  encodeAssertionResponse,
} from './webauthn';

// The server (go-webauthn) emits creation and request options nested
// under a top-level `publicKey` key. navigator.credentials.create()
// expects the inner shape, so decodeCreationOptions MUST unwrap the
// wrapper. Regression coverage for a real E2E-breaking bug where the
// dialog passed the wrapped object straight to the decoder and ran
// into `Cannot read property 'id' of undefined`.
describe('decodeCreationOptions', () => {
  const innerShape = {
    rp: { id: 'localhost', name: 'Kita' },
    user: { id: 'AAAAAAAAABw', name: 'u@e.com', displayName: 'u' },
    challenge: 'aopWi_XCarYV0bLLfL2D2hKQYpOX5lOBHupsKqwEY8s',
    pubKeyCredParams: [{ type: 'public-key' as const, alg: -7 }],
    timeout: 300000,
  };

  it('unwraps the go-webauthn publicKey wrapper', () => {
    const opts = decodeCreationOptions({ publicKey: innerShape });
    expect(opts.challenge).toBeInstanceOf(ArrayBuffer);
    expect(opts.user.id).toBeInstanceOf(ArrayBuffer);
    expect(opts.rp.id).toBe('localhost');
  });

  it('accepts a flat shape for forward-compatibility', () => {
    const opts = decodeCreationOptions(innerShape);
    expect(opts.challenge).toBeInstanceOf(ArrayBuffer);
    expect(opts.user.id).toBeInstanceOf(ArrayBuffer);
  });

  it('decodes excludeCredentials ids to ArrayBuffers', () => {
    const opts = decodeCreationOptions({
      publicKey: {
        ...innerShape,
        excludeCredentials: [
          { type: 'public-key', id: 'YWFh' },
          { type: 'public-key', id: 'YmJi' },
        ],
      },
    });
    expect(opts.excludeCredentials).toHaveLength(2);
    expect(opts.excludeCredentials?.[0].id).toBeInstanceOf(ArrayBuffer);
  });
});

describe('decodeRequestOptions', () => {
  const innerShape = {
    challenge: 'aopWi_XCarYV0bLLfL2D2hKQYpOX5lOBHupsKqwEY8s',
    rpId: 'localhost',
    timeout: 300000,
    allowCredentials: [{ type: 'public-key' as const, id: 'YWFh' }],
    userVerification: 'preferred' as const,
  };

  it('unwraps the publicKey wrapper', () => {
    const opts = decodeRequestOptions({ publicKey: innerShape });
    expect(opts.challenge).toBeInstanceOf(ArrayBuffer);
    expect(opts.allowCredentials?.[0].id).toBeInstanceOf(ArrayBuffer);
    expect(opts.rpId).toBe('localhost');
  });

  it('accepts a flat shape', () => {
    const opts = decodeRequestOptions(innerShape);
    expect(opts.challenge).toBeInstanceOf(ArrayBuffer);
  });
});

describe('base64url round-trip', () => {
  it('encodes + decodes arbitrary bytes losslessly', () => {
    const src = new Uint8Array([0, 1, 2, 255, 128, 64, 32, 16, 8, 4, 2, 1]);
    const encoded = bufferToBase64url(src);
    expect(encoded).not.toContain('=');
    expect(encoded).not.toContain('+');
    expect(encoded).not.toContain('/');
    const decoded = new Uint8Array(base64urlToBuffer(encoded));
    expect(Array.from(decoded)).toEqual(Array.from(src));
  });
});

describe('encodeRegistrationResponse / encodeAssertionResponse', () => {
  const fakeBuf = (b: number[]) => new Uint8Array(b).buffer;

  it('serialises an attestation response into base64url fields', () => {
    const cred = {
      id: 'cred-id',
      rawId: fakeBuf([1, 2, 3]),
      type: 'public-key',
      response: {
        attestationObject: fakeBuf([4, 5, 6]),
        clientDataJSON: fakeBuf([7, 8, 9]),
        getTransports: () => ['internal'],
      },
      getClientExtensionResults: () => ({}),
    } as unknown as PublicKeyCredential;
    const out = encodeRegistrationResponse(cred);
    expect(out.id).toBe('cred-id');
    expect(out.rawId).toBe(bufferToBase64url(fakeBuf([1, 2, 3])));
    const response = out.response as Record<string, unknown>;
    expect(response.attestationObject).toBe(bufferToBase64url(fakeBuf([4, 5, 6])));
    expect(response.transports).toEqual(['internal']);
  });

  it('serialises an assertion response and omits userHandle when null', () => {
    const cred = {
      id: 'cred-id',
      rawId: fakeBuf([1]),
      type: 'public-key',
      response: {
        authenticatorData: fakeBuf([2]),
        clientDataJSON: fakeBuf([3]),
        signature: fakeBuf([4]),
        userHandle: null,
      },
      getClientExtensionResults: () => ({}),
    } as unknown as PublicKeyCredential;
    const out = encodeAssertionResponse(cred);
    const response = out.response as Record<string, unknown>;
    expect(response.userHandle).toBeUndefined();
    expect(response.signature).toBe(bufferToBase64url(fakeBuf([4])));
  });
});
