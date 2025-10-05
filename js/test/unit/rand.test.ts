import { rng } from '../../src/utils/rand';

describe('Random number generator (rng)', () => {
  test('generates hex string of default length 16', () => {
    const result = rng();
    expect(result).toHaveLength(32); // 16 bytes = 32 hex chars
    expect(result).toMatch(/^[0-9a-f]+$/);
  });

  test('generates hex string of custom length', () => {
    const result = rng(8);
    expect(result).toHaveLength(16); // 8 bytes = 16 hex chars
    expect(result).toMatch(/^[0-9a-f]+$/);
  });

  test('generates hex string of length 32', () => {
    const result = rng(32);
    expect(result).toHaveLength(64); // 32 bytes = 64 hex chars
    expect(result).toMatch(/^[0-9a-f]+$/);
  });

  test('generates different values on each call', () => {
    const val1 = rng();
    const val2 = rng();
    expect(val1).not.toBe(val2);
  });

  test('handles zero length', () => {
    const result = rng(0);
    expect(result).toHaveLength(0);
    expect(result).toBe('');
  });
});
