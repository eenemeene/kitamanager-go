import { getErrorMessage } from '../client';

describe('getErrorMessage', () => {
  it('extracts message from axios error response', () => {
    const error = {
      response: {
        data: {
          message: 'Invalid credentials',
        },
      },
    };

    expect(getErrorMessage(error, 'Fallback message')).toBe('Invalid credentials');
  });

  it('returns fallback for error without response', () => {
    const error = new Error('Network error');

    expect(getErrorMessage(error, 'Fallback message')).toBe('Fallback message');
  });

  it('returns fallback for error without message in response', () => {
    const error = {
      response: {
        data: {},
      },
    };

    expect(getErrorMessage(error, 'Fallback message')).toBe('Fallback message');
  });

  it('returns fallback for null error', () => {
    expect(getErrorMessage(null, 'Fallback message')).toBe('Fallback message');
  });

  it('returns fallback for undefined error', () => {
    expect(getErrorMessage(undefined, 'Fallback message')).toBe('Fallback message');
  });

  it('returns fallback for non-object error', () => {
    expect(getErrorMessage('string error', 'Fallback message')).toBe('Fallback message');
    expect(getErrorMessage(123, 'Fallback message')).toBe('Fallback message');
  });
});
