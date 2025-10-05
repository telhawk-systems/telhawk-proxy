/**
 * @jest-environment node
 */

import { sign } from '../../src/transport/sign';

describe('HMAC signing', () => {
  let originalCrypto: Crypto;

  beforeEach(() => {
    originalCrypto = globalThis.crypto;
  });

  afterEach(() => {
    // Restore all mocks
    jest.restoreAllMocks();
    if (!originalCrypto) {
      delete (globalThis as any).crypto;
    }
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
    Object.defineProperty(globalThis, 'crypto', {
      value: undefined,
      configurable: true,
      writable: true
    });
    const result = await sign('test data', 'secret');
    expect(result).toBeNull();
    
    // Restore
    Object.defineProperty(globalThis, 'crypto', {
      value: originalCrypto,
      configurable: true,
      writable: true
    });
  });

  test('returns null when crypto.subtle is undefined', async () => {
    Object.defineProperty(globalThis, 'crypto', {
      value: {},
      configurable: true,
      writable: true
    });
    const result = await sign('test data', 'secret');
    expect(result).toBeNull();
    
    // Restore
    Object.defineProperty(globalThis, 'crypto', {
      value: originalCrypto,
      configurable: true,
      writable: true
    });
  });

  test('generates hex signature when crypto is available', async () => {
    const mockImportKey = jest.spyOn(globalThis.crypto.subtle, 'importKey')
      .mockResolvedValue('mock-key' as any);
    const mockSign = jest.spyOn(globalThis.crypto.subtle, 'sign')
      .mockResolvedValue(new Uint8Array([0x48, 0x65, 0x6c, 0x6c, 0x6f]).buffer);
    
    const result = await sign('test data', 'secret');
    
    expect(result).not.toBeNull();
    expect(result).toMatch(/^[0-9a-f]+$/);
    expect(result).toBe('48656c6c6f'); // "Hello" in hex
    
    mockImportKey.mockRestore();
    mockSign.mockRestore();
  });

  test('calls crypto.subtle.importKey with correct parameters', async () => {
    const mockImportKey = jest.spyOn(globalThis.crypto.subtle, 'importKey')
      .mockResolvedValue('mock-key' as any);
    const mockSign = jest.spyOn(globalThis.crypto.subtle, 'sign')
      .mockResolvedValue(new Uint8Array([0x48, 0x65, 0x6c, 0x6c, 0x6f]).buffer);
    
    await sign('test data', 'my-secret');
    
    expect(mockImportKey).toHaveBeenCalledWith(
      'raw',
      expect.any(Uint8Array),
      { name: 'HMAC', hash: 'SHA-256' },
      false,
      ['sign']
    );
    
    mockImportKey.mockRestore();
    mockSign.mockRestore();
  });

  test('calls crypto.subtle.sign with correct parameters', async () => {
    const mockImportKey = jest.spyOn(globalThis.crypto.subtle, 'importKey')
      .mockResolvedValue('mock-key' as any);
    const mockSign = jest.spyOn(globalThis.crypto.subtle, 'sign')
      .mockResolvedValue(new Uint8Array([0x48, 0x65, 0x6c, 0x6c, 0x6f]).buffer);
    
    await sign('test data', 'my-secret');
    
    expect(mockSign).toHaveBeenCalledWith(
      'HMAC',
      'mock-key',
      expect.any(Uint8Array)
    );
    
    mockImportKey.mockRestore();
    mockSign.mockRestore();
  });

  test('returns null on crypto error', async () => {
    const mockImportKey = jest.spyOn(globalThis.crypto.subtle, 'importKey')
      .mockRejectedValue(new Error('Crypto error'));
    
    const result = await sign('test data', 'secret');
    expect(result).toBeNull();
    
    mockImportKey.mockRestore();
  });

  test('returns null on sign error', async () => {
    const mockImportKey = jest.spyOn(globalThis.crypto.subtle, 'importKey')
      .mockResolvedValue('mock-key' as any);
    const mockSign = jest.spyOn(globalThis.crypto.subtle, 'sign')
      .mockRejectedValue(new Error('Sign error'));
    
    const result = await sign('test data', 'secret');
    expect(result).toBeNull();
    
    mockImportKey.mockRestore();
    mockSign.mockRestore();
  });

  test('handles different body content', async () => {
    const mockImportKey = jest.spyOn(globalThis.crypto.subtle, 'importKey')
      .mockResolvedValue('mock-key' as any);
    const mockSign = jest.spyOn(globalThis.crypto.subtle, 'sign')
      .mockResolvedValue(new Uint8Array([0x48, 0x65, 0x6c, 0x6c, 0x6f]).buffer);
    
    const result1 = await sign('{"event":"test"}', 'secret');
    const result2 = await sign('{"event":"other"}', 'secret');
    
    expect(result1).not.toBeNull();
    expect(result2).not.toBeNull();
    // Both should succeed (mocked)
    
    mockImportKey.mockRestore();
    mockSign.mockRestore();
  });
});
