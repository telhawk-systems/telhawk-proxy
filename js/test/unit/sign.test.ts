import { sign } from '../../src/transport/sign';

// Mock crypto.subtle for testing
const mockCrypto = () => {
  const encoder = new TextEncoder();
  
  return {
    subtle: {
      importKey: jest.fn().mockResolvedValue('mock-key'),
      sign: jest.fn().mockImplementation(async (algorithm, key, data) => {
        // Simple mock: just return the data as signature
        return new Uint8Array([0x48, 0x65, 0x6c, 0x6c, 0x6f]).buffer;
      })
    }
  };
};

describe('HMAC signing', () => {
  let originalCrypto: any;

  beforeEach(() => {
    originalCrypto = global.crypto;
  });

  afterEach(() => {
    global.crypto = originalCrypto;
  });

  test('returns null when no secret provided', async () => {
    const result = await sign('test data');
    expect(result).toBeNull();
  });

  test('returns null when secret is empty string', async () => {
    const result = await sign('test data', '');
    expect(result).toBeNull();
  });

  test('returns null when crypto is undefined', async () => {
    (global as any).crypto = undefined;
    const result = await sign('test data', 'secret');
    expect(result).toBeNull();
  });

  test('returns null when crypto.subtle is undefined', async () => {
    (global as any).crypto = {};
    const result = await sign('test data', 'secret');
    expect(result).toBeNull();
  });

  test('generates hex signature when crypto is available', async () => {
    global.crypto = mockCrypto() as any;
    
    const result = await sign('test data', 'secret');
    
    expect(result).not.toBeNull();
    expect(result).toMatch(/^[0-9a-f]+$/);
    expect(result).toBe('48656c6c6f'); // "Hello" in hex
  });

  test('calls crypto.subtle.importKey with correct parameters', async () => {
    const mock = mockCrypto();
    global.crypto = mock as any;
    
    await sign('test data', 'my-secret');
    
    expect(mock.subtle.importKey).toHaveBeenCalledWith(
      'raw',
      expect.any(Uint8Array),
      { name: 'HMAC', hash: 'SHA-256' },
      false,
      ['sign']
    );
  });

  test('calls crypto.subtle.sign with correct parameters', async () => {
    const mock = mockCrypto();
    global.crypto = mock as any;
    
    await sign('test data', 'my-secret');
    
    expect(mock.subtle.sign).toHaveBeenCalledWith(
      'HMAC',
      'mock-key',
      expect.any(Uint8Array)
    );
  });

  test('returns null on crypto error', async () => {
    const mock = mockCrypto();
    mock.subtle.importKey = jest.fn().mockRejectedValue(new Error('Crypto error'));
    global.crypto = mock as any;
    
    const result = await sign('test data', 'secret');
    expect(result).toBeNull();
  });

  test('returns null on sign error', async () => {
    const mock = mockCrypto();
    mock.subtle.sign = jest.fn().mockRejectedValue(new Error('Sign error'));
    global.crypto = mock as any;
    
    const result = await sign('test data', 'secret');
    expect(result).toBeNull();
  });

  test('handles different body content', async () => {
    global.crypto = mockCrypto() as any;
    
    const result1 = await sign('{"event":"test"}', 'secret');
    const result2 = await sign('{"event":"other"}', 'secret');
    
    expect(result1).not.toBeNull();
    expect(result2).not.toBeNull();
    // Both should succeed (mocked)
  });
});
